# FIX.1 — nil-safe KeycloakImport substitutions

Story: `docs/roadmap.md` FIX.1 (👍 Approved 2026-11-03). Driver: issue #11
(nil-pointer panic), pattern-swept per the house rule (#209-style: fix the
pattern, not the instance).

## Why (one paragraph)

`SecretOptionVar` legally carries either an inline `value` or a `secret` ref,
but two code paths dereference `.Secret` unconditionally. A single
`KeycloakImport` with a value-only substitution panics the controller: once in
`HasSecretReference` (called from the secret watch mapper — crash-loops the
manager on the next labeled-secret event, a cluster-wide DoS any tenant can
trigger), and once in `ImportRealmSecret.CreateOrUpdate` (panics that import's
reconcile). Value-only substitutions are also silently ignored rather than
used.

## Governing constraints

- CLAUDE.md invariant 3 (idempotent reconcilers) — fix must not change
  resource ownership or flow.
- CLAUDE.md invariant 1 (secrets by reference) — inline values stay legal for
  non-secret substitutions (display names etc.); credentials guidance is
  SEC.7's problem, not this story's.
- Method: fix the pattern — both deref sites, one change.

## Design

- `api/v1alpha1/keycloakimport_types.go` — `HasSecretReference`: skip
  substitutions with `Secret == nil`.
- `internal/resources/realm/realm_import_secret.go` — `CreateOrUpdate`
  substitution loop: `Secret == nil` → use `Value` (escaped, same as secret
  values). No CRD schema change, no new fields, no RBAC change.

## Tests (red first)

- `api/v1alpha1/keycloakimport_types_test.go` (new): table over substitution
  shapes — secret-ref match, secret-ref no-match, value-only (panics today),
  mixed, empty.
- `internal/resources/realm/realm_import_secret_test.go` (new): fake-client
  `CreateOrUpdate` with a value-only + a secret-ref substitution; asserts the
  rendered realm JSON contains both (panics today), and the escaping path.

## Out of scope

Validation that exactly one of `value`/`secret` is set (SEC.8, #12);
credentials-by-reference enforcement (SEC.7, #10); negative-path tests for the
other watch mappers (#27).

## Done means

Both new tests red before the fix, green after; `make test` + `make lint`
green; roadmap FIX.1 ✅ with this doc linked; closes #11.
