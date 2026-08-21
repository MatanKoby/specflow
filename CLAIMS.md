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

### Batch RC — drift is a state you can leave
- Owner: claude
- Started: 2026-08-21 14:26

## Completed

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
- **What counts as a layer is deliberately per project.** Seams differ per repo (this one has spec,
  templates, root managed files, CLI, tests); an enumerated list shipped in a template would be wrong
  in most installs and would read as a contract rather than a heuristic.
- **`specflow/procedures/claim-batch.md`** tests the retention rule **before claiming**, in the
  eligibility section next to the heading-grep guidance:
  `sed -n '/^## Completed/,$p' CLAIMS.md | grep -c '^### '` — more than 5 means run `prune-ledgers`
  and commit it on its own before the claim. Pruning at finish alone doesn't help whoever claims next:
  they still read the overgrown ledger on the way in, and that read is the cost. Same 5 that `finish`
  enforces, so no second threshold and no stop-and-ask.
- **Verification:** `init` into temp repos in both modes — full carries both changes, spec-only
  carries neither and still never names the queue. `specflow verify` clean, `config.check` green,
  managed regions refreshed by a self-hosted upgrade.

**Not shipped, on purpose**
- The report's write-side context rule (edit via the file tool, not the shell, outranking harness
  modes) was **dropped after discussion with Matan**: the tool is a proxy for the real cost, which is
  re-emitting a file's contents to change part of it, and the "outranks your harness" clause has
  specflow claiming jurisdiction over the host environment. Parked with its reasoning in
  `spec/open-questions.md` → *CLI / upgrade behavior*, alongside the optional `next` file-spread item
  (**Batch NX**, `[NOT READY]`).

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
- **The fallback is the default branch**, not an error branch: no notes file → the exact args the
  workflow ran before, plus a `⚠` line in the job summary. Neither branch can fail the job. A
  release that ships no archives is far worse than one with a plain body, and that failure mode
  (v0.1.3, v0.1.4) is the reason `draft: false` exists in the first place.
- **`CLAIMS.md` → *Releases need the user's approval*** now states that `meta: release vX.Y.Z`
  covers the notes file as well as the version bump, and why a missed file stays missed (fixing a
  published body needs an API token; `git push` over SSH doesn't provide one).
- **`.github/release-notes/v0.1.6.md`** backfilled as the worked example, in the shape the spec
  specifies. The published v0.1.6 release keeps its generated body — the user explicitly didn't want
  it updated.
- **Scope addition:** a comment in `.goreleaser.yaml` above `changelog:` marking that block as the
  fallback path, since `--release-notes` bypasses it entirely and the config otherwise reads as
  live on every release.

**Verification.** Both workflow branches simulated locally against real `GITHUB_OUTPUT` /
`GITHUB_STEP_SUMMARY` files: present → `release --clean --release-notes=.github/release-notes/v0.1.6.md`;
absent → `release --clean` plus the warning. Both YAML files parse. `act` isn't available here, so
the job itself is unproven until the next tag — the first real test is the next release.

**Follow-up.** The notes file is a convention the workflow can't enforce at commit time (it only
sees the tag). If a release ever ships with the generated body again, the next lever is a `ci.yml`
check on `meta: release` commits.

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
- **The one-time adoption.** Every install made before this has no adapter baselines at all. On the
  next `upgrade` each adapter is compared to the current template: identical → record the baseline
  silently; anything else → `.specflow-bak` then replace. That is what carries existing installs
  across; from then on the normal clean/drift rules apply. Verified on this repo: `upgrade` adopted
  all four stubs, the hook was already current, and the four backups were byte-identical to `HEAD`.
- **`verify` and `status` read the same decisions.** `verify` now lists the adapters (a deleted,
  truncated, or hand-mangled one used to pass clean); `status` gained a **stale** row, separating
  "you edited it" from "specflow moved and this file didn't" — it printed the 4 behind stubs on this
  repo before the upgrade and `none` after.
- **The stubs learned the verbs.** A skill loads *before* the procedure, so its "In short:" is what
  gets acted on, and all four described hand-editing markdown only. Each now names its verb
  (`specflow claim`, `specflow finish`, and for `prune-ledgers` the fact that `specflow finish`
  already does the `CLAIMS.md` half); `spec-edit`'s addition is `full-only`-gated so a spec-only
  install still names no queue machinery.
