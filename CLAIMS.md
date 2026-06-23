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

### Batch BI — Brownfield-aware `init` (inject-with-consent, review handoff)
- Owner: claude
- Started: 2026-06-23 17:08

## Completed

### Batch CFG — Config file, commit/push levers & safety fixes (v0.1 foundation)
- Owner: claude
- Started: 2026-06-23 14:51
- Finished: 2026-06-23 17:02
- Commit: 5204656

**What shipped.** The v0.1 foundation the other batches read. The stamp is renamed to
`specflow/config.json` (out of the repo root) and now carries a **`config` block** —
`agents` / `mode` / `commit` / `push` — alongside internal state (versions, schema, `managed`
hashes); all references + tests updated. The two **commit/push levers** are honored by the
procedures: `AGENTS.md` gains an authoritative *Commit & push authority* section defining
`config.commit` / `config.push` (`agent|user`), and `claim-batch.md` / `finish-batch.md` each carry
a compact lever-note — when `commit: user` the agent alerts at commit points and supplies a suggested
message rather than committing; when `push: user` it commits but never pushes; default stays
`agent` / `agent`. (The *interactive* lever choice at `init` is deferred to Batch BI's init overhaul;
CFG lands the field + the procedure behavior with agent/agent defaults.) **Safety fixes:** a managed
file with no recorded baseline is now treated as **drift** — `upgrade` writes a `.specflow-new`
sidecar and never overwrites (closes risk A); a corrupt/hand-edited `config.json` fails with a
friendly "fix or restore it" message; `init` now **requires a git repo** and refuses otherwise
(nothing written), closing the no-undo risk on brownfield repos. **CLI hygiene:** colour output is
suppressed when stdout isn't a TTY or `NO_COLOR` is set (no escape codes in piped/redirected output).
10 tests pass; `go vet` clean; self-host `upgrade` is a clean idempotent no-op. **Follow-ups:** the
init-time lever prompt + brownfield two-phase flow land in Batch BI; the `_DONE` archives move under
`specflow/history/` in BI.

### Batch G1 — Port the CLI to Go (full replace)
- Owner: claude
- Started: 2026-06-21 05:02
- Finished: 2026-06-21 06:16
- Commit: ac82cec

**What shipped.** The CLI is now a single statically-compiled **Go binary** (`cmd/specflow` +
`internal/kit`), replacing the Node `bin/specflow.js` at functional parity (`init`, `upgrade`,
`--version`, `--help`, unknown-command exit, non-git warning). Templates are embedded via
`//go:embed all:templates` (the `all:` prefix keeps the dotfiles — `.claude/`, `.cursor/`, etc.).
The smoke suite was ported to `go test`, which builds and drives the real binary against temp repos.
Verified parity three ways: `init` output is **byte-identical** to the Node CLI, the stamp is
semantically identical (managed SHA-256 hashes match), and `upgrade` is a clean idempotent no-op;
self-hosted by running the Go `upgrade` on this repo (regions untouched, only the stamp bumped).
Node files removed (`bin/specflow.js`, `package.json`, `test/smoke.js`); CI switched to
`go vet`/`build`/`test` on ubuntu + macos; `.gitignore` drops node_modules, adds `/dist/`; README +
`architecture.md` file-map updated to the Go layout; pending batches (1/2/3/5/W/NB) retargeted from
Node to Go; the obsolete Node-version-guard open question dropped. **Follow-up:** Batch G2 — the
GoReleaser → GitHub Releases pipeline with `curl|sh` + Homebrew front-ends — ships the binary.

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
change is canonicalized, outside text preserved, no backup). Self-hosting check: ran `upgrade` on the
repo root — only the marker line changed in each managed file. No follow-ups.

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
smoke suite green (outside-text-survives, drift-not-clobbered, pre-marker-migration). Self-hosting:
specflow's own root `AGENTS.md` + procedures migrated to the format, stamp now carries `managed`.
Spec updated (`architecture.md`, `open-questions.md`). **Follow-ups deferred:** none specific to U;
`--dry-run` (Batch 5) and `status`/drift-flag (Batch 2) build naturally on this.
