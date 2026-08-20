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

Editing a published release's notes afterward is fine and expected; the generated body is a plain
commit list.

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

### Batch RN — Authored release notes
- Owner: claude
- Started: 2026-08-20 16:43

## Completed

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

### Batch CE — Context economy + `config.check`
- Owner: claude
- Started: 2026-08-20 11:50
- Finished: 2026-08-20 11:56
- Commit: 25f419e

**What shipped.** Three changes aimed at the recurring per-batch context cost, all of them reaching
existing installs through `upgrade`:

1. **Read-shape steps in all four procedures.** Wherever a procedure said *what* state to check, it
   now also names the cheap read: `grep` the headings, then slice the one section by line number.
   `claim-batch` carries the two greps for its eligibility checks, `finish-batch` and `prune-ledgers`
   locate entries by line range, and `spec-edit` extends the read-the-index rule to inside a file.
2. **`AGENTS.md` → *Working economically*.** Read by headings; batch independent reads into one turn;
   never re-read to confirm your own write; run the one check command; read the batch's declared file
   list first. The queue-specific bullets are wrapped `full-only`, so a spec-only install does not
   get told about machinery it doesn't have.
3. **`config.check`.** A new config string: the repo's single check command. `init` asks for it
   (skippable) or takes `--check=`, `status` shows it, and `finish-batch` quotes it back before the
   final commit. specflow never validates or executes it.

**Why.** Instrumented over one long batch in a specflow-managed repo, file reads were 45% of tool
calls and 88% of context spend, and the single biggest read was a `cat` of a 22 KB `CLAIMS.md` to
answer a question its headings settle. Measured here: 419 bytes of headings against 17.2 KB for the
whole file (41x), 224 bytes against 7.5 KB for the queue (33x), and both ledgers are read 3 to 5
times per batch. Rationale in `spec/architecture.md` → *Context economy*.

**Two decisions worth carrying.** The field is named `check`, not `verify`, because `specflow verify`
already means install-integrity and an agent told to "run config.verify" would plausibly type the
wrong command. And this is deliberately *not* a smaller `CLAIMS.md` retention: retention is a count
of 5 for reasons already recorded under *Ledger lifecycle*, and the waste was the read shape, not
the entry count.

**Verification.** `go vet` + `go test ./...` green, including five new tests: `--check=` recorded and
surfaced by `status`; the skipped answer stored as an empty string; a check command containing quotes
and backslashes still producing valid JSON (the config template is text-substituted, so the value is
`json.Marshal`-escaped); the interactive prompt; and a legacy install whose config has no `check` key
at all, where `status` degrades to "not set". Rendered `AGENTS.md` checked in both modes (5 bullets
full, 4 spec-only) with `specflow verify` clean in each. Self-hosted `upgrade` refreshed this repo's
own managed files, `drift none`, and the repo now records its own check command.

**Follow-ups deferred.** `upgrade` does not backfill `check` into installs that predate it: the key
stays absent, which reads as "not set", and `status` prints the one-line hint on how to add it.
Batch QV depends on this batch and is next.

### Batch RD — Release auto-publish, and the user approves every release
- Owner: claude
- Started: 2026-08-16 13:05
- Finished: 2026-08-16 13:10
- Commit: e279fb9