- **Drive-by fix.** `normalizePath` in `internal/kit/queue.go` trimmed a leading `.` along with
  trailing prose punctuation, so `.claude/…`, `.github/…`, and `.cursor/…` compared as different
  files and were invisible to `specflow next`'s overlap check. Caught by this batch's own file list.

**Verification.** 10 new tests in `cmd/specflow/main_test.go` (init baselines the adapters; a stale
adapter refreshes with no backup; a no-baseline install adopts, backing up only what differs; an
edited adapter is untouched with a `.specflow-new` sidecar; a deleted one is caught by `verify` and
restored by `upgrade`; each stub names its verb; the spec-only `spec-edit` stub names none; dotfile
paths keep their leading dot). `gofmt`/`go vet`/`go test ./...` clean, and the whole flow was run
end-to-end against this repo's own install.

**Follow-up.** An existing install gets one `.specflow-bak` per adapter it didn't already match —
expected and reported by `upgrade`, but worth a line in the release notes so users know to delete
them.

### Batch QV — Queue verbs (`next`, `claim`, `finish`)
- Owner: claude
- Started: 2026-08-20 12:47
- Finished: 2026-08-20 12:58
- Commit: a7df418

**What shipped**
- **`internal/kit/queue.go`** (new, 500 lines): the declared-batch-shape parser plus the three verbs.
  `ParseQueue` reads the heading (`## Batch <id> [TAG] — <title>`, backticked or bare tag), the
  optional `**Depends on:**` line (parenthetical rationale ignored, "none" understood), and the
  `### Files this batch creates/edits` list (backticked paths, `dir/{a,b}.md` brace-expanded). A
  batch missing the file list, or sharing its id with another section, comes back with `Problem` set
  and is never offered as claimable. `ParseClaims` errors out if either `##` section heading is
  missing, which is what keeps a hand edit from being rewritten.
- **`specflow next [--json]`** — read-only. Applies the whole Eligibility section of
  `claim-batch.md` in one call, in the order the procedure states it: tag → unparseable → already
  claimed → dependency → file overlap with an in-progress batch. Blocked batches print with the
  reason; the JSON form carries id, title, files, and reason.
- **`specflow claim <id> [--as <agent>]`** — writes the In-progress entry (heading, Owner from
  `config.agents`, `Started` in UTC) and refuses any batch `next` would not offer. `--as` is
  required only when several agents are wired.
- **`specflow finish <id> --commit <sha> [--summary-file <path>] [--done-file <path>]`** — does
  steps 2, 3, 4, and 4a of `finish-batch.md`: entry to the top of `## Completed` with
  Finished/Commit/summary, batch section deleted from `BUILD_QUEUE.md`, paragraph filed in
  `BUILD_QUEUE_DONE.md`, `## Completed` pruned to its 5 newest with the overflow moved verbatim to
  `CLAIMS_DONE.md`. Every edit is computed in memory first, so a parse failure writes nothing.
- **No verb commits**, and no verb writes prose: `--summary-file` (or `-` for stdin) and
  `--done-file` carry the agent's words, and the CLI owns only placement, format, and timestamps.
  Omitting either flag still moves the batch and names what the agent owes by hand.
- **Procedures name the verbs as the fast path** (`claim-batch.md` Eligibility + Claim,
  `finish-batch.md` step 1, `prune-ledgers.md` When-to-run), with every manual step kept intact so
  non-CLI agents and hand edits work exactly as before. The queue template documents the declared
  shape.

**Verification.** `test -z "$(gofmt -l cmd internal)" && go vet ./... && go test ./...` green.
Nine new tests: table-driven parser cases (tag forms, dependency forms, brace expansion, missing
file list, empty file list, duplicate ids), the eligibility and overlap rules, the JSON shape,
claim's refusals, a full `next` → `claim` → `finish` round trip asserting the prune boundary at 5
and verbatim archiving, an unparseable-ledger case asserting nothing is rewritten, and a check that
no verb creates a commit. This batch was itself finished with `specflow finish`, which is how the archive-ordering bug fixed in
`a7df418` was caught: both archives are newest-first, and the pruned entry was landing at the end.

**Deviation from the declared file list.** The parser landed in a new `internal/kit/queue.go` rather
than in `kit.go`, which is already 1314 lines. No other batch was in progress, so nothing raced.

**Follow-up.** `next` reports Batch P as unparseable (no declared file list). That is user-owned
queue prose and P is `[NOT READY]`, so it was left alone.

