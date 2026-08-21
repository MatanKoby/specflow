# Claims — Archive

Older completed entries archived from `CLAIMS.md`. Reference-only, newest archived batch at
the top. Append-only institutional memory: entries arrive verbatim and are never rewritten.

Written by `specflow/procedures/prune-ledgers.md`, which keeps the 5 newest completed entries in
`CLAIMS.md` and moves everything older here. Don't hand-move entries; run the procedure (Claude:
the `prune-ledgers` skill) so the retention rule stays consistent.

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

### Batch CH — Claude Code batch-boundary hook (opt-in)
- Owner: claude
- Started: 2026-07-12 11:39
- Finished: 2026-07-12 14:45
- Commit: 7b3ffdc

**What shipped.** The Claude-only deterministic backstop for the finish-batch step-6 handoff, on top
of the portable FH text. (1) **Hook** `templates/agents/claude/.claude/hooks/specflow-handoff-reminder.sh`:
a `PostToolUse(Bash)` script that gates cheaply on `git commit` in the command, then confirms via the
landed **HEAD subject** `^meta: complete batch-` (matching the landed subject, not the command text,
handles `-m`/`-F`/heredoc uniformly, proves the commit succeeded, and self-de-dups since the next
commit is `meta: claim …`), and emits `{"decision":"block","reason":…}` to halt the agentic loop and
feed the step-6 reminder back so the agent must act on it. Soft-deps on `jq`; **fails open** (exit 0)
if jq is absent so it can never break a commit. (2) **CLI** (`cmd/specflow/main.go`): prints an
opt-in paste-notice at the end of `init`/`add-agent` when the hook was just installed (claude + full
mode), recommending committed `.claude/settings.json` (no auto-merge — JSON has no marker-merge
path). (3) **Adapter relay** (`templates/agents/claude/CLAUDE.md`, full-only region): tells the agent
to surface the hook-setup step on install/upgrade instead of burying it. (4) **spec-only omits** the
hook (`internal/kit/kit.go` `specOnlyOmits`) — no batch boundary to backstop. (5) **`upgrade`
extended** to place newly-shipped non-managed adapter files create-once (scoped to the `agents/`
tree, so a user-deleted base file is never resurrected) — without this the backstop reached only
fresh inits, never existing installs.

**Manual prereq.** The hook is inert until registered in `.claude/settings.json`; the CLI/relay
surface the exact block. Activating it in *this* repo is deferred as a user opt-in (see follow-up).

**Verification.** `go test ./...` green, uncached (4 new tests: install+notice, add-agent incl.
no-dupe-on-re-add, jq-gated script behavior asserting block-on-match / silent otherwise, and
upgrade-adds-missing-hook incl. not-resurrecting a deleted base file). `go vet` + `gofmt` clean.
Exercised end-to-end: the shipped script (block/silent cases), `init`/`add-agent`/`upgrade` across
full + spec-only, and the embed-manifest test auto-covers the new file. Propagated to the dogfood
install via `specflow upgrade` (0.1.0→0.1.1, refreshed CLAUDE.md + added the hook here; `verify`
clean, no drift). Design ref: `spec/open-questions.md` → Quality / enforcement.

**Follow-up (deferred).** Registering the hook in this repo's own `.claude/settings.json` to activate
the backstop for agents working here — a user opt-in (committed vs local; loop-blocking behavior),
not bundled into the batch.

### Batch RF — Ship the research-flow convention
- Owner: claude
- Started: 2026-07-11 20:49
- Finished: 2026-07-11 20:57
- Commit: 9fbba41

**What shipped.** Made the lightweight research-note flow a shipped part of specflow, so a fresh
install carries the convention (not just this self-hosted repo). Three template edits, all
self-contained with **no queue/claim references** so spec-only inherits them: (1)
`templates/base/AGENTS.md` — the spec-discipline region's spec-edit pointer now names the optional
**pre-design research step** and the gate-free `spec/research/` home (dated snapshots, written on the
go, conclusions graduate into `open-questions.md` / `roadmap.md`); (2)
`templates/base/specflow/procedures/spec-edit.md` — a new **"Research notes"** section placed as the
pre-design exception to the archive rule, covering the gate-free / write-as-you-go / graduate-upward
lifecycle; (3) `templates/base/spec/README.md` — documents the optional `research/` sub-folder in the
file map. Design ref: `spec/workflow.md` → *Research notes*, `spec/research/README.md`. The dogfood
`spec/README.md` already carried the convention (it's the project's own spec, not a managed region),
so only the two managed files needed propagation.

