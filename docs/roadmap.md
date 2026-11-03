# rhbk-operator — Delivery Roadmap

The delivery view: epics → user stories, where every story delivers tangible,
demo-able output. The method behind this file — the loop, the lifecycle gate,
the planning-doc contract — is codified in [method.md](method.md).

## How we work

- **Story format:** *As a `<role>`, I want `<capability>`, so that `<value>`* —
  with a **Done:** acceptance line. Stories are coded (`SEC.1`, `TEN.1`, …).
- **Story lifecycle:** **📝 Draft** (written, not agreed — may be re-scoped or
  killed) → **👍 Approved** (greenlit) → **🔨 In progress** → **✅ Done** (Done
  line met, `make test` + `make lint` green, planning doc linked). Nothing moves
  from Draft to In-progress without explicit approval.
- **One story at a time**, end-to-end, then demoed — no big horizontal layers.
- Security findings live as labeled GitHub issues; a story references the
  issue(s) it closes.

## Vision (one paragraph)

Give every team their **own RHBK instance** — own namespace, own realms, own
`spec.version`, own upgrade schedule — managed by one cluster-scoped operator,
with isolation strong enough that no team can touch another team's instance.
The operator automates the OpenShift toil (TLS, Routes, monitoring, realm
import, providers, sizing) that makes self-service Keycloak otherwise
impractical. Upgrade independence without isolation is a liability; the SEC
epic is what makes the TEN epic safe to offer.

## Epic TEN — per-team instances

| Story | Status |
|---|---|
| **TEN.1** — As a platform engineer, I want each team to run its own `Keycloak` CR in its own namespace with an independently chosen `spec.version`, so that teams upgrade on their own schedule without coordinating. **Done:** tenancy model documented (namespace-per-team, DB-per-instance, resource expectations); two instances on different versions upgrade independently in e2e; cross-tenant isolation stories (SEC.1) referenced as prerequisites. | 📝 Draft |

## Epic SEC — security hardening

| Story | Status |
|---|---|
| **SEC.1** — As a team admin, I want `KeycloakImport` to be unable to target a `Keycloak` instance in another namespace (or only via explicit opt-in on the target CR), so that no other team can create or overwrite realms in my instance. **Done:** cross-namespace import is rejected (or gated by target-side allowlist); envtest proves the denial; existing same-namespace flow untouched. | 📝 Draft |
| **SEC.2** — As a platform engineer, I want provider JARs verified against a required checksum before they reach the Keycloak classpath, so that a compromised download URL cannot inject code into our IdP. **Done:** `spec.providers[].sha256` required; init container verifies and fails closed; test covers mismatch. | 📝 Draft |
| **SEC.3** — As a platform engineer, I want the RHBK StatefulSet to set an explicit restricted securityContext, so that pods pass `restricted-v2`/PSA without relying on SCC defaults. **Done:** runAsNonRoot, seccompProfile, drop ALL capabilities, no privilege escalation set on pod + containers; e2e green on OpenShift. | 📝 Draft |
| **SEC.4** — As a team admin, I want the operator to create NetworkPolicies per instance (JGroups only pod-to-pod, HTTPS from router, management from monitoring), so that other workloads cannot reach my instance's cache cluster. **Done:** policies created/owned by the CR; cluster traffic verified in e2e; cache mTLS evaluated and decided (ADR). | 📝 Draft |
| **SEC.5** — As a cluster admin, I want the operator's ClusterRole narrowed to what it actually needs, so that a compromised operator pod has the smallest possible blast radius. **Done:** verbs audited per resource; unneeded ones dropped; trust boundary documented. | 📝 Draft |
| **SEC.6** — As a team admin, I want to pin my instance's image by digest, so that an upgrade is always an explicit CR change and never a silent re-pull of a moved tag. **Done:** digest form supported in `spec.version` (or `spec.image`); `PullAlways` + mutable-tag default revisited; docs updated. | 📝 Draft |
| **SEC.7** — As a security engineer, I want admin and DB credentials to be accepted only as Secret references, so that no plaintext credential ever lands in a CR, Git repo, or StatefulSet spec. **Done:** CRD validation rejects inline `value` for password fields; migration note for existing CRs. | 📝 Draft |
| **SEC.8** — As a platform engineer, I want CRD-level validation (CEL or webhook) for version strings, instances, and reference shapes, so that bad CRs fail at admission instead of at reconcile. **Done:** invalid CRs rejected with actionable messages; envtest coverage. | 📝 Draft |
| **SEC.9** — As a security engineer, I want production logging hygiene (no `--verbose` defaults, no development logger, secured metrics), so that logs and endpoints don't leak more than they must. **Done:** verbose flags opt-in; zap production config; metrics served securely by default. | 📝 Draft |

## Epic FIX — correctness

| Story | Status |
|---|---|
| **FIX.1** — As an operator of the operator, I want `HasSecretReference` to tolerate substitutions without a Secret ref, so that one malformed `KeycloakImport` cannot panic the controller for the whole cluster. **Done:** nil-safe; regression test with value-only substitution; fuzz-ish table over substitution shapes. | 📝 Draft |

## Epic DX — developer experience

| Story | Status |
|---|---|
| **DX.1** — As a contributor, I want a `make verify` umbrella target (fmt · vet · build · lint · test) and an edit-hook that runs it, so that "done but unverified" cannot exist. **Done:** target exists, CI uses it, hook wired in `hack/`. | 📝 Draft |
| **DX.2** — As a new user, I want the README to describe what this operator actually is (not kubebuilder scaffolding), so that teams can evaluate and adopt it. **Done:** README covers purpose, comparison to official RHBK operator, CR examples, install. | 📝 Draft |
