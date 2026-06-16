# specflow

This repo uses **specflow**, a shared protocol for AI coding agents. Google Antigravity reads
`AGENTS.md` at the repo root natively (its highest-priority context file) — that file is the full
protocol. This rule just reinforces it.

**Read `AGENTS.md` before doing anything.** Work is specced (`spec/`), split into batches
(`BUILD_QUEUE.md`), and each batch is claimed in git (`CLAIMS.md`) before code is written. The
three procedures live in `specflow/procedures/` — read the relevant one before acting:

- Before starting any new batch → `specflow/procedures/claim-batch.md`
- Before editing any `spec/**` file or persisting a design decision → `specflow/procedures/spec-edit.md`
- When wrapping up a batch → `specflow/procedures/finish-batch.md`

Commit grammar: `batch-N:`, `meta:`, `spec:`. Never force-push the shared branch. Never write
claim state into `BUILD_QUEUE.md`.