**Verification.** `go test ./...` green (2 new tests: `TestResearchFlowConventionShipped` full-mode
and `TestResearchFlowInSpecOnly`, both asserting `init` ships the convention across AGENTS.md +
spec-edit.md + spec/README.md; the pre-existing spec-only banned-word test confirms the research
sections stay queue-free). `go vet` + `gofmt` clean. Propagated the two managed files (root
`AGENTS.md`, root `spec-edit.md`) to the dogfood copies via `specflow upgrade` (dry-run showed exactly
those two refreshing; applied; `specflow verify` clean, no drift). **Follow-up:** none — this is the
last of the queued research-flow work; together with Batch FH's step-6 rework it forms the user-facing
payload of the `v0.1.1` release cut immediately after this batch.

### Batch 3 — Broaden the test suite
- Owner: claude
- Started: 2026-07-11 19:52
- Finished: 2026-07-11 20:02
- Commit: 677b265

**What shipped.** Locked behavior beyond the existing file-existence smoke checks, in three areas.
(1) **Content assertions** on the generated full-mode `AGENTS.md` (`TestAgentsMdContentSections`):
its key sections (*Commit & push authority*, *File ownership*, *The work queue*, *The claims file*,
*The procedures*, *Commit message convention*, *Editing rules*), the **commit-grammar table** header
plus its `batch-N` / `meta: claim` / `meta: complete` / `spec:` rows, and that the procedures section
references the real `specflow/procedures/*.md` paths (not bare filenames). (2) **Adapter + picker
coverage**: the interactive agent picker driven over piped stdin — numeric multi-select
(`TestInitInteractivePicksAgentsByNumber`, "3,4,5" → copilot/bob/antigravity, unpicked agents absent)
and the "a" all-shortcut (`TestInitInteractiveAllShortcut`) — plus the **`--all`** flag
(`TestInitAllFlagWiresEveryAdapter`) asserting each of the five adapters' instruction files is
written, carries the managed region markers, and has a baseline hash, then that a follow-up
`upgrade` stays a clean no-op. This is the first coverage of the **copilot / bob / antigravity**
adapters. (3) An **embed-manifest test** in the root package (`templates_test.go`) that walks the
on-disk `templates/**` tree and asserts every file is embedded **byte-for-byte**, explicitly counting
dot-path templates so the `//go:embed all:` dotfile footgun can't regress unnoticed.

**Verification.** `go test ./...` green (4 new tests: 3 in `cmd/specflow/main_test.go`, 1 in the new
root-package `templates_test.go` — the root package had no tests before). `go vet` + `gofmt` clean.
Proved the embed guard actually bites: temporarily dropping `all:` from the `//go:embed` directive
made the manifest test fail on exactly the dropped dotfiles (Claude skills, cursor/copilot adapters),
then reverted to green. **Note:** the embed-manifest test lives in a new root-package file rather than
`cmd/specflow/`, because that is where the `embed.FS` and `Templates()` are — the batch's file list
allowed for splitting tests out. **Deferred (out of scope):** the batch's suggested `internal/kit/`
unit-test split was not done — the behavior is fully covered end-to-end through the built binary, so
a redundant unit layer wasn't warranted now.

### Batch FH — finish-batch step-6 handoff rework
- Owner: claude
- Started: 2026-07-11 19:35
- Finished: 2026-07-11 19:43
- Commit: 2962ae1

**What shipped.** Reworked step 6 of `finish-batch.md` so the end-of-batch context handoff is hard
to skip. Root cause (from a retro): step 6 was the only finish step that produced no artifact, so a
skip was invisible and cheap under throughput pressure and got rationalized away as "noise." The
rewrite (a) states the payoff to the user (cheaper next batch, more reliable next batch, a decision
point), (b) names and refutes the "it's noise" excuse, and (c) requires a fixed terminal handoff
line so an omission is self-evidently non-compliant. Step 7 now clarifies that a user's "continue"
authorizes the next claim but does not waive the step-6 line. Edited the canonical template
(`templates/base/specflow/procedures/finish-batch.md`) and propagated to this repo's dogfood copy
via `specflow upgrade` (stamp rebaselined; `verify` clean). Portable across all agents (text only);
the Claude-Code deterministic backstop is queued separately as Batch CH.

