# Architecture

## What specflow is

A single statically-compiled Go binary (`cmd/specflow`) plus a `templates/` tree, with no
third-party dependencies. The CLI scaffolds the protocol files into a host repo and refreshes them
over time. There is no runtime library the host imports — specflow's output is **plain markdown +
git**, which is what makes it cross-agent and language-neutral.

```
specflow/                   (this repo — the tool)
  cmd/specflow/             the CLI entry: init, upgrade, --version  (+ planned: add-agent, status, --new-batch, verify)
  internal/kit/             scaffold (init) + non-destructive refresh (upgrade): markers, region hashing, the stamp
  templates.go              embeds templates/ into the binary (//go:embed all:templates)
  templates/
    base/                   files every install gets (AGENTS.md, queue/claims skeletons, spec/, specflow/)
    agents/<agent>/         per-agent adapter stubs, copied when that agent is selected
  cmd/specflow/main_test.go builds + drives the binary against temp repos, asserts behavior
```

## Cross-agent model

`AGENTS.md` is the **universal base** every agent reads — it carries the full protocol. Per-agent
**stubs** are generated only for agents with their own mandatory config home: Claude Code
(`CLAUDE.md` + native skills), Cursor (`.cursor/rules`), Copilot (`.github/copilot-instructions.md`),
IBM Bob (`.bob/rules/`), Google Antigravity (`.agents/rules/`). Bob and Antigravity also read
`AGENTS.md` natively, so their stubs are reinforcement, not load-bearing. The three procedures live
once in `specflow/procedures/`; Claude's skills are thin triggers that point at them (single source).

specflow is deliberately **not** packaged as a Claude plugin — a plugin is Claude-only, and the
requirement is that every agent honors the same protocol.

## The host-repo file-contract

`init` writes, and the ownership boundary is the load-bearing decision:

| Path | Owner | On `upgrade` |
|---|---|---|
| `AGENTS.md`, `specflow/procedures/**` | **specflow** (mechanism) | managed region refreshed — **never user/agent text** |
| `specflow/config.json` | specflow (config + state) | version bumped |
| `BUILD_QUEUE.md`, `CLAIMS.md` (root) · `specflow/history/{BUILD_QUEUE_DONE,CLAIMS_DONE}.md` (archives) | user/agents (state) | untouched |
| `spec/**` | user (content) | untouched |
| per-agent instruction files (`CLAUDE.md`, `.github/copilot-instructions.md`, `.cursor/rules/…`, …) | **specflow** (a marker-wrapped region) + user (content outside) | region refreshed — **never user text** |

The `upgrade` column is about `init` / `upgrade` only. Three later verbs (`next`, `claim`,
`finish`) do write the state files on the agent's behalf: see *Queue verbs* below.

**specflow owns the mechanism; the host owns content and state.** Hard invariant: **both `init` (when
it injects into a file that already exists) and `upgrade` are non-destructive — they never remove or
overwrite text authored by a user or another agent**, in any file. They write only specflow's own
marker-delimited region; everything outside is preserved. `init` additionally **never writes without
consent and never commits** (see below).

## Ledger lifecycle — the active state files stay bounded

`BUILD_QUEUE.md` and `CLAIMS.md` are read on every batch, so both are **bounded working sets**, not
growing logs. Each has a matching archive under `specflow/history/`. Mechanics live in
`specflow/procedures/prune-ledgers.md`.

The two are bounded by different rules, because they answer different questions:

- **`BUILD_QUEUE.md` — zero retention, enforced at finish.** A done batch is not queue content at
  all: `finish-batch` deletes its section outright and collapses it to a one-paragraph summary in
  `BUILD_QUEUE_DONE.md`. That stays in `finish-batch.md` rather than moving here, because removing a
  finished batch is part of *completing* it, not deferrable housekeeping. Pruning only sweeps up what
  leaked past: sections for batches already in `CLAIMS.md` `## Completed`, and batches dissolved or
  absorbed into another.
