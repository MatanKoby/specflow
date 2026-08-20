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

### Batch CE — Context economy + `config.check`
- Owner: claude
- Started: 2026-08-20 11:50

<!-- One entry per actively claimed batch. -->

## Completed

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

### Batch 4 — README badges + file-map
- Owner: claude
- Started: 2026-08-10 05:11
- Finished: 2026-08-10 07:09
- Commit: 041f1bc

**What shipped.** A full README rewrite, directed by the user ahead of a public launch post. Three
badges (CI, auto-tracking release, MIT). New structure: centered hero (badges + nav + a hand-written
ASCII `specflow init` console demo) → causal-chain pitch → five **Why** blocks → Install → Quick
start → How it works (four prose steps + a mermaid flowchart showing two agents claiming in
parallel) → file map → Agents → Who it's for → two `<details>` blocks (upgrade/status/verify,
spec-only) → How it differs → For AI agents.

**File-map deliverable.** An annotated file tree with `[owner]` tags per path, followed by the
existing ownership table. Tree for orientation, table for the contract.

**Agent-accessibility (user-added scope).** The README is now executable by an agent told "install
specflow here": (1) an HTML comment as the **first thing in the raw file** pointing agents at the
right section (invisible when rendered); (2) a `## For AI agents` section with seven ordered
imperative steps — confirm git repo, install binary, **ask the user which agents + which mode rather
than guessing**, run `init` non-interactively, show the diff and commit as `meta: install specflow`,
relay the Claude-Code handoff hook block with its rationale, then read `AGENTS.md` before any work;
(3) step 7 states plainly that the README is the pitch and `AGENTS.md` is the protocol, so an agent
does not try to work from the README.

**Verified, not assumed.** Ran a real `init --agents=claude,cursor` into a temp repo, which corrected
two drafted-from-memory errors: the install writes **16 files, not 11**, and `init` creates only
`spec/README.md` (no `architecture.md` / `research/`), so the tree now shows the near-empty `spec/`
and frames that as intentional. `--version`, `add-agent`, `status`, and `verify` output were each
confirmed before being described; the hook JSON is copied from live `init` output. Anchors, code
fences, and `go vet` / `go test` all check.

**Positioning choices.** Lead is continuity ("your agents forget between sessions; specflow makes
that not matter"), with token savings as the third Why block and its "where the spec covers the
ground" condition stated inline. Added an explicit *honest about enforcement* callout (the protocol
is written guidelines, nothing executable checks it — Batch E territory). No em dashes, per the
user's global writing rule.

**Follow-ups deferred.** (1) The animated demo (GIF/asciinema) stays deferred as the batch specified
— the ASCII console block is the designed visual standing in for it. (2) The *How it differs*
paragraph (Spec Kit / Kiro / OpenSpec / BMAD) was written from
`spec/research/2026-07-competitive-landscape.md` and **not re-verified against those projects' current
behavior**; worth a check before the launch post. (3) A 1280x640 GitHub social-preview image was
recommended to the user and not produced.

### Batch SZ — spec-file 600-line hard cap
- Owner: claude
- Started: 2026-08-06 10:14
- Finished: 2026-08-06 11:03
- Commit: 02e6a41

