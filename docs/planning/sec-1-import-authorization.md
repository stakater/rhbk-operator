# SEC.1 — KeycloakImport cross-namespace authorization

Story: `docs/roadmap.md` SEC.1 (👍 Approved 2026-11-03). Driver: issue #4
(cross-tenant realm takeover via cross-namespace `KeycloakImport`).

## Why (one paragraph)

Any namespace can create a `KeycloakImport` targeting any `Keycloak` instance
in the cluster; with `overrideIfExists: true` it silently replaces another
team's realm — clients, redirect URIs, identity providers. In the per-team
model this is a cross-tenant takeover path. But cross-namespace import is also
today's *legitimate* usage (teams keep realm CRs in their own namespaces,
instances live elsewhere), so an outright same-namespace-only rule would break
the current 5-team deployment with no migration path.

## Decision

**Target-side opt-in allowlist, secure by default** (issue #4 option 2):

- Imports from the instance's **own namespace are always allowed**.
- Cross-namespace imports are allowed **only** if the target `Keycloak` lists
  the source namespace in `spec.realmImportPolicy.allowedNamespaces`
  (`"*"` = any namespace).
- No policy ⇒ same-namespace only. **This is a breaking change** for existing
  cross-namespace imports; migration is one field on the instance CR, owned by
  the instance owner — which is the point.

## Governing constraints

- CLAUDE.md invariant 2 (tenant isolation is load-bearing) — the check lives
  on the *target's* spec, so only the instance owner can widen it.
- CLAUDE.md invariant 4 (honest status) — a denied import reports Ready=False
  with an actionable message; it is not an error loop (no requeue storm).
- Invariant 5 (no silent scope-widening) — CRD field + controller gate only;
  CEL/webhook enforcement stays SEC.8 (#12).

## Design

- `api/v1alpha1/keycloak_types.go`: add `RealmImportPolicy` struct
  (`AllowedNamespaces []string`) + optional `Spec.RealmImportPolicy`; method
  `(*Keycloak).AllowsImportFrom(namespace string) bool`.
- `internal/controller/keycloakimport_controller.go`: after fetching the
  target instance (and before readiness checks), deny when
  `!keycloak.AllowsImportFrom(cr.Namespace)` via `HandleError` with message
  `cross-namespace import denied: ...`.
- `make manifests generate` for CRD YAML + deepcopy.
- e2e fixtures (`rhbk.yaml`, `rhbk_sized.yaml`) gain
  `realmImportPolicy.allowedNamespaces: [{{ .realmNamespace }}]`; both e2e
  tests pass `realmNamespace`. Sample `Keycloak` shows the knob commented.
- NOT changing: `KeycloakInstance.Namespace` field stays (no API break for
  same-namespace users), job/secret flow, RBAC, watches.

## Tests (red first)

- `api/v1alpha1/keycloak_types_test.go` (new): `AllowsImportFrom` table —
  same namespace (no policy), other namespace (no policy), listed, unlisted,
  wildcard, empty list.
- `internal/controller/keycloakimport_controller_test.go`:
  - new It: cross-namespace import with **no policy** → Ready=False, denial
    message, **no import secret created** in the target namespace (red: today
    the secret is created).
  - existing cross-namespace Its: target CR gains
    `realmImportPolicy.allowedNamespaces: [rhbk-import]` — they stay green and
    become the allow-path coverage.

## Out of scope

CEL/webhook admission enforcement (#12); label-selector form of the policy
(add if a real need appears); NetworkPolicy isolation (#7); e2e run on a live
cluster (no cluster in this environment; covered when #21 wires e2e).

## Done means

New unit + envtest denial tests red before, green after; existing import
tests green with the opt-in; `make test` green, manifests/deepcopy no-diff
after regen; samples + fixtures updated; roadmap SEC.1 ✅; closes #4.
Release note: existing cross-namespace imports require adding
`realmImportPolicy.allowedNamespaces` to the target instance CR.
