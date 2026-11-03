# CLAUDE.md

Guidance for working in this repository.

## What this is

**rhbk-operator** — Stakater's opinionated operator for running **Red Hat Build of
Keycloak (RHBK)** on OpenShift. It deploys the official RHBK image
(`registry.redhat.io/rhbk/keycloak-rhel9`) and automates the OpenShift-specific toil
the official Red Hat operator leaves to the user: TLS via service-CA serving certs,
Route creation, ServiceMonitor, declarative realm import with secret substitution,
provider/SPI loading, and load-based sizing.

- **CRDs:** `Keycloak` and `KeycloakImport`, group `sso.stakater.com/v1alpha1`
- **Platform:** OpenShift-only by design (Routes, service-CA, SCC)
- **Stack:** Go, kubebuilder/controller-runtime, envtest, OLM bundle + FBC catalog

**Product direction** (Draft stories in `docs/roadmap.md`): one RHBK instance **per
team namespace**, each team controlling its own version/upgrades without
coordination — which makes cross-tenant isolation a first-class property, not an
afterthought.

## Repo map

| Path | What lives there |
|---|---|
| `api/v1alpha1/` | CRD Go types (`keycloak_types.go`, `keycloakimport_types.go`, `sizing.go`) |
| `internal/controller/` | The two reconcilers (`keycloak_controller.go`, `keycloakimport_controller.go`) |
| `internal/resources/rhbk/` | StatefulSet, Service, discovery (JGroups) Service, Route builders |
| `internal/resources/realm/` | Realm import secret + job, provider-download init container |
| `internal/resources/monitoring/` | ServiceMonitor |
| `config/` | kustomize manifests, RBAC, CRDs, samples |
| `bundle/`, `catalog/` | OLM bundle and file-based catalog (see `DEPLOY.md` for release flow) |
| `test/e2e/` | e2e tests |
| `docs/` | method, roadmap, planning docs, ADRs |

## How we build — the method

The full method (our customized BMAD: **story → plan → red → green → verify →
audit**) is codified in **`docs/method.md`**. The short version, in required order
for any non-trivial change:

1. **Roadmap story** in `docs/roadmap.md` — *As a `<role>`, I want `<capability>`,
   so that `<value>`* — with a **Done:** line. Stories are coded (e.g. `SEC.1`).
2. **Approval gate.** A story is 📝 Draft until a human marks it 👍 Approved.
   Nothing is built from a Draft — comply-by-default is the failure mode.
3. **Planning doc** in `docs/planning/<slug>.md` (copy `TEMPLATE.md`) — a context
   capsule a cold session could execute. Committed before implementation.
4. **TDD.** Failing test first (red → green → refactor), table-driven, through
   envtest/fakes. No production code without a test that demanded it.
5. **DoD.** `make test` and `make lint` green — output actually seen; story marked
   ✅ with the planning-doc link; commit + push on a feature branch, never `main`.

**ADRs** (`docs/adr/`) record decisions with their rejected alternatives. Accepted
ADRs are immutable — a changed mind writes a superseding ADR.

## Invariants — rules for every change

1. **Secrets by reference.** Credentials (admin, DB, substitutions) flow through
   `SecretKeySelector`s. Never add a code path that logs, echoes, or inlines a
   secret value — including in CR status messages and job logs.
2. **Tenant isolation is load-bearing.** Anything that lets one namespace affect
   another team's instance (cross-namespace imports, shared labels/selectors) is a
   security surface — treat changes there as such and say so in the PR.
3. **Reconcilers are idempotent and owner-scoped.** Every created resource gets a
   controller reference (or owner labels + finalizer cleanup where cross-namespace);
   `CreateOrUpdate` with a mutate fn, never blind creates.
4. **Honest status.** `status.conditions` reflects observed state; never set Ready
   on unverified assumptions. Same for your own reports: "couldn't verify" beats a
   confident guess.
5. **Don't widen scope silently.** If a fix needs a bigger change, stop and say so.

## Definition of Done

A coding change is done only when:

- `make test` passes (runs manifests + generate + fmt + vet + envtest suite) and
  `make lint` is clean — **run them, see them green**, don't claim it.
- Generated artifacts are current: `make manifests generate` produces no diff
  (CRD YAML under `config/crd/`, `zz_generated.deepcopy.go`).
- New/changed behavior ships with tests (controller tests via envtest; pure logic
  like sizing as plain table-driven unit tests).
- Docs moved with the code: roadmap story status, planning doc, any affected
  `docs/` content.

There is no `make verify` umbrella target or edit-hook here yet (kube-ward has
one) — adding it is a Draft DX story in the roadmap. Until then, run the targets
yourself.

## Go / operator conventions

- **controller-runtime idioms:** builders own their resource type under
  `internal/resources/`; reconcilers orchestrate, builders build. Keep that split.
- **Table-driven tests** by default; envtest for anything that touches the API
  server; e2e under `test/e2e/` against a real cluster.
- **Narrow RBAC.** New `+kubebuilder:rbac` markers request the minimum verbs; the
  operator's ClusterRole is a security surface (it can read secrets cluster-wide),
  so every widening needs justification in the PR.
- **Watches are label-filtered.** The manager cache only watches secrets labeled
  `rhbk.stakater.com/watched=true` (`cmd/main.go`) — keep new watches bounded the
  same way.
- **OpenShift awareness is explicit.** Route/service-CA/SCC assumptions are features
  here, not accidents — but keep them in the builders, not smeared through the
  reconcilers.

## Release flow

See `DEPLOY.md`: bump `VERSION` in the Makefile → build/push operator + bundle
images → `opm render` into `catalog/index.yaml` → maintain OLM upgrade path
(`replaces`/`skips`). Images are tagged `v{{VERSION}}`.
