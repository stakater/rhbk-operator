# Realm Key Preservation and Rotation

Design for keeping realm signing keys stable across `KeycloakImport` reconciles, and for
rotating them deliberately when that is actually wanted.

Status: proposed. One assumption still needs an end-to-end check — see
[Open assumption](#open-assumption).

## Problem

Importing a realm with `spec.overrideIfExists: true` changes the realm's signing key, and
therefore the `kid` advertised in its JWKS. Every consumer that had cached the old JWKS
starts rejecting tokens.

The import path is an offline `kc.sh import` job built from a copy of the Keycloak
StatefulSet pod template (`internal/resources/realm/realm_import_job.go`). With
`--override=true`, Keycloak's `OVERWRITE_EXISTING` strategy **deletes the realm and
recreates it**. Realm keys live as `components` of type
`org.keycloak.keys.KeyProvider` — `rsa-generated`, `rsa-enc-generated`, `hmac-generated`,
`aes-generated`. A fresh realm gets fresh providers, so fresh RSA keypairs. Since the `kid`
is a thumbprint of the public key, a new key means a new `kid`.

The same event also discards users, consents, and sessions. That is out of scope here but
worth knowing when choosing `overrideIfExists`.

Two independent contributors:

1. **The realm JSON carries no key components.** An admin-console *partial* export strips
   `components`; a full `kc.sh export` includes them with `privateKey`/`certificate`. Realm
   JSON maintained by hand from a partial export therefore regenerates keys on every import.

2. **The operator re-imports more often than it needs to.** Job identity is
   `importSecret.Resource.ResourceVersion`
   (`internal/controller/keycloakimport_controller.go:152`). That is not content-derived, so
   deleting the secret by hand, recreating the CR, or a finalizer cleanup all mint a new
   resourceVersion and trigger another destructive override.

### Why this surfaces as an outage

The reported symptom comes from Istio `RequestAuthentication`. Istiod learns keys only by
polling `jwks_uri` on a timer — `PILOT_JWT_PUB_KEY_REFRESH_INTERVAL`, 20 minutes by default
— and does not refetch when it sees an unfamiliar `kid`:

- [istio/istio#49044](https://github.com/istio/istio/issues/49044) — feature request for
  refresh-on-unknown-kid. Closed as stale.
- [envoyproxy/envoy#14557](https://github.com/envoyproxy/envoy/issues/14557) — the
  underlying Envoy gap. Closed as completed, but the final comment (Feb 2025) reports the
  fix was reverted. Do not rely on it without testing your build.

So the failure window is not a race, it is deterministic:

```
 Keycloak   abc123 active ─────────┤ realm recreated ├─── xyz789 active
 Istio has  {abc123} ──────────────────────────────────── {abc123}  …up to 20 min…
 Tokens                                                    signed xyz789 → 401
```

The new `kid` goes from nonexistent to signing everything in one instant. There is no window
in which Istio could have learned it, so it fails every time.

### Who is non-compliant

OpenID Connect Core 1.0 §10.1.1 puts the burden on the verifier: the signer "can begin using
a new key at its discretion and signals the change to the verifier using the `kid` value",
and "the verifier knows to go back to the `jwks_uri` location to re-retrieve the keys when it
sees an unfamiliar `kid` value."

Keycloak is compliant. Istio is not. This design is a workaround for a non-compliant
verifier, not a fix for a Keycloak defect — but §10.1.1 also says rotation happens at the
signer's *discretion*, which implies intent. Rotating on every config sync is not a rotation
policy, it is a side effect. That is the part worth fixing regardless of Istio.

## Rejected alternatives

**Restore the old key as a passive secondary after import.** Keeps old tokens verifiable,
which is what §10.1.1 recommends of a signer — but it does not address this failure. Istio's
problem is that the *new* `kid` is unknown, not that the old one vanished. Nothing is being
signed with the retained key, and Istio is reading its cached copy regardless. No effect on
the outage.

**Change auth mechanism.** App-level validation (Nimbus, `node-jwks-rsa`, `keyfunc`),
ext_authz, and token introspection all avoid the bug, because they either implement
refetch-on-unknown-kid or never touch JWKS. But it only fixes the consumer that migrates:
other gateways, partner integrations, and S2S clients inherit the problem, long-lived tokens
break under any verifier, and introspection trades the issue for per-request latency and a
hard runtime dependency on Keycloak. Re-architecting mesh auth around an operator that
rotates keys it had no reason to rotate is the larger and riskier change.

**Operator-managed inline JWKS in `RequestAuthentication`.** Mirroring
`/realms/{realm}/protocol/openid-connect/certs` into `jwtRules[].jwks` propagates via xDS in
seconds instead of 20 minutes. Rejected: it needs cluster-wide write on
`security.istio.io/RequestAuthentication` to reach tenant namespaces, makes the operator a
single point of failure for all mesh auth, fights GitOps in every namespace at once, and
still fixes only Istio. The rotation design below achieves safe propagation without it.

**Operator-generated pinned keys.** Works, but changes the `kid` once at adoption and puts
key generation in the operator when Keycloak already has a perfectly good key sitting there.

## Design

### 1. Verbatim restore

Before each import, capture the realm's `KeyProvider` components and inject them back into
the rendered realm JSON **verbatim** — same `providerId`, same config, same priority, same
active flag.

Keycloak's default-key bootstrap checks whether a component with `providerId ==
"rsa-generated"` already exists (likewise `rsa-enc-generated`, `hmac-generated`,
`aes-generated`). Restoring verbatim satisfies that check, so Keycloak skips generation and
loads the key straight out of the imported component config — which is already how the key
survives pod restarts. Same key, same `kid`, indefinitely.

This is what a full `kc.sh export` would have given you, automated, so hand-maintained realm
JSON stays free of key material.

```
 Reconcile — content hash of spec.json + resolved substitutions changed
      │
      ├─ 1. realm exists?  ── no ──► skip to 4 (Keycloak generates the first key)
      │
      ├─ 2. FETCH  GET /admin/realms/{realm}/components
      │              ?type=org.keycloak.keys.KeyProvider
      │
      ├─ 3. ARCHIVE to Secret <cr>-realm-keys   { capturedAt, components[] }
      │      ✗ on failure → failed condition, STOP. No import without a backup.
      │
      ├─ 4. RENDER  spec.json + substitutions + archived components, VERBATIM
      │      → import Secret
      │
      ├─ 5. JOB  kc.sh import --override=true
      │        └─ providers already present → bootstrap skips generation → same kid
      │
      └─ 6. sts annotation → rolling restart
```

Ordering is the safety property. A crash between "realm deleted" and "key persisted" loses
the key irrecoverably, so the archive write must complete before the job is created, and a
failed capture must block the import rather than proceed without it.

Substitution (`%`-delimited templating in `expandTemplate`) runs before component injection,
so JSON manipulation does not collide with it.

### 2. Content-hash job identity

Replace `importSecret.Resource.ResourceVersion` with a truncated SHA-256 of `spec.json` plus
resolved substitutions, as job identity, StatefulSet annotation value, and a `status` field.
Imports then fire only on genuine content change and are idempotent across CR recreation,
secret deletion, and operator restarts.

**The hash must not include the injected key components.** They change whenever keys change,
which would change the hash, which would trigger another import — a permanent rotation loop.

### 3. Deliberate rotation

Rotation is a separate concern from importing, on its own clock. Three slots, one threshold:

```
 created ──rotateAfter──► promoted ──rotateAfter──► demoted ──rotateAfter──► removed
         (next)                     (active)                 (previous)
         passive                    signing                  passive
```

Keycloak's native key states map directly: **Active** (signs, published), **Passive**
(published, does not sign), **Disabled** (not published). Passive is exactly the "known but
not yet used" state required.

Because every key is published for a full `rotateAfter` before it ever signs, the
pre-distribution window is satisfied by construction — Istio has had the whole period to
poll. Because it stays published for a full `rotateAfter` after it stops signing, retention
for in-flight tokens is also satisfied. Both windows fall out of one number.

```
if now.Sub(status.lastRotation) > rotateAfter {
    promote(next)       // next     → active   (priority 100 + active=true)
    demote(active)      // active   → passive, priority 80
    remove(previous)    // previous → gone
    create()            // new next, passive, priority 90
    status.lastRotation = now
}
```

Keycloak selects the highest-priority *active* key for signing, so raising the incoming key's
priority makes the flip take effect immediately and demoting the outgoing one afterwards is a
no-op for signing. No instant with zero or ambiguous active keys.

Removal lags one full cycle behind demotion. Dropping the outgoing key at flip time would
invalidate every token it signed in the preceding minutes; lagging costs one extra JWKS entry.

Evaluated lazily, only on reconciles that happen anyway. This is safe because every timer is
a lower bound — late evaluation lets a key serve slightly longer or stay published slightly
longer, and neither causes an outage. Premature promotion and under-retention are the
dangerous directions, and lazy evaluation cannot produce either. The check is idempotent: two
cycles overdue rotates once, and the next reconcile rotates again.

## API

```yaml
spec:
  realmKeys:                # presence = preserve keys verbatim across imports
    rotateAfter: 2160h      # optional; omit to pin forever and never rotate
```

Default (`realmKeys: {}`) is the safe behaviour: keys survive imports, nothing ever changes.
Omitting `realmKeys` entirely preserves today's behaviour.

`rotateAfter` is an **eligibility floor, not a schedule**. Rotation happens on the first
reconcile after the threshold, which may be later. It is deliberately not named `maxAge`:
with lazy evaluation there is no ceiling to guarantee.

Note that total key lifetime is **3× `rotateAfter`** — one period as `next`, one as `active`,
one as `previous`. `rotateAfter: 2160h` means each key exists for 270 days.

Single validation constraint:

```
rotateAfter > max(slowest verifier refresh interval, longest token lifetime)
```

Istio's default is 20 minutes; access tokens are minutes. Anything above ~1h is safe and a
90-day policy clears it by orders of magnitude. Worth validating anyway — the failure mode of
`rotateAfter: 5m` is silent and reintroduces the original outage.

### State

One timestamp, in status:

```yaml
status:
  lastRotation: 2026-04-28T09:14:00Z
```

Since all three slot transitions happen in the same operation, the active key's time in
service is exactly `now - lastRotation`. No per-key timestamps are needed.

Note this is **time in service**, not creation time. After a flip the incoming key was created
a full `rotateAfter` ago but has been active for zero seconds; keying off creation time would
make it instantly eligible again and rotate on every reconcile until the keys ran out.
`lastRotation` cannot express that mistake.

Slot identity comes from priority, which is already load-bearing because Keycloak signs with
the highest-priority active key:

| Slot | Priority | Active |
|---|---|---|
| `active` | 100 | yes |
| `next` | 90 | no |
| `previous` | 80 | no |

Priorities stay fixed per slot, so each rotation moves keys between them rather than
recomputing anything. Promote the highest-priority passive key, remove the lowest.

If `lastRotation` is ever lost — CR recreated, status wiped — treat it as bootstrap and stamp
it fresh. That delays the next rotation rather than triggering one, which is the safe
direction.

## Implementation

| Area | Change |
|---|---|
| `api/v1alpha1/keycloakimport_types.go` | `RealmKeys *RealmKeys` on the spec; `lastRotation` on status |
| `internal/keycloak/admin.go` (new) | Minimal admin client: token against `master`, components GET/PUT/DELETE |
| `internal/resources/realm/keys.go` (new) | Archive Secret, component rendering, rotation state machine |
| `internal/resources/realm/realm_import_secret.go` | Inject components in `MutateFn`, after `expandTemplate` |
| `internal/controller/keycloakimport_controller.go` | Capture before job creation; content-hash job identity; rotation check |

Admin credentials come from the Keycloak CR (`api/v1alpha1/keycloak_types.go:49`), reached
over the in-cluster Service. RBAC is unchanged — secrets and jobs in the target namespace,
nothing cluster-wide.

The admin client serves both capture and rotation, but the two triggers stay decoupled:
imports fire on config change, rotation on the key's own clock. Coupling them is what caused
the original problem.

## Open assumption

Everything above rests on one behaviour that needs an end-to-end check before implementation:

> A `KeyProvider` component with `providerId: "rsa-generated"` carrying an existing
> `privateKey`/`certificate` in its config suppresses Keycloak's default key generation **and**
> loads that key.

Confidence is high on the `providerId`-keyed bootstrap check, and the key-loading half is how
persistence across restarts already works — but it is internal behaviour and should be
verified against the RHBK version in use rather than assumed.

Secondary: whether preserving the component `id` matters for a clean restore, or whether
Keycloak assigns a fresh one harmlessly.

## Trade-offs

**Pinning means keys do not rotate unless asked.** A compromised key stays valid until
someone intervenes. That is the right default — unannounced rotation on every config sync is
not a security feature — but it makes `rotateAfter` the mechanism for anyone who wants
scheduled rotation, and manual intervention the answer for compromise.

**Refetch-on-unknown-kid would not fix revocation anyway.** Pulling a compromised key from
the JWK Set produces no unfamiliar `kid`, so no verifier refetches; revocation latency is
bounded by cache TTL regardless of compliance. Different problem, different lever.

**Offline tokens change the retention arithmetic.** If they are signed with the realm RSA key,
their lifetime can exceed `rotateAfter`, and removing `previous` invalidates live grants.
Confirm which artifacts are signed with the RSA key versus the realm HMAC key before settling
on a value.