- **`CLAIMS.md` — retention of 5, enforced by pruning.** Recent completions *are* live claims
  content: an agent reads back a few entries for continuity. The **5** most recent completed entries
  stay; older ones move to `CLAIMS_DONE.md`, newest at top, unedited.

Retention is a **count, not a line or byte budget.** Measured across 26 completed entries in a
long-running install, entries ran 35 to 126 lines (median 61) — a 3.6x spread, so a byte budget
would keep a nondeterministic number of batches and could sever an entry mid-record. Entries are
self-contained, so cutting on an entry boundary matters more than an exact ceiling. A count is also
deterministic: two agents prune to the same result.

Pruning is **not** gated behind a stop-and-ask, unlike the spec-file cap below. That cap asks
because splitting a spec file is lossy and needs judgment about concerns. Archiving a claim is
lossless and mechanical — the entry moves one file away, unedited — and dependency resolution
already accepts either location: `claim-batch` resolves a dependency against `CLAIMS.md`
`## Completed` **or** `CLAIMS_DONE.md`. An ask with no judgment behind it is just a prompt.

Pruning is its own procedure, not a step inlined into `finish-batch.md`, for two reasons: a ledger
that is already overgrown needs a **catch-up pass** that archives many entries at once, unrelated to
any batch finishing; and the user may want to run it by hand. `finish-batch` therefore *delegates*
to it. Per the cross-agent rule above, the rules live in the **procedure** (every agent reads it) and
the Claude skill stays a thin trigger — pruning must not become a Claude-only feature.

**The check runs at both ends of a batch, not only at finish.** An agent that prunes only on the way
out still reads the overgrown ledger on the way in, and the read is the cost. `claim-batch` therefore
tests the same condition before claiming, against the retention rule already stated above: more than
five entries under `CLAIMS.md` `## Completed` means prune first. No new threshold is introduced, and
the test is free — it is the heading grep the claim already runs.

Both archives are append-only institutional memory, reference-only, and never rewritten — the same
posture as `spec/archive.md`, and likewise exempt from any size cap.

## Context economy — the read side of the ledger

The lifecycle above bounds what the state files *hold*. This bounds what an agent *reads out of*
them, a separate cost and a larger one. Instrumented over one long batch in a specflow-managed repo,
file reads were 45% of all tool calls and 88% of context spend, and the biggest single read was a
`cat` of `CLAIMS.md` to answer a question the headings alone settle.

Three rules, written into the **procedures** rather than left to agent judgment, because "read less"
only becomes actionable when the procedure names the cheap read:

- **Slice, never `cat`.** Eligibility questions (is Batch N claimed? which batches are un-done?) are
  answered by headings. Measured on this repo: `grep -E '^###|^- Owner:' CLAIMS.md` returns 419 bytes
  against 17.2 KB for the whole file (41x), and `grep '^## Batch' BUILD_QUEUE.md` returns 224 bytes
  against 7.5 KB (33x). Both ledgers are read 3 to 5 times per batch (claim, finish, prune), so
  full-reading them is the largest recurring context cost in the protocol. Read headings first, then
  slice the one section you need. The same applies to a `spec/` file: its `#` headings, then the
  matching section.
- **One check command, not three.** See `config.check` under *Config & state*.
- **Batch independent reads, and never re-read to confirm your own write.** A failed edit reports an
  error; a `grep` that confirms it worked buys nothing.

This is deliberately **not** a smaller retention count. Retention is a count of 5 for the reasons
given above, and entries at that size are what a resuming agent needs. The waste was reading 17 KB
to answer a 400-byte question, not keeping 5 entries.

## Spec organization — concern-per-file and the 600-line cap

`spec/` is organized concern-per-file, with sub-folders and a per-folder `README.md` index as it
grows. The **primary** reason to split a file is that it has started holding **two concerns** —
that fires long before any line count, and it is what the spec-edit procedure's concern-matching
rule exists to catch.