**Verification.** `go test ./...` green, `go vet`/`gofmt` clean. `upgrade --dry-run` showed exactly
one file refreshing; applied; the dogfood copy is back in sync with the template; `specflow verify`
reports all regions intact (no drift).

### Batch 5 — `--dry-run` (preview)
- Owner: claude
- Started: 2026-07-05 04:40
- Finished: 2026-07-05 05:14
- Commit: 7f5f70b

**What shipped.** A **`--dry-run`** flag on `init` and `upgrade` that prints the exact planned file
operations and exits **without touching disk**. **`init --dry-run`** reuses `PlanInit` and renders
*would create / would inject (content preserved) / already-wired / would skip*; it's always
non-interactive (no agent/consent prompts) and previews the default agent (claude) when `--agents`
is omitted, honoring `--spec-only`/`--all`. **`upgrade --dry-run`** renders *would refresh / add /
migrate (→ `.specflow-bak`) / drift (→ `.specflow-new`, not overwritten)*, and *already current* when
there's nothing to do. **Refactor (no behavior change):** the per-file upgrade classification is now
a shared `decideUpgrade` + `upgradeDecisions` pair consumed by both the apply path (`Upgrade`, kept
byte-for-byte in behavior) and the new read-only `PlanUpgrade` — so the preview and the real run can
never diverge. Help text for both commands documents the flag.

**Verification.** `go test ./...` green (6 new dry-run tests + the full existing upgrade suite,
which guards the refactor). `go vet` + `gofmt` clean. Manually confirmed a fresh-dir `init --dry-run`
and an `upgrade --dry-run` over a drifted+stripped install write nothing (no files, no sidecars, stamp
unchanged).

**Milestone.** This is the **last v0.1 batch** — v0.1's definition-of-done (roadmap.md) is now met in
code. Next: cut the real `v0.1.0` tag (the release pipeline was proven in Batch G2).

### Batch 2 — `specflow status`
- Owner: claude
- Started: 2026-07-05 04:23
- Finished: 2026-07-05 04:39
- Commit: 3f67292

**What shipped.** A read-only **`specflow status`** that orients a user/agent at a glance, writing
nothing. **`kit.Status`** (in `internal/kit/kit.go`) assembles a snapshot from the stamp + repo:
**kit version** (stamp `kitVersion` vs. the running binary, with a *run upgrade* hint on mismatch),
**install mode**, **wired agents**, the **commit/push levers**, **active claims** parsed out of
CLAIMS.md's In-progress section (each `###` heading paired with its `Owner:`; `none`/unset renders as
*unassigned*), the **un-done batch count** (a `^## Batch` count over BUILD_QUEUE.md, since done
batches are removed from that file), and a **drift flag** listing any managed region whose hash no
longer matches its recorded baseline (the same test `verify`/`upgrade` use). The CLI
(`cmd/specflow/main.go`) renders an aligned label/value block (NO_COLOR-aware via the shared paint
helpers), exits non-zero when specflow isn't installed, and marks the queue **n/a** for a spec-only
install (no BUILD_QUEUE.md/CLAIMS.md). Registered in `dispatch` + top-level/`--help` usage.

**Verification.** `go test ./...` green (7 new tests: fresh install, active claims incl.
owner/unassigned, drift flag, version mismatch + upgrade hint, spec-only queue-n/a, not-installed
non-zero exit, and `--help`). `go vet` + `gofmt` clean. Manually exercised across all scenarios.

**Note (not part of this batch).** A parallel session's git worktree appeared at
`.claude/worktrees/` during the batch; a `.gitignore` entry for it would prevent accidental commits
(flagged for the user, left out of scope here).

### Batch 1 — `specflow add-agent <name>`
- Owner: claude
- Started: 2026-07-03 09:41
- Finished: 2026-07-05 04:18
- Commit: fa3aa3d

