# DX.1 — `make verify` umbrella target and edit-hook

Story: `docs/roadmap.md` DX.1 (👍 Approved 2026-11-22, requested as the pilot
PR for issue #17). Driver: issue #17.

## Why (one paragraph)

The Definition of Done is "`make test` and `make lint` green, and
`make manifests generate` leaves no diff", but it is three manual steps and
nothing runs them for you. CI runs `make lint` and `make test` through the
shared `stakater/.github` verify action. `make test` regenerates CRDs, RBAC,
deepcopy and formatting **in place** before it tests, so CI tests the
regenerated tree and never checks that the committed generated files are
current. Locally, `make lint` fails on the maintainers' Go 1.27 with
typecheck noise (golangci-lint v1.64.2 cannot type-check under it; first
seen in PR #31), so the local answer and the CI answer disagree. The result is
that "done" is a claim, not a check — the gap issue #17 names.

## Governing constraints

- Method invariant 3 (honesty over completeness): the gate must give the same
  answer locally and in CI, or a green local run means nothing.
- Method invariant 4 (docs move with the code): CLAUDE.md DoD, `docs/method.md`
  (loop + practices table), `docs/roadmap.md` lifecycle text and the README
  "Developing" block all name `make test` + `make lint` today.
- CLAUDE.md invariant 5 (don't widen scope silently): no golangci-lint
  upgrade, no Go bump, no change to the shared `stakater/.github` workflows.
- Existing CI: the shared reusable workflows run lint/test only through their
  own composite action (`make lint`, `make test`); they take no custom target.

## Design

**`Makefile`** — new `verify` target in a `##@ Verify` section:

```make
GO_MOD_VERSION := $(shell sed ... go.mod)   # "1.23.0"; "1.24" → "1.24.0"

verify: export GOTOOLCHAIN = go$(GO_MOD_VERSION)+auto
verify: ## Definition of Done gate: ...
	hack/verify-no-diff.sh $(MAKE) fmt manifests generate
	$(MAKE) vet build lint test
```

- Order: fmt · manifests · generate (no-diff), then vet · build · lint · test.
- **No-diff check:** `hack/verify-no-diff.sh <cmd>` snapshots the working tree
  (tracked + untracked, minus `.gitignore`) into a throwaway git index, runs
  the command, snapshots again, and fails with the diff if the command changed
  any file. It compares before/after, not against `HEAD`, so it works on a
  dirty tree — the hook runs it mid-change. Including `fmt` means unformatted
  Go fails too (today `make test` would silently reformat it in CI).
- **Toolchain pin:** `verify` exports `GOTOOLCHAIN=go<go.mod version>+auto`,
  the same Go that CI's `setup-go` (`go-version-file: go.mod`) installs.
  Checked: golangci-lint's result depends on the `go` it runs, not the one it
  was built with — a Go 1.27-built binary passes under go1.23.0, a
  go1.23.0-built binary fails under Go 1.27. Only `verify` is pinned;
  `make test`/`make lint` keep using the local Go. Command-line override
  stays possible (`make verify GOTOOLCHAIN=local`).
- `build`/`test` re-run the generators in the second sub-make (~4 s warm).
  Accepted: it keeps the existing targets unchanged.

**`hack/verify-hook.sh`** + **`.claude/settings.json`** — Claude Code
PostToolUse hook on `Edit|Write|MultiEdit`:

- Reads `tool_input.file_path` from the JSON on stdin (`jq` if present, else
  `grep`/`sed` taking the first `file_path`).
- Runs only for `*.go`, `go.mod`, `go.sum`, `*.yaml`, `*.yml`, `Makefile`;
  skips gitignored files and files outside this repository (it compares
  `git rev-parse --git-common-dir`, so any worktree of this repo counts and
  `make verify` runs in the tree that holds the edited file).
- Output goes to `bin/verify-hook.log` in the verified tree (gitignored, so
  it does not trip the no-diff check; overwritten each run). Success: exit 0.
  Failure: exit 2 (the only code Claude Code feeds back to Claude for
  PostToolUse) with the last 100 lines and the log path on stderr — full
  `go test -v` output would flood the context.
- Settings: `timeout: 900` (a cold first run downloads Go 1.23, the tools and
  the envtest binaries), `statusMessage`.

**CI** — `.github/workflows/pull_request.yaml` and `push.yaml`: a new `verify`
job (checkout, `setup-go` from `go.mod`, `make verify`, `contents: read`).
Checkout uses `persist-credentials: false` (the job runs PR code and needs
no token). The change is additive: the shared job is not changed, keeps its
own lint and unit-test steps, and runs in parallel. In `push.yaml`, `verify`
gets the same `[skip-ci]` condition as the push job.

**Docs** — CLAUDE.md (workflow step 5, DoD, drop the "no `make verify` yet"
paragraph), `docs/method.md` (loop node, practices table row, lifecycle),
`docs/roadmap.md` (lifecycle text, DX.1 status), README "Developing".

NOT changing: existing targets (`test`, `lint`, `build`, `manifests`,
`generate`), `.golangci.yml`, golangci-lint version, Go version, the shared
workflows, image build. No CRD, RBAC or Go code change.

## Tests (red first)

No Go seam: this is build machinery, so the red/green proof is a recorded
manual run, listed in the PR.

1. Red: on `main`, `make verify` → `No rule to make target 'verify'`.
2. Green: `make verify` passes on the branch, on a host with Go 1.27
   (toolchain pin in effect).
3. Red (stale generated file): change a CRD field doc comment without
   regenerating → `make verify` fails at the no-diff step and names
   `config/crd/bases/...`; revert → green.
4. Red (unformatted Go): break formatting in a `.go` file → fails at the
   no-diff step; revert.
5. Dirty tree: an uncommitted but correctly regenerated change → no-diff
   passes (it compares before/after, not `HEAD`).
6. Hook, fed payloads on stdin: `.md` file → exit 0, no make run; file
   outside the repo → exit 0; `.go` file with a compile error → exit 2, tail
   + log path on stderr; clean `.go` file → exit 0. Both `jq` and the
   `sed` fallback.
7. Baseline (already run): on `main`, `make lint` with Go 1.27 fails
   (typecheck); with `GOTOOLCHAIN=go1.23.0+auto` it passes.

## Out of scope

- Upgrading golangci-lint (v2) or Go so `make lint` works on new Go without
  the pin — separate story; the pin then follows `go.mod` automatically.
- Making the shared job `needs: verify` (no image from a red gate) and turning
  off its own lint/unit-test steps (they then run twice) — it changes and
  disables existing CI steps, so it is a maintainer decision, not part of
  this story. A `VERIFY_TARGET` input on the shared `stakater/.github`
  workflows would also remove the duplication — cross-repo change.
- Enforcing red-first itself: the hook proves the tree is green, not that a
  test was written first; that stays review.
- e2e in `make verify` (needs a live OpenShift cluster, #21).
- A Stop hook (run once per turn instead of per edit) — a possible follow-up
  if per-edit runs prove too slow.

## Done means

Roadmap Done line: target exists, CI uses it, hook wired in `hack/`. Plus:
`make verify` green locally (output seen), the stale-file case seen red, CI's
`verify` job green on the PR, docs above updated, roadmap DX.1 ✅ with this
doc linked; closes #17.