**What shipped.** `.goreleaser.yaml` `release.draft` flipped `true` → `false`, so a pushed `v*` tag
now publishes a public release with its archives attached and no manual step. Plus the counterpart
rule at the top of this file (*Releases need the user's approval*) and the decision recorded in
`spec/architecture.md` → artifact host.

**Why.** The draft gate put the release *copy* in front of the *binaries*. v0.1.3 and v0.1.4 were
both created by hand in the GitHub UI instead of published from GoReleaser's draft, so each shipped
a release with **zero assets** while the real draft sat unpublished beside it. `install.sh:55`
resolves `releases/latest` and downloads `specflow_<ver>_<os>_<arch>.tar.gz` from it, so the public
`curl … | sh` install 404'd both times until it was noticed and repaired by hand. v0.1.1 has no
release at all for the same reason. A body can be edited after publish; a missing archive cannot.

**The trade, and where the checkpoint went.** Auto-publish means the release notes are GoReleaser's
generated commit list (the `^meta:`/`^spec:`/`^docs:` filters leave just the `batch-*:` lines — two
of them for v0.1.4) rather than the user's prose, which is now an edit made after publishing. It
also means a tag push is instantly public and effectively irreversible, so the human checkpoint did
not disappear, it moved earlier: from "publish the draft" to "should we tag at all", which is the
user's call every time.

**Verification.** `goreleaser check` passes against `@latest` v2 — matching what the workflow's
`version: "~> v2"` resolves to. (v2.5.0 rejects the config on `archives.formats`, a field added
later; that is a stale-pin artifact, not a config defect.) `go test ./...` green, `specflow verify`
clean on all 7 managed files.

**Follow-ups deferred.** Changelog prettification (`release.header`, `changelog.groups`) — the
generated body is adequate. Backfilling a v0.1.1 release. The end-to-end proof is the next tag push:
confirm it lands published with 6 assets and no manual step.

### Batch PR — Ledger pruning (`prune-ledgers`, the fourth procedure)
- Owner: claude
- Started: 2026-08-14 11:03
- Finished: 2026-08-14 11:33
- Commit: dd7a1e9

**What shipped.** A fourth procedure, `specflow/procedures/prune-ledgers.md`, plus its thin Claude
skill. `CLAIMS.md` had no pruning mechanism at all: `finish-batch` appended each completed entry to
`## Completed` and nothing ever reached `specflow/history/CLAIMS_DONE.md`. The archive shipped and
`AGENTS.md` documented it, but no procedure wrote to it. The only "archive when it grows long"
sentence lived *inside* `CLAIMS_DONE.md`, a file agents are told is reference-only and never open,
and it carried no threshold. So this was a missing step, not a skipped one, and no agent was at
fault for the bloat.

**The rule.** `## Completed` keeps its **5** newest entries; older ones move verbatim to
`CLAIMS_DONE.md`, newest at top. `## In progress` is never touched (a stale claim is a handoff
question, not something to archive). Retention is a **count, not a byte budget**, so pruning always
cuts on an entry boundary and two agents pruning independently reach the same result. Includes a
**catch-up pass** for installs that predate the procedure, and a `BUILD_QUEUE.md` sweep for sections
whose batch is already completed, dissolved, or absorbed.

**Grounded in real data, not a guess.** The first draft of the spec hard-coded N=5 on assumption and
the user stopped it. Measured against a long-running host install (`agents/kapara`): 26 completed
entries, 1,789 lines, 125 KB of `CLAIMS.md`, against a 206-byte untouched archive header, byte-for-
byte the same failure as this repo's own 18-entry ledger. Entry lengths there ran 35 to 126 lines,
median 61: a 3.6x spread, which is what settled count over byte budget. That install's queue was
healthy (549 lines of `BUILD_QUEUE_DONE.md` prove `finish-batch` step 4 is followed), so the sweep
is deliberately narrow.

**Design choices worth keeping.** (1) A separate procedure rather than an inlined finish-batch step,
because an overgrown ledger needs a many-entry catch-up unrelated to any batch finishing, and the
user wanted to run it by hand. (2) `finish-batch` delegates as **step 4a**, not a renumber:
"step-6 handoff" is a named concept across the Claude hook, the spec, and the README, and
renumbering would break every reference. (3) Rules live in the **procedure**, skill stays a thin
trigger, per the cross-agent invariant. Putting them in the skill would silently make pruning
Claude-only. (4) No stop-and-ask, unlike the spec 600-line cap: archiving is lossless and mechanical,
and `claim-batch` already resolves dependencies against either location.

**Go side.** `prune-ledgers` joins `specOnlyOmits` (procedure + skill) and `kit.QueueTokens` so a
mode leak is caught. `MANAGED` already covers `specflow/procedures` as a directory, so `upgrade`
picked the new file up with no change there.

**Verified against the real CLI, not assumed.** Built and ran it on this repo: `upgrade --dry-run`
first flagged the hand-edited copies as drifted (correct behavior, since their baselines no longer
matched), so the root copies were removed and rewritten by `upgrade`, which recorded the new
baseline hash. `verify` now lists four procedures. `go test ./...` uncached green, `gofmt`/`go vet`
clean. Two new tests lock the retention number, the no-stop-and-ask framing, the delegation, the
thin-trigger skill (with a line-count ceiling), and the spec-only omission. Dogfooded immediately:
this entry's own finish ran step 4a and archived 13 entries, taking `CLAIMS.md` from 18 completed
entries (~520 lines) to 5 (245 lines).

**Follow-ups deferred.** (1) Nothing enforces pruning. Like the rest of the protocol it is
honor-system until Batch E, and `verify --batch` is the natural home for a "CLAIMS over retention"
check. (2) Host installs need `specflow upgrade` plus one manual `prune-ledgers` run to catch up;
kapara has 21 entries to archive. (3) The kapara agent was never consulted: the message sat unread
behind a cross-session approval gate, so the numbers here were read from its files directly.