**What shipped.** A new **`specflow add-agent <name> [<name>...]`** command that wires another
agent's adapter into an already-initialized repo. **`kit.AddAgent`** (in `internal/kit/kit.go`) reads
the stamp, no-ops if the agent is already in `config.agents` (reported as *already installed*), then
walks `agents/<key>/` and, per file: **creates** a missing adapter file (rendered for the install
mode); **injects** specflow's marker region into an existing **instruction file** (CLAUDE.md etc.),
preserving the user's content, or **leaves it as-is** when it already carries a region or points at
AGENTS.md; and **skips** any other specflow-owned file already present. It then records the agent in
`config.agents` and refreshes the managed-region baselines for the full agent set (so `upgrade`
tracks the new instruction file). **Mode-aware:** a spec-only install doesn't gain the
claim/finish skills (`specOnlyOmits` filter). The CLI (`cmd/specflow/main.go`) validates every name
up front against the known-agent list (clean error + non-zero exit on a typo), guards the
not-installed case, prints a per-agent summary, and ends with the **review-then-commit** handoff —
**it never commits** (same discipline as `init`). Registered in `dispatch` + top-level/`--help`
usage.

**Verification.** `go test ./...` green (8 new tests: adapter+stamp+managed, multi-add +
already-present no-op, brownfield inject, already-wired left-as-is, spec-only skill omission,
unknown/not-installed guards, post-add `verify`, and `--help`). `go vet` + `gofmt` clean. Manually
exercised end-to-end across all scenarios.

**Follow-ups deferred.** `remove-agent` (still *decision pending* in `open-questions.md` → CLI /
upgrade behavior) is out of scope here.

### Batch G2 — Go release + install pipeline
- Owner: claude
- Started: 2026-06-24 08:25
- Finished: 2026-07-03 09:35
- Commit: 50c59aa

**What shipped.** Zero-runtime distribution for the Go binary. **GoReleaser** (`.goreleaser.yaml`,
v2) cross-compiles a **5-target matrix** (linux amd64/arm64, darwin amd64/arm64, windows amd64;
windows/arm64 ignored) as `-trimpath` + `CGO_ENABLED=0` static builds with `main.version` injected
from the tag; `tar.gz` archives (`.zip` on Windows) named `specflow_<version>_<os>_<arch>` bundle
LICENSE + README, alongside a `sha256` `checksums.txt`. A **`Release` workflow**
(`.github/workflows/release.yml`) fires on any `v*` tag → `goreleaser release --clean` with the
auto `GITHUB_TOKEN` (`contents: write`) → a **draft** release (`prerelease: auto`; review then
publish from the UI). An **`install.sh`** `curl … | sh` front-end detects OS/arch, resolves the
latest published release (or `SPECFLOW_VERSION`), downloads + checksum-verifies + extracts the
matching archive, installs to `/usr/local/bin` (sudo) or `~/.local/bin`, and honors `NO_COLOR`/
non-TTY. README install section rewritten to the binary flow (+ `go install …@latest` fallback).

