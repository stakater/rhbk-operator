# <Story code> — <short title>

<!--
The planning doc is a CONTEXT CAPSULE (docs/method.md): a cold session with no
memory and no conversation history must be able to execute this story from this
file alone, without re-scouting the codebase. Fill every section; write "none"
rather than deleting one — an explicit "none" is information, an absent section
is a question. Delete this comment block in the real doc.
-->

Story: `docs/roadmap.md` <Story code> (status must be 👍 Approved before this
doc is implemented). Driver: <the issue, security finding, or need that
prompted it — cite issue numbers/links>.

## Why (one paragraph)

<What breaks or is missing today, for whom, and why now. If the story came
from a security finding or incident, name the issue ids.>

## Governing constraints

<The ADRs and CLAUDE.md invariants this change must respect, by number/name,
with one clause each on how they bite here. E.g. "Invariant 2: tenant
isolation — the new field must not let one namespace select another's
instance.">

## Design

<The change, at seam level: which packages, which files, which types — exact
paths. New CRD fields / types shown as snippets, including kubebuilder markers
and what `make manifests` will regenerate. State explicitly what is NOT
changing. RBAC marker changes called out separately.>

## Tests (red first)

<The failing tests to write BEFORE implementing: test file, seam (unit /
envtest / e2e), and the behavior each pins — including the negative/regression
cases that must stay green. If a live incident or finding motivated the story,
reproduce it as a test case.>

## Out of scope

<What this story deliberately does not do, and where that work lives instead
(a follow-up story code, an ADR to write first, or "accepted limitation" with
the honest consequence stated).>

## Done means

<Restate the roadmap Done line, plus anything mechanical: make test + make
lint green, make manifests/generate no-diff, docs updated (which ones),
upgrade-path note if the CRD schema changed.>