**600 lines is a hard cap, not a nudge.** It backstops the case where a file quietly accumulated a
second concern and nobody noticed:

- When an edit would push a live design file past its current limit, the agent **stops and asks the
  user** rather than deciding for itself. It presents the file's **section headlines** (so the user
  can judge where a split would fall) and states whether it believes the file still holds a
  **single concern**.
- The ask always carries the cost, in these words: *"The bigger a spec file is, the more I read when
  I need even just a small chunk from it, so it's best the file is small in advance. But, you're the
  boss."*
- The agent **never asks the user to pick a number.** The only question is split or keep.
- **If the user says keep**, the agent writes a waiver marker as the **first line of the file**
  recording that the user decided it, the timestamp (UTC), and the next threshold:
  `<!-- specflow:size-ok - user approved this file over 600 lines on YYYY-MM-DD HH:MM UTC; next check at 800. -->`
  The threshold advances by **+200 lines** each time, so the next ask lands at 800, then 1000. A
  waiver is never permanent silence.

**Exempt: `archive.md` and `research/`.** Both grow monotonically by design — `archive.md` is
institutional memory and research notes are dated snapshots the archive rule already forbids
rewriting — so neither has a concern-split available and the cap would only produce an
unanswerable prompt.

The 600 figure is load-bearing elsewhere: it is the assumption behind *no spec Q&A* (specs fit in
context, so asking the invoking agent is free) — see `research/2026-07-competitive-landscape.md` #7.

## Install modes — full vs spec-only

`init` installs one of two modes (the user picks at init; recorded in the stamp):

- **Full** (default) — the whole protocol: spec → queue → claim → build.
- **Spec-only** (`--spec-only`) — just the **spec discipline**: agents create / update / organize
  `spec/` (concern-per-file hierarchy, splitting large files, archiving stale content) under gate 1
  (propose → user approves), with **no** queue / claim / batch machinery. A lighter on-ramp; can
  graduate to full later.

Spec-only writes `AGENTS.md` (spec sections only), `spec/`, the **spec-edit** procedure, the stamp,
and the selected agent stubs; it omits `BUILD_QUEUE.md` / `CLAIMS.md` / `claim-batch` / `finish-batch`
and their skills.

**One source, composed — not two forks.** Every template that ships in both modes is
**section-composed**: its managed region is built from tagged sub-sections, and spec-only **omits the
batch/claim sections** at render time. There is a single template (organic, no drifting variants);
the mode decides which sub-sections are emitted, and the baseline hash is taken over the *rendered*
region.
- `AGENTS.md` sub-sections: **spec discipline** (always) · **commit/push authority** (always) ·
  **batch & claim** (full only).
- `spec-edit.md` sub-sections: the spec-organization core (always) · the *"a decision also goes to
  the queue"* persistence step (full only).
- **Every per-agent instruction file** (`CLAUDE.md`, `.cursor/rules/specflow.mdc`,
  `.github/copilot-instructions.md`, `.bob/rules/specflow.md`, `.agents/rules/specflow.md`) and the
  **`spec-edit` skill stub**: the one-paragraph model description, the auto-trigger list, and the
  commit grammar are all mode-dependent. Spec-only needs *replacement wording*, not just deletion —
  an adapter must still describe the protocol it did install.
- `spec/README.md` ships as user-owned content, so its reading-order line is composed at render
  time and cannot be corrected retroactively by `upgrade` (see *Ownership* above).

**Composition scope is a hard invariant: a generated file may never name machinery its own install
mode omits.** An install that points agents at `BUILD_QUEUE.md`, `CLAIMS.md`, `claim-batch`, or
`finish-batch` when spec-only did not create them is a defect, not cosmetic drift. The spec
sub-sections are **self-contained** — they never reference the queue/claims, so spec-only reads
cleanly.

