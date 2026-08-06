<!-- specflow:start - managed by specflow; do not edit inside these markers (your edits block specflow upgrade). Add your own notes outside them. -->
# specflow

This repo uses **specflow**, a shared protocol for AI coding agents. IBM Bob loads `AGENTS.md`
at the repo root automatically — that file is the full protocol. This rule just reinforces it.

<!-- specflow:full-only:start -->
**Read `AGENTS.md` before doing anything.** Work is specced (`spec/`), split into batches
(`BUILD_QUEUE.md`), and each batch is claimed in git (`CLAIMS.md`) before code is written. The
three procedures live in `specflow/procedures/` — read the relevant one before acting:

- Before starting any new batch → `specflow/procedures/claim-batch.md`
- Before editing any `spec/**` file or persisting a design decision → `specflow/procedures/spec-edit.md`
- When wrapping up a batch → `specflow/procedures/finish-batch.md`

Commit grammar: `batch-N:`, `meta:`, `spec:`. Never force-push the shared branch. Never write
claim state into `BUILD_QUEUE.md`.
<!-- specflow:full-only:end -->
<!-- specflow:spec-only:start -->
**Read `AGENTS.md` before doing anything.** Design is written down in `spec/`, and the user
approves a design before it is persisted or built. The spec procedure lives in
`specflow/procedures/` — read it before acting:

- Before editing any `spec/**` file or persisting a design decision → `specflow/procedures/spec-edit.md`

Commit grammar: `spec:` for spec edits, `meta:` for tooling. Never force-push the shared branch.
<!-- specflow:spec-only:end -->
<!-- specflow:end -->
