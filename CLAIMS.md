# Claims

Execution-state ledger, managed by coding agents. Records who is working on what and the recent
completion log. The user does not normally edit this. Procedures:
`specflow/procedures/claim-batch.md` and `specflow/procedures/finish-batch.md`.

Entry format:

```
### Batch N — <short title>
- Owner: <agent>
- Started: YYYY-MM-DD HH:MM        (UTC)
- Finished: YYYY-MM-DD HH:MM       (only in Completed)
- Commit: <short SHA>              (only in Completed)
- Handoff note: ...                (only when a mid-batch handoff occurred)
```

## In progress

<!-- One entry per actively claimed batch. -->

## Completed

<!-- Recent finishes, newest first. Older entries archived to CLAIMS_DONE.md. -->

### Batch U2 — Self-documenting, edit-resistant region markers
- Owner: claude
- Started: 2026-06-17 15:21
- Finished: 2026-06-17 15:55
- Commit: 66442e0

**What shipped.** The `specflow:start` marker now carries an inline note — `managed by specflow; do
not edit inside these markers (your edits block specflow upgrade). Add your own notes outside them.`
— so a human editing the raw file is warned (the note is an HTML comment, invisible in rendered
markdown). To keep that safe, `extractRegion` matches markers by their `specflow:start`/`specflow:end`
**token** (regex `START_RE`/`END_RE`) rather than an exact string, and returns the matched marker
text; the clean-path rewrite re-applies the *template's* marker wording, so the note can evolve
without breaking parsing or forcing a migration. Smoke suite at 19 checks (added: marker-wording
change is canonicalized, outside text preserved, no backup). Dogfooded: ran `upgrade` on the repo
root — only the marker line changed in each managed file. No follow-ups.

### Batch U — Non-destructive upgrade redesign
- Owner: claude
- Started: 2026-06-17 14:54
- Finished: 2026-06-17 14:59
- Commit: 42cd047

**What shipped.** `upgrade` no longer wholesale-overwrites managed files. Each managed file
(`AGENTS.md` + the three procedures) wraps its generated content in `<!-- specflow:start -->` /
`<!-- specflow:end -->` markers; `init` records a SHA-256 of each region in the stamp's new
`managed` map. On `upgrade`: a clean region (hash matches baseline) has only its between-markers
content replaced (everything outside preserved verbatim); a drifted region (hash differs) is left
untouched, with the fresh version dropped to a `<file>.specflow-new` sidecar and reported; a
pre-marker file is migrated (backed up to `<file>.specflow-bak`, then rewritten with markers).
Implemented in `bin/specflow.js`; markers added to `templates/base/AGENTS.md` + procedures; 18-check
smoke suite green (outside-text-survives, drift-not-clobbered, pre-marker-migration). Dogfooded:
specflow's own root `AGENTS.md` + procedures migrated to the format, stamp now carries `managed`.
Spec updated (`architecture.md`, `open-questions.md`). **Follow-ups deferred:** none specific to U;
`--dry-run` (Batch 5) and `status`/drift-flag (Batch 2) build naturally on this.