`verify` enforces this: per-region baseline hashes prove a region is *unmodified*, but they cannot
prove it is *mode-appropriate* (each mode hashes its own rendering, so a full-mode paragraph
wrongly shipped into a spec-only install still matches its own baseline). A **mode-consistency
check** therefore scans managed regions for queue/claim tokens whenever the stamp records
`mode: spec-only`, and reports them as drift.

## init / upgrade

- **`init`** — interactive (or `--agents=…` / `--all`). It **requires a git repository** — specflow's
  safety net is the no-commit + `git diff` review, so without git there is no undo; if the cwd is not
  in a git work tree, `init` stops with *"specflow only works in git repositories — run `git init`
  first"* and writes nothing. It **does not skip or silently overwrite**; it plans, gets consent,
  writes, then hands off for review — and **never commits**:
  1. **Pick agents.** The universal `AGENTS.md` is always included; per-agent instruction files are
     added for the selected agents. Everything below happens only after this selection.
  2. **Phase 1 — files to modify (inject).** specflow's contribution to a shared agent file is a
     marker-wrapped *region*. For every target file that **already exists** (`AGENTS.md` always; each
     selected agent's instruction file as applicable), `init` shows *what* region it will inject and
     *why*, then asks **once** to allow the whole set (batched consent, not per-file). Existing
     content is preserved — the region is inserted between markers (the non-destructive model,
     applied at init time). `init` is **idempotent**: for a per-agent file it injects only when
     needed — if the file already carries specflow's marker region it is refreshed, not duplicated,
     and if it already references `AGENTS.md` (a markdown link, a mention, or an `@AGENTS.md` import)
     `init` reports it as already wired and adds no second pointer. If the user declines, `init` still
     writes everything else: a declined
     **per-agent** file just means that agent isn't auto-wired — it works as soon as its instruction
     file points at `AGENTS.md` (the single source), which is exactly what the injected region adds;
     a declined **`AGENTS.md`** region (or a missing procedure / stamp) means specflow can't work
     properly, which `specflow verify` reports.
  3. **Phase 2 — files to create.** `init` then explains the specflow-owned files it will create
     fresh (`BUILD_QUEUE.md`, `CLAIMS.md`, `spec/`, `specflow/**`, the skills) and why.
  4. **Write** the approved modifications + creations and fill the stamp, while tracking its **own**
     list of every file it created or modified (not derived from `git` — the tree may carry
     unrelated changes).
  5. **Hand off for review.** Print the tracked created/modified list and tell the user they can
     review `git diff` and remove anything unwanted — with the caveat that **specflow may be limited
     or not work properly in some contexts** if required pieces are removed — then commit when
     satisfied, ideally as the install's **own commit** (e.g. `meta: install specflow`) before any
     batch work, so a later batch-enforcement check sees a clean tree. Point them at **`specflow
     verify`**, which re-checks install integrity (a required
     file missing, or present but missing its managed block). Re-init is guarded (a stamp already
     present → bail).

  **Non-interactive `init`** (`--agents=` / `--all`, for agents and CI) skips the prompts and
  **proceeds** with the modifications, then notifies the user to check `git status` / `git diff` and
  approve — safe because `init` never commits (no separate `--yes` flag needed).
- **`upgrade`** — refreshes specflow's managed mechanism to the installed kit version and bumps the
  stamp. **Hard invariant: `upgrade` never removes or overwrites text authored by a user or another
  agent, in any file.** Each managed file wraps its generated content in marker-delimited regions
  (`<!-- specflow:start … -->` … `<!-- specflow:end -->`); the start marker carries a "do not edit
  inside" note (an HTML comment — invisible in rendered markdown). Markers are matched by their
  `specflow:start` / `specflow:end` **token**, not exact text, so the note can evolve without
  breaking parsing or forcing a migration (a clean `upgrade` canonicalizes a file's markers to the
  template's current wording). `init` records a SHA-256 of each region in the stamp's `managed` map.
  On `upgrade`:
  - **Clean region** (on-disk hash matches the stored baseline) → only the content *between* the
    markers is replaced; everything outside is preserved verbatim, and the baseline is re-recorded.
  - **Drifted region** (hash differs — someone edited inside the markers) → left **untouched**; the
    fresh version is written to a `<file>.specflow-new` sidecar and reported, so nothing is clobbered.
  - **Pre-marker file** (an install predating markers) → migrated: backed up to `<file>.specflow-bak`,
    then rewritten with markers.

  **Two managed tiers, one drift contract.** The rules above govern files that carry *both*
  generated and human text (`AGENTS.md`, the agent instruction files, the procedures), so only the
  marker-wrapped region is specflow's. The **adapters** — the Claude skill stubs and the
  finish-batch handoff hook — are wholly generated: they carry no markers because there is no user
  prose to protect. They were **create-once** through v0.1.5, and that was a defect: a corrected
  skill stub or hook never reached a repo that already had the file, so every existing install sat
  on whatever shipped the day it was installed, no matter how many times it upgraded. They are now
  managed as **whole files** — the same contract, with the hash taken over the entire rendered file
  instead of a region:
  - **Clean file** (hash matches the recorded baseline) → replaced with the current version; the
    baseline is re-recorded.
  - **Drifted file** (hash differs — the user edited it) → left untouched; the fresh version is
    written to `<file>.specflow-new`, exactly as for a drifted region.
  - **No baseline** (an install predating whole-file management) → **one-time adoption**: a file
    already identical to the current version is adopted silently, and any other is backed up to
    `<file>.specflow-bak` and replaced. This is the only way an existing install can converge —
    without a baseline there is nothing to distinguish a stale copy from an edited one, and the
    backup means the choice costs the user nothing. From that upgrade on, the file has a baseline
    and the two rules above apply normally.

  Mode consistency still overrides both: an adapter naming machinery its mode omits is replaced
  regardless of drift, because the leak itself proves the content is stale specflow text.

  `verify` and `status` read the same baselines. `verify` covers the adapters alongside the region
  files — a deleted, truncated, or hand-mangled skill stub or hook is reported, where before it
  passed clean. `status` separates two things a single "drift" line used to conflate: **drifted**
  (you edited it — `upgrade` will not touch it) and **stale** (specflow moved and this file didn't
  — `upgrade` will refresh it). Without that split a stamp matching the binary reads as "everything
  is current" while a skill stub is several versions behind.

## Config & state — one file

**One** machine file, in specflow's own folder (**not** the repo root): `specflow/config.json`
(renamed from the legacy `.spec-batch.json`). It holds both **user config** and **internal state** —
written by `init`, updated by later calls (`add-agent`, lever/mode changes):

- **`config`** — the user's choices: `agents`, `mode` (`full` | `spec-only`), `commit`
  (`agent` | `user`), `push` (`agent` | `user`), `check` (the repo's single check command, or
  empty when not configured).
- **internal** — `kitVersion`, `schemaVersion`, `initializedAt`, `upgradedAt`, and the `managed`
  map (per managed file → SHA-256 of its rendered region, or of the **whole** rendered file for the
  marker-less adapters) that powers the drift detection above. The two kinds of entry never collide
  because the file sets are disjoint: a path is either region-managed or whole-file-managed, and
  which one it is comes from the kit, not the stamp.

**`check` is the repo's one check command**, asked at `init` and skippable; empty means not
configured, and the procedures then say nothing about it. It is named `check` rather than `verify`
so it never reads as the `specflow verify` command, which checks install integrity and is a
different thing. It exists because every repo has a check sequence specflow currently has no place
to name: an agent that does not know the repo runs the type-checker, then the linter, then the
tests as three separate calls, and rediscovers that split on every batch forever. One recorded
string (`npm run verify`, `make check`, `cargo test`) collapses it to a single call. specflow
neither validates nor executes it: it is a string the procedures quote back, so a wrong value costs
one failed command and is fixed by editing `config.json`, not by a migration.

Living under `specflow/` keeps the repo root clean — the convention good tools follow (a dedicated
folder, not yet another root dotfile). A human-readable mirror (`specflow/config.md`) is planned with
Batch W. Procedure prose is *instructions, not data*, so a refresh replaces specflow's managed region
(never user/agent text) with no migration; `schemaVersion` gates the rare case where the **format**
changes — no migration runner until a format actually breaks. *(The rename + `config` block is a
scheduled code change, folded into the v0.1 work.)*

## Queue verbs — the CLI as the agent's hands

`init` and `upgrade` never touch the state files. Three separate verbs do, on the agent's behalf and
only when the agent asks:

- **`specflow next [--json]`** — read-only. Prints the batches claimable *right now*: no exclusionary
  tag, absent from `CLAIMS.md`, dependencies satisfied, and no file overlap with anything in
  progress. That is the whole eligibility section of `claim-batch.md`, which an agent otherwise
  answers with 6 to 9 reads across two files.
- **`specflow claim <N>`** — writes the `## In progress` entry (heading, `Owner` from
  `config.agents`, `Started` in UTC) at the top of `CLAIMS.md`.
- **`specflow finish <N> --commit <sha>`** — moves the entry to the top of `## Completed` with
  `Finished` and `Commit`, appends the agent's summary (`--summary-file`, or stdin), deletes the
  batch section from `BUILD_QUEUE.md`, appends the agent's one-paragraph summary
  (`--done-file`) to `BUILD_QUEUE_DONE.md`, and prunes `CLAIMS.md` to its 5 newest completed entries.

**The division of labor: the CLI owns placement, format, and timestamps; the agent owns every word of
prose.** No verb composes a sentence a human will read, and **no verb commits** — committing stays
with the `commit` / `push` levers and the procedures.

The stronger argument is not turns saved but **format determinism**. Entries are hand-written today
by whichever agent holds the batch, in whatever shape it infers from the neighbors, while
`internal/kit/kit.go` parses them back with regexes to power `status`, `verify`, and pruning. A verb
makes the shape machine-guaranteed across Claude, Cursor, Copilot, and anything added later, which is
what makes those parsers trustworthy rather than best-effort.

**This is a positioning change, made deliberately.** The output stays plain markdown plus git,
readable and editable by hand, and the verbs stay **optional**: the procedures keep every manual step,
so an agent (or a person with an editor) can still do all of it, and a repo whose agent never calls
the CLI behaves exactly as before. What changes is that specflow is no longer only a scaffolder of
text. Batch NB (`--new-batch`) already assumed this, since it writes `spec/` and appends to the queue.
Decided 2026-08-20.

### Declared batch fields

`next` cannot answer the overlap check while "Files this batch creates/edits" is a prose convention
that batches follow when they remember to. Each batch section in `BUILD_QUEUE.md` therefore declares
a fixed shape, which the shipped template demonstrates: the heading
(`## Batch <id> [TAG] — <title>`), an optional `**Depends on:** Batch X[, Batch Y]` line, and a
`### Files this batch creates/edits` list. Everything else in the section stays free prose.

Parsing is line-oriented and forgiving, and **fails loudly rather than silently**: a batch missing a
declared field is reported by `next` as unparseable, never quietly treated as claimable. The queue
remains user-owned and hand-editable, so the parser must never be the reason a user's edit is lost.

### Batch size

A batch is sized by **the layers it crosses, not the deliverables it lists.** Three edits inside one
layer is a small batch; one edit each to the spec, a template, the CLI, and its tests is a wide one,
however short each item reads in the queue. Width is what sets the context a batch costs: every layer
crossed pulls in its own files, conventions, and verification, and the agent holds all of them at once
until the batch closes.

What counts as a layer is **per project and deliberately not enumerated here** — in this repo they run
roughly spec, `templates/`, the root managed files, the CLI, and its tests, but that list is an example,
not a contract. The authoring rule is the part that generalizes: when a batch's declared file list spans
more layers than its goal needs, split it along the layer seam. Split batches also declare disjoint file
lists, which is exactly what `next` needs to let them run in parallel.

## Distribution

specflow ships as a **single, statically-compiled Go binary** — **no runtime (Node, Python, …)
required** on the user's machine. This is what makes "install into any repo, any language, no
friction" literally true: the binary scaffolds and upgrades the markdown + git protocol, and the
*output* is already language-neutral. (The CLI was a zero-dep Node script through v0.x; it is being
ported to Go — see `BUILD_QUEUE.md`. Decided 2026-06-21.)

- **Artifact host: GitHub Releases.** A `v*` tag triggers **GoReleaser** in a GitHub Action that
  cross-compiles per OS/arch (macOS arm64/x64, Linux x64/arm64, Windows x64), attaches archives +
  checksums to the release, and can update a Homebrew tap / Scoop manifest in the same run. GitHub
  Releases *is* the registry — no npm/PyPI account needed to host the binary.
- **The tag publishes.** `release.draft: false` — a pushed tag goes straight to a public release,
  no draft to approve. Through v0.1.4 it drafted instead, which meant the release *copy* gated the
  *binaries*: twice a release was hand-written in the UI rather than published from the draft,
  shipping zero assets while `install.sh` resolved `releases/latest` and 404'd for every user. Notes
  are editable after the fact; a missing archive is not. Because the push is therefore irreversibly
  public, the human checkpoint moves to the decision to tag — the agent needs the user's explicit
  approval for every release (`CLAIMS.md` → *Releases need the user's approval*). Decided 2026-08-16.
- **The release body is authored, not generated.** GoReleaser's default body is a filtered commit
  list, which answers "what commits landed" — not the question its actual reader has. The reader is
  overwhelmingly an **agent** deciding whether a repo it maintains needs `specflow upgrade` and what
  that will do to the tree; the humans who read it are few and technical. So each release carries a
  hand-written `.github/release-notes/vX.Y.Z.md`, written in the same commit as the version bump and
  passed to GoReleaser with `--release-notes`. Writing it *in the release commit* is the point: the
  tag push is irreversible and public, so the notes get reviewed in the same diff as the bump rather
  than edited in afterwards — which needs an API token the agent doesn't have, and which nobody
  remembers to do. A missing file falls back to the generated changelog and never fails the job: a
  release that ships no archives is far worse than one with a plain body.

  **Shape, in priority order** — action first, because the reader is deciding what to run:
  1. **What to run**, as a command block, and whether it's required.
  2. **What it changes on disk**: exact paths, and what happens to a file the user edited.
  3. **Behavior changes** to output an agent may be parsing (a new `status` row, a changed verb).
  4. **Fixes**, then install lines and the compare link.

  No marketing register, no emoji, no "we're excited to". Name paths, flags, and commands exactly.

- **Install front-ends.** v1 ships two, both resolving to the same release binary: a `curl … | sh`
  script that detects OS/arch, and **Homebrew** (`brew install`) for macOS/Linux. Deferred post-v1
  (see `open-questions.md` → Distribution): an **npm wrapper** so `npx specflow` still works for the
  JS ecosystem (esbuild pattern), and **Scoop/Winget** for Windows. `go install …@latest` works as a
  bonus for users who have Go.
- **Templates travel inside the binary** (Go `embed`), so there is no separate `files` allowlist to
  curate and the repo self-hosting the protocol at its own root never leaks into the shipped artifact.
- **Caveat:** a future Claude-plugin packaging must target `templates/agents/claude/.claude/`, never
  a self-hosted `.claude/` at the repo root.

## SaaS frontier (later)

The git tier above is the core and stands alone offline. A hosted tier (`spec_for_ai-as-a-service`)
would be an upstream **producer** that authors/syncs `spec/` + `BUILD_QUEUE.md` into the repo; the
git tier executes them and never depends on it. **The file-contract is the API between the tiers** —
keep it stable and versioned and the SaaS bolts on without a rewrite.
