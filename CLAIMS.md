# Claims

Execution-state ledger, managed by coding agents. Records who is working on what and the recent
completion log. The user does not normally edit this. Procedures:
`specflow/procedures/claim-batch.md`, `specflow/procedures/finish-batch.md`, and
`specflow/procedures/prune-ledgers.md`.

## Releases need the user's approval

**Never cut a release without Matan's explicit go-ahead, every single time.** "Cut a release" means
any of: bumping the version in `cmd/specflow/main.go` or `specflow/config.json`, creating a `v*` tag,
or pushing one. Approval for one release is never approval for the next.

A pushed tag is now self-publishing (`.goreleaser.yaml` → `release.draft: false`): GoReleaser builds
the archives and puts them straight on a public GitHub Release, and `install.sh` resolves
`releases/latest`, so the push is immediately live for every user running `curl … | sh`. There is no
draft to review and no undo worth the name. The checkpoint that used to sit at "publish the draft"
now sits at "should we tag at all" — and it is the user's, not the agent's.

**The release commit writes the notes.** `meta: release vX.Y.Z` covers the version bump *and*
`.github/release-notes/vX.Y.Z.md`; the workflow passes that file to GoReleaser, so the pushed tag
publishes the body you wrote. Miss it and the release still ships, with GoReleaser's commit list as
the body and a warning in the job summary — but fixing it afterwards needs a GitHub API token, which
`git push` over SSH does not provide, so in practice a missed file stays missed. Write it before you
tag. Shape and house style: `spec/architecture.md` → *Distribution*; worked example:
`.github/release-notes/v0.1.6.md`.

Entry format:

```
### Batch N — <short title>
- Owner: <agent>
- Started: YYYY-MM-DD HH:MM        (UTC)
- Finished: YYYY-MM-DD HH:MM       (only in Completed)
- Commit: <short SHA>              (only in Completed)
- Handoff note: ...                (only when a mid-batch handoff occurred)

<up to 8 lines of "What shipped": what changed, where, what a resuming agent must know first>
- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch N
```

The completed entry is a **stub**, not the record. This file is re-read on every claim, finish, and
prune, so the batch's full narrative lives in `specflow/history/BUILD_QUEUE_DONE.md` and the stub
says only enough for a resuming agent to know whether it needs to go read it. `specflow finish`
refuses a stub over 8 lines. Entries above the LW line predate the rule and are left as written.

## In progress

<!-- One entry per actively claimed batch. -->

### Batch MC — migrate-claims, so 0.1.8's ledger shape reaches old entries
- Owner: claude
- Started: 2026-08-21 14:38

## Completed

### Batch RC — drift is a state you can leave
- Owner: claude
- Started: 2026-08-21 14:26
- Finished: 2026-08-21 14:36
- Commit: 2bda486

Reconciling a drifted managed file was destructive and never cleared. The `.specflow-new` sidecar
now carries your file with the fresh region spliced in, so `mv` is the correct reconciliation for
both managed tiers; a region that already matches the template is adopted, so taking the sidecar
ends the drift instead of re-drifting forever; and `specflow waive <file>` (`--all`, `--clear`)
keeps a deliberate edit without a baseline re-record, which would have let the next `upgrade`
destroy the edit being blessed.

**The waiver is the design correction to make before building on this**: the queue entry proposed
`specflow adopt` as a baseline re-record, and that shape is a trap, not a nicety.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch RC

### Batch LW — Ledger weight: bound the entry, not just the count
- Owner: claude
- Started: 2026-08-21 13:30
- Finished: 2026-08-21 13:38
- Commit: 5dcced8

**What shipped**
- `CLAIMS.md` entries are now stubs (8 lines max + pointer); the full narrative goes to `BUILD_QUEUE_DONE.md`.
- `specflow finish --stub-file` refuses an over-length stub before writing; `--summary-file` still works.
- `BUILD_QUEUE.md`'s preamble is capped at 45 lines, waivable with `specflow:size-ok`; `prune-ledgers` §3 audits it.
- `next` and `verify` print ledger line counts and warn past a bound.
- Absorbed the companion Batch NX was carrying, so NX no longer needs it.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch LW

### Batch CD — Batch-width and prune discipline in the procedures
- Owner: claude
- Started: 2026-08-21 06:14
- Finished: 2026-08-21 06:17
- Commit: e2ef30c

**What shipped**
- **`AGENTS.md` → *The work queue*** (full-only) now states the sizing rule: a batch is sized by the
  **layers it crosses**, not the deliverables it lists, and one spanning more layers than its goal
  needs gets split on the layer seam. The split pieces declare disjoint file lists, which is what the
  parallelism rule directly above it already required.
- **`specflow/procedures/spec-edit.md`** carries the same rule where batch sections are actually
  written (persisting a decision → step 3), so an agent turning a decision into queue entries meets it
  at the moment it matters, not only when reading `AGENTS.md`.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch CD

### Batch RN — Authored release notes
- Owner: claude
- Started: 2026-08-20 16:43
- Finished: 2026-08-20 16:46
- Commit: 26b2cbc

**What shipped**
- **`.github/workflows/release.yml`** gained a `Resolve release notes` step: it maps the pushed tag
  to `.github/release-notes/<tag>.md` and, when that file exists, runs GoReleaser with
  `--release-notes=<path>`. Confirmed against the real GoReleaser v2 binary that the flag exists and
  "will skip GoReleaser changelog generation", so the authored body replaces the commit list rather
  than sitting beside it.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch RN

### Batch AF — Adapter files upgrade like everything else
- Owner: claude
- Started: 2026-08-20 13:56
- Finished: 2026-08-20 14:03
- Commit: f5f578f

**What shipped**
- **The second managed tier.** `MANAGED` only ever covered files with a marker-wrapped region, so
  the wholly-generated adapters — the four Claude `SKILL.md` stubs and
  `.claude/hooks/specflow-handoff-reminder.sh` — were create-once: `upgrade` placed one when absent
  and otherwise left it alone forever. `internal/kit/kit.go` now hashes each adapter *whole* into
  the stamp's `managed` map (`adapterEntries` / `decideAdapter` / `adapterDecisions`), and both
  tiers funnel through one `applyDecision`, so a region and an adapter in the same state are
  written and reported identically.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch AF