**What shipped.** The size rule in `spec-edit.md` was a nudge ("consider whether the next bite of
content wants its own file") that had never fired: this repo's largest spec file is 177 lines. Its
"~20k tokens" gloss was also about double the real figure (600 lines is roughly 10-11k tokens at
this corpus's 68 chars/line). Rewrote it as a **hard cap**: before an edit crosses the file's
current limit the agent stops and asks the user, presenting the file's **section headlines**, a
**single-concern** claim, and the read-cost warning **verbatim** ("The bigger a spec file is, the
more I read when I need even just a small chunk from it, so it's best the file is small in advance.
But, you're the boss."). It never asks the user to pick a number. A *keep* is recorded as a
`<!-- specflow:size-ok … -->` **first line** of the file with a UTC timestamp and the next
threshold, which advances **+200** each time (600 → 800 → 1000), so a waiver silences one threshold
and never the rule. **`archive.md` and `research/` are exempt** — both grow monotonically by design
and have no concern to split off, so the ask would have no good answer. Mirrored into this repo's
dogfood procedure and into the `spec-edit` SKILL.md (description + body), whose summary loads into
every Claude session. Design ref: `spec/architecture.md` → *Spec organization*.

**Verification.** `go test ./...` uncached green, `go vet` + `gofmt` clean. Three new tests, two of
them asserting the shipped procedure carries all four parts of the ask in both install modes (the
warning matched word-for-word over whitespace-normalized text, since a paraphrase would drop the
"you're the boss" that makes it a question rather than a lecture) and that the superseded nudge
wording and token gloss are gone. The **marker-collision test** proves `specflow:size-ok` is inert
to the region (`specflow:start\b`) and composition (`specflow:full-only:`) regexes on the two
surfaces that can actually parse it: inside a region (the procedure ships the literal as its
example) and above one (a waiver on a managed file). Confirmed it bites by swapping in the
near-miss token `specflow:start-ok`, which **does** match `specflow:start\b` (the `\b` fires before
the hyphen) and silently pushes AGENTS.md onto the drift path. A first draft of that test put the
waiver on `spec/architecture.md` and was **vacuous** — `spec/**` is user-owned, so no specflow
command ever parses it; the exposure is entirely in managed files.

**Known gap, surfaced not fixed.** `upgrade` refreshed the managed procedure here but **not** the
`spec-edit` skill stub: non-managed adapter files are create-once, and SL's `staleAdapterFiles`
repair only fires on a spec-only mode leak. So a *full*-mode install whose stub content merely went
stale still needs a hand-mirror (done here). Worth a batch if stub prose keeps changing.

**Note.** The stub template was rewrapped so its inline composition markers sit at line ends —
stripping them mid-line left a ragged short line in both rendered modes.

### Batch SL — spec-only mode leaks queue/batch language
- Owner: claude
- Started: 2026-08-06 07:17
- Finished: 2026-08-06 07:35
- Commit: 3b265e0

**What shipped.** The fix for a user-reported 0.1.2 defect: `init --spec-only` generated files that
instructed agents to use `BUILD_QUEUE.md`, `CLAIMS.md`, `claim-batch`, and `finish-batch`, none of
which the mode installs. (1) **Template gating** — the six templates with no mode markers now carry
`specflow:full-only` / `specflow:spec-only` pairs with *replacement* spec-only wording, not
deletion: the four adapter rule-files (cursor `.mdc` incl. its frontmatter `description:`, copilot,
bob, antigravity), `.claude/skills/spec-edit/SKILL.md` (incl. its YAML `description:`, which loads
into every session's skill listing), and `templates/base/spec/README.md`. (2) **`CLAUDE.md` guards
widened** past the step-6 hook paragraph to the protocol description, "three procedures", and the
claim/finish trigger bullets — it advertised the two skills `specOnlyOmits` had just skipped. (3)
**Mode-consistency check** (`kit.ModeLeaks` + `kit.QueueTokens`, shared with the tests): baseline
hashes are taken over the *rendered* region, so full-mode prose in a spec-only install matches its
own hash and reported clean — hashes prove a region is unmodified, never that it is
mode-appropriate. `verify` now scans managed regions for queue/claim tokens when the stamp records
spec-only, and prints the mode it validated. (4) **`upgrade` made a complete fix path**
(`staleAdapterFiles`): skill stubs and hooks are non-managed create-once files with no baseline, so
upgrade previously left an existing spec-only repo with a wrong `spec-edit` stub forever. The repair
keys off the mode leak itself, which proves the content is stale specflow text; full mode returns
nil from `ModeLeaks`, so a customized stub is never touched.

**Verification.** `go test ./...` uncached green, `go vet` + `gofmt` clean. Four new tests, written
before the fix and observed failing across all six files: a whole-install walk (covers a new
template the day it lands, rather than a hand-maintained file list), a `verify`-catches-leak test,
a legacy-upgrade-repair test, and a full-mode-stub-untouched test guarding the repair from being
destructive. A prose regex on the test side catches queue/claim wording that names no identifier
(`spec/README.md`'s "claiming a batch: read the queue entry first"), which the token list cannot
see. Exercised end-to-end against a **real v0.1.2 binary** built from `2951d78`: the new `verify`
reports 6 problems where 0.1.2 said "All good", and `upgrade` then clears every specflow-owned file.
Propagated to this repo's dogfood install (`upgrade` refreshed `CLAUDE.md`; `verify` clean).

**Known residue, by contract.** `spec/README.md` is user-owned, so `upgrade` must never rewrite it;
an existing spec-only install keeps its line 36 until the user edits it. Fixed at source, so new
installs are clean. Design ref: `spec/architecture.md` → *Install modes*.

**Follow-up.** Batch SZ (600-line cap) was unblocked by this batch and is next in the queue.
