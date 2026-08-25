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

<up to 8 prose lines of "What shipped": what changed, where, what a resuming agent must know first>
- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch N
```

The completed entry is a **stub**, not the record. This file is re-read on every claim, finish, and
prune, so the batch's full narrative lives in `specflow/history/BUILD_QUEUE_DONE.md` and the stub
says only enough for a resuming agent to know whether it needs to go read it. `specflow finish`
refuses a stub over 8 lines. **The cap counts prose lines**: blank lines and the `Full narrative`
pointer are free, so a stub with paragraph breaks is longer than 8 lines on disk and still passes.
`finish` writes the pointer itself when the stub omits it, and refuses one naming another batch.
Entries above the LW line predate the rule and are left as written.

## In progress

<!-- One entry per actively claimed batch. -->

### Batch FL - a finish leaves the queue smaller
- Owner: claude
- Started: 2026-08-25 11:15

## Completed

### Batch PD - prune-ledgers section 3, duplication-first
- Owner: claude
- Started: 2026-08-25 10:41
- Finished: 2026-08-25 10:43
- Commit: ef8354c

**What shipped.** `prune-ledgers.md` section 3 is duplication-first. New **3a**: grep the archives,
`CLAIMS.md` and `spec/` for a distinctive phrase from each preamble paragraph, and a paragraph with
a citation is a delete, mechanically, no ask, on the same losslessness bar sections 1 and 2 use.
Only uncited paragraphs become the three piles, and they go to the user **once, as piles with
counts**, not paragraph by paragraph. New **3b** prescribes the report shape.

The section now also names QD's other three warnings and says **the length is the weakest of the
four**: a 40-line preamble naming a dozen shipped batches is rotten, a 60-line live one may be fine.

Managed copies and `config.json` baselines re-recorded through a locally built `upgrade`.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch PD

### Batch QD - the queue warning says what is wrong, not just how long
- Owner: claude
- Started: 2026-08-25 10:35
- Finished: 2026-08-25 10:39
- Commit: 218dc17

**What shipped.** `Weigh` reports four things about the queue preamble instead of one line count:
the count with an address (`N lines above the first "## Batch" heading`, plus the heading holding
the bulk), the split between archived and live batch ids it names, a `claimable` line pointing at a
batch that has shipped, and a heading that reads as a batch but misses the declared shape.

The ratio is the diagnostic: 43 archived ids against 2 live says a file is narrating history, which
a line count cannot say at any length. `verify` gets all four, `--json` carries the numbers.

**The near-miss check is also the guard on the count itself**: an unparsed heading is preamble
prose, so it inflates the number and hides batches from `next` at the same time.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch QD

### Batch EM — the emitters obey the rule ED wrote
- Owner: claude
- Started: 2026-08-25 08:34
- Finished: 2026-08-25 08:39
- Commit: 20ba874

**What shipped.** specflow no longer writes an em dash at runtime. `queue.go`'s claim entry and
`archiveHeading` join with ` - `, and every emitted string in `queue.go`, `kit.go` and `main.go`
(console output, `--help`, errors, warnings) uses plain punctuation.

The parsers are deliberately untouched: the `TrimLeft` sets at `queue.go:120` / `:686` and the `sep`
slice at `:194` still accept the em and en dash, because every install that ran 0.1.9 or earlier has
em-dash headings that `finish`, `next` and `migrate-claims` locate by heading. Three tests pin both
halves. Go comments stay as written; `AGENTS.md`'s style rule now covers emitted strings.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch EM

### Batch ED — one mechanical pass, no rewording
- Owner: claude
- Started: 2026-08-24 14:14
- Finished: 2026-08-24 14:22
- Commit: 358dd08

Swept every em and en dash out of what specflow ships: `templates/**`, `specflow/procedures/*.md`,
and this repo's own managed copies (`AGENTS.md`, `CLAUDE.md`, `.claude/skills/**`,
`.claude/hooks/**`). Punctuation only, no rewording. A fresh `specflow init` now writes an install
with no em dash in it. The managed baselines in `specflow/config.json` were re-recorded through a
locally built `upgrade`, so `verify` is clean and a self-hosted upgrade still agrees with the kit.
Out of scope on purpose and still dashed: `spec/**`, `README.md`, the ledgers, and the Go source,
where `internal/kit/queue.go` still writes `### Batch N — title` into downstream ledgers.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch ED

### Batch FS — the stub contract says what the code already does
- Owner: claude
- Started: 2026-08-21 15:01
- Finished: 2026-08-24 14:12
- Commit: 41ae200

**What shipped.** The three loose ends Batch LW left in the stub contract.

- The cap counts **prose lines** (blank lines and the pointer are free). Now stated wherever the
  cap is stated: `finish-batch.md` step 3, `templates/base/CLAIMS.md`, `finish --help`, and
  `StubMaxLines`' own doc comment.
- `finish` supplies the `Full narrative` pointer when the stub omits one (only alongside a
  `--done-file`), and **refuses** one that names a different batch instead of rewriting it.
- `BUILD_QUEUE_DONE.md` is no longer described as "one-paragraph summaries" in the queue template,
  `prune-ledgers.md`, or `spec/architecture.md` — it holds the full narrative.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch FS