**Verification.** `goreleaser release --snapshot` produced all 5 archives + `checksums.txt` with the
exact names `install.sh` expects; install.sh's checksum-grep + extract path was replicated locally
(binary runs, injected version correct). A pushed throwaway `v0.0.1-test` tag ran the workflow
**green** ([run 28645497812](https://github.com/MatanKoby/specflow/actions/runs/28645497812)) and
produced a draft release with all assets — draft correctly hidden from the public API, its assets not
anonymously downloadable. Test tag + draft deleted afterward.

**Deferred (post-v0.1).** Homebrew tap (a `brews:` block slots into `.goreleaser.yaml` when the tap
repo lands), npm wrapper (`npx specflow`), Scoop/Winget — see `open-questions.md` → Distribution. The
full public `curl … | sh` path is only exercisable against a *published* release, so it stays
unproven until the real `v0.1.0`.

### Batch SO — Spec-only install mode (`--spec-only`)
- Owner: claude
- Started: 2026-06-24 06:33
- Finished: 2026-06-24 07:05
- Commit: 93519b0

**What shipped.** `specflow init --spec-only` — a lighter install that keeps only the **spec
discipline** and omits the batch/claim machinery. It installs `AGENTS.md` (spec sections), `spec/`,
the **spec-edit** procedure + skill, the stamp (mode recorded), and the selected agent stubs; it
omits `BUILD_QUEUE.md`, `CLAIMS.md`, the `claim-batch`/`finish-batch` procedures, those two skills,
and the `specflow/history/` archives. **Section composition (one source, no forks):** `AGENTS.md`
and `spec-edit.md` now carry `specflow:full-only` / `specflow:spec-only` marker pairs *inside* their
managed region; `renderBody` keeps the pair matching the install mode (stripping its markers) and
drops the other pair whole. Own-line markers (paragraphs, list items, table rows) and inline clause
markers are handled separately so a drop never splits a markdown table or leaves a stray blank line;
runs of 3+ newlines collapse back to one blank line. The baseline hash is taken over the **rendered**
region, so drift detection works per-mode. The tag token (`specflow:full-only:…`) never collides with
the region token (`specflow:start`/`end`). **Mode plumbing:** `{{MODE}}` placeholder in the config
template → `config.mode`; `mode` threaded through `PlanInit`/`ApplyInit`/`classifyInit`/`initFiles`/
`managedEntries`/`computeManaged`/`recordManaged`/`fillStamp`; `upgrade` and `verify` read the mode
from the stamp, re-render the managed region for it, and filter the managed/placed-file sets via
`specOnlyOmits`. CLI: `--spec-only` flag, mode-aware Phase-2 / review-handoff text, and updated
`init --help` + top-level usage. **Verification:** `go test ./...` green (six new spec-only tests:
omissions + mode stamp, batch-free managed files with no leftover tags, full-mode completeness, clean
spec-only `upgrade`, spec-only `verify`); `gofmt`/`go vet` clean. Self-hosted: ran `upgrade` on this
(full-mode) repo — root `AGENTS.md` refreshed to the composed wording, `spec-edit.md` byte-identical,
`verify` clean, second `upgrade` a no-op. **Follow-ups deferred (per architecture, in scope for later):**
the graduation path spec-only → full (`enable-batching` / re-run); and the per-agent stubs + the
`spec-edit` skill description are **not** section-composed yet, so a spec-only Claude install still
ships a `CLAUDE.md` whose pointer mentions queue/claim + the two uninstalled skills (cosmetic — the
authoritative composed `AGENTS.md` is correct).

### Batch BI — Brownfield-aware `init` (inject-with-consent, review handoff)
- Owner: claude
- Started: 2026-06-23 17:08
- Finished: 2026-06-24 06:12
- Commit: 6d14a90

**What shipped.** `init` is now brownfield-aware and non-destructive, in two consent-gated phases:
Phase 1, for each target file that **already exists**, injects specflow's marker-wrapped region at
the top (existing content preserved); Phase 2 explains, then creates, the specflow-owned files.
`init` tracks its **own** created/modified/declined list and ends with a "review `git diff`, verify
nothing was damaged, then commit" handoff — it **never commits**; non-interactive runs proceed and
point at `git status` (no `--yes` flag). All five per-agent instruction files (`CLAUDE.md`, copilot,
cursor, bob, antigravity) are now **marker-wrapped and managed**, so `upgrade` refreshes their region
— but only for installed agents. **Idempotent injection:** refresh an existing region, and skip with
an "already wired" note when a per-agent file already references `AGENTS.md` (`AGENTS.md` itself
excepted — it must *carry* the protocol). **Tier-aware notices** on declined/missing pieces (Tier 1
`AGENTS.md`/procedures → can't work properly; Tier 3 per-agent → that agent isn't auto-wired, works
once its file points at `AGENTS.md`). Added **per-subcommand help** (`init`/`upgrade`/`verify --help`)
and a basic **`specflow verify`** install-integrity check (config valid, Tier-1 present with intact
regions + drift warnings, Tier-3 present/wired; exits non-zero on a Tier-1 problem; `verify --batch`
stubbed until Batch E). **Relocated the `_DONE` archives** to `specflow/history/` and updated every
path reference (`AGENTS.md`, procedures, queue/claims headers, the finish-batch skill, README,
tests); templates moved in lockstep so fresh installs place them there. Two bugs fixed along the way:
a shared package-level stdin reader (the per-prompt `bufio.NewReader` dropped buffered input across
prompts), and a guard so `upgrade` never adopts/overwrites an untouched brownfield file (region
absent **and** no recorded baseline). Executed as internal `batch-BI:` chunks — key commits
`ce60443` (per-agent files managed), `b766da2` (two-phase init), `b57bd50` (idempotent injection +
tier-aware decline), `ec31a2d` (per-subcommand help), `f583f39` (`verify`), `6d14a90` (`_DONE`
relocation). ~22 tests pass; `gofmt`/`go vet` clean; self-host `upgrade` idempotent; brownfield-init
/ verify / fresh-init smokes pass. **Follow-up:** Batch SO (spec-only mode) is next and shares the
template/section + marker work.

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

<!-- Recent finishes, newest first. Bounded working set: the 5 newest stay here and
     prune-ledgers moves the rest to specflow/history/CLAIMS_DONE.md. -->

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
