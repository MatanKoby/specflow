# Build Queue — Completed History

One-paragraph summaries of every shipped batch, newest at the top. Skim this for context when
picking a new claim. The full implementation history is in `git log` + `specflow/history/CLAIMS_DONE.md`.

<!-- Append a summary here when you finish a batch (see specflow/procedures/finish-batch.md).
     Format, e.g.:

## Batch 1 — <title>
Shipped <what> in <where>. Key commit `<sha>`. <One line on any follow-up deferred.>
-->

## Batch LW — Ledger weight: bound the entry, not just the count
Pruning bounded the ledgers by **count** — `CLAIMS.md` to its 5 newest completed entries, `BUILD_QUEUE.md` to zero completed batches — and both rules were working. Neither bounded the size of a *single* entry, nor the prose that is not an entry at all. A downstream install running at the prescribed retention of 5 still carried a 27 KB `CLAIMS.md`; this repo reproduced both halves (20 KB `CLAIMS.md`, and 59 of `BUILD_QUEUE.md`'s 149 lines sitting above the first batch heading). Three changes. **(1) The narrative is written once.** `finish-batch` asked for prose about one batch in two files, authored independently, with neither a superset of the other. `CLAIMS.md` now takes a **stub** — metadata, at most 8 lines of "What shipped", and a pointer — and `specflow/history/BUILD_QUEUE_DONE.md` takes the **full narrative**, since nothing reads the archive on the hot path while `CLAIMS.md` is re-read on every claim, finish, and prune. `specflow finish` grows `--stub-file` (`--summary-file` kept as the old name, so an agent on an older procedure copy still files its prose) and refuses an over-length stub *before writing anything*, so the fix is to move prose into the done-file and retry against an untouched repo. Hard reject rather than stop-and-ask: unlike splitting a spec file, moving a paragraph into the archive loses nothing, so there is no judgment to put to the user. Blank lines and the pointer don't count against the cap. **(2) The queue preamble is capped at 45 lines** (the shipped template is 33), reusing the `specflow:size-ok` stop-and-ask from Batch SZ and re-asking every +15, with a new `prune-ledgers` section 3 that sorts preamble paragraphs into delete / relocate-via-`spec-edit` / keep. That section *does* ask, because deciding which spec file owns a stranded paragraph is a judgment call about concerns. The generalisable finding, now recorded in `spec/architecture.md` → *Ledger lifecycle*: the preamble is where an agent parks a durable fact when it cannot decide which `spec/` file owns it — at finish time the queue is already open, writing there is one edit, and no retention rule ever comes back for it, so it fills. **(3) `specflow next` and `specflow verify` report both ledgers' line counts** and warn past a bound (the retention count, and the preamble cap, honoring the waiver). This is reporting, not a second retention rule: the cut stays a count for its determinism, and what a count cannot reveal is a file that grew heavy while its count stayed correct. It absorbs the optional companion Batch NX was carrying, which NX's section no longer claims. Dogfooded in the same batch: this repo's preamble went 59 → 28 lines, with the v0.1.x release history relocated to `spec/roadmap.md` → *Release lines*, and the stale "v0.1.7 is open, not tagged" line deleted. Key commit `5dcced8`. Verified by `gofmt`/`go vet`/`go test` plus four new tests (over-cap refusal writes nothing anywhere, the cap counts prose only, the legacy flag still lands, and weight reporting warns then honors the waiver), and a self-hosted `upgrade` refreshed the managed procedure and skill copies.

## Batch CD — Batch-width and prune discipline in the procedures
Two guidance changes distilled from a session that exhausted a 373 K-token context window. Batches are now sized by **the layers they cross**, not the deliverables they list (`AGENTS.md` → *The work queue*, plus `spec-edit.md` where batch sections get written); a batch spanning more layers than its goal needs is split on the layer seam, which also gives the pieces the disjoint file lists that parallel claiming requires. What counts as a layer stays per project on purpose. And `claim-batch.md` now tests the ledger-retention rule **before** claiming, not only at finish — more than 5 completed `CLAIMS.md` entries means prune first — because pruning only on the way out leaves the next agent reading the overgrown file on the way in, which is where the cost actually lands. A third item from the same report (a write-side rule preferring the file-editing tool over shell edits, overriding harness modes) was discussed and **not adopted**; it is parked with its reasoning in `spec/open-questions.md`. Design: `spec/architecture.md` → *Batch size* and *Ledger lifecycle*.

## Batch RN — Authored release notes
**Batch RN — Authored release notes.** The GitHub Release body is now written, not generated:
`.github/release-notes/<tag>.md` is authored in the same commit as the version bump and passed to
GoReleaser via `--release-notes`, so a pushed tag publishes it directly. Writing it in the release
commit is the mechanism rather than a convention — the tag push is irreversible, so the notes are
reviewed in the same diff as the bump, and editing a published body afterwards needs a GitHub API
token that `git push` over SSH doesn't provide. A missing file falls back to the generated changelog
with a job-summary warning and never fails the job. The house style (action first, then what changes
on disk, then behavior changes an agent may be parsing) is specced in `architecture.md` →
Distribution and keyed to the real reader: an agent deciding whether a repo it maintains needs
`specflow upgrade`. `v0.1.6`'s notes are backfilled as the worked example; its published release
keeps the generated body. Repo-internal — no version line. Commit `26b2cbc`.

## Batch AF — Adapter files upgrade like everything else
**Batch AF — Adapter files upgrade like everything else.** Closed the create-once hole that froze
every install's skill stubs and handoff hook at whatever shipped on install day: the marker-less
adapter files are now managed as *whole* files on the same drift contract as a region (clean is
replaced, edited is left alone with a `.specflow-new` sidecar), with a one-time adoption path that
carries installs predating the tier across — identical to the template adopts silently, anything
else is backed up to `.specflow-bak` first. `verify` covers the adapters instead of passing a
deleted or mangled one clean, `status` gained a `stale` row that separates "you edited it" from
"specflow moved and this file didn't", and the four skill stubs finally name the queue verbs
v0.1.5 shipped into the procedures. Also fixed `normalizePath`, which stripped the leading dot from
`.claude/`-style paths and hid them from the queue's overlap check. Commit `f5f578f`.

## Batch QV — Queue verbs (`next`, `claim`, `finish`)
Shipped the three queue verbs. `specflow next [--json]` answers the whole eligibility section of
`claim-batch.md` in one read-only call (tag, already-claimed, dependency, file overlap), `specflow
claim <id>` writes the In-progress entry, and `specflow finish <id> --commit <sha>` moves the entry
to `## Completed`, deletes the batch from the queue, files both archive paragraphs, and prunes to
the 5 newest. Batch sections now have a declared shape the parser reads, and a batch missing a field
is reported unparseable rather than offered. The CLI owns placement, format, and timestamps; the
agent still writes every word of prose (`--summary-file`, `--done-file`), and no verb commits. The
procedures name the verbs as the fast path while keeping every manual step. Key commits `80df6c4`, `2346480`, `a7df418`.

## Batch CE — Context economy + `config.check`
Cut the protocol's recurring per-batch context cost. The four procedures now name the cheap read
(grep the headings, slice the one section) wherever they say what state to check; `AGENTS.md` gained
a *Working economically* section (batch independent reads, never re-read your own write, one check
command), with its queue-specific bullets `full-only` so spec-only installs stay clean; and
`config.check` records the repo's single check command, asked at `init` (skippable, or `--check=`),
shown by `status`, quoted back by `finish-batch`, and never executed by specflow. Named `check`
rather than `verify` to avoid colliding with the `specflow verify` command. Key commit `25f419e`.
Measured basis: 41x on `CLAIMS.md`, 33x on `BUILD_QUEUE.md`, both read 3 to 5 times per batch.
Follow-up: `upgrade` does not backfill `check` into older installs (absent reads as "not set").

## Batch RD — Release auto-publish, and the user approves every release
Flipped `.goreleaser.yaml` `release.draft` to `false`, so a pushed `v*` tag now publishes a public
release with its archives attached instead of a draft awaiting a manual click. The draft gate had put
the release *copy* in front of the *binaries*: v0.1.3 and v0.1.4 were each created by hand in the
GitHub UI rather than published from GoReleaser's draft, shipping releases with **zero assets** while
the real drafts sat unpublished, and since `install.sh` resolves `releases/latest` the public
`curl … | sh` install 404'd both times until repaired by hand (v0.1.1 has no release at all for the
same reason). The trade is that release notes become GoReleaser's generated commit list, edited after
publish rather than written before. Because a tag push is now instantly public and irreversible, the
human checkpoint moved earlier rather than vanishing: the counterpart rule at the top of `CLAIMS.md`
requires the user's explicit approval for every release, covering the version bumps, the tag, and the
push. Decision recorded in `spec/architecture.md` → artifact host. Verified with `goreleaser check`
against `@latest` v2, matching the workflow's `~> v2`. Key commit `e279fb9`. Follow-ups deferred:
changelog prettification (`release.header`, `changelog.groups`), backfilling a v0.1.1 release, and
the end-to-end proof that the next tag lands published with 6 assets.

## Batch PR — Ledger pruning (`prune-ledgers`, the fourth procedure)
Closed the gap that let `CLAIMS.md` grow without bound: `finish-batch` appended every completed
entry and no procedure ever wrote to `specflow/history/CLAIMS_DONE.md`, so the archive shipped with
its 206-byte header untouched in every install. New `specflow/procedures/prune-ledgers.md` keeps the
**5** newest `## Completed` entries and moves older ones there verbatim, never touching
`## In progress`; retention is a count rather than a byte budget so pruning cuts on an entry
boundary and two agents reach the same result (measured on a 26-entry host install: 35 to 126 lines
per entry, median 61). Includes a catch-up pass for installs predating it and a narrow
`BUILD_QUEUE.md` sweep for completed/dissolved/absorbed sections. `finish-batch` delegates as step
**4a** (numbered, not renumbered, so the "step-6 handoff" references survive); the rules live in the
procedure and the Claude skill stays a thin trigger so pruning does not become Claude-only. Go side:
`specOnlyOmits` + `kit.QueueTokens`. Key commit `dd7a1e9`. Follow-up deferred: nothing enforces
pruning until Batch E, where `verify --batch` is the natural home for a retention check.

## Batch 4 — README badges + file-map
Rewrote `README.md` ahead of a public launch post, directed by the user. Three badges (CI,
auto-tracking release, MIT); a centered hero with an ASCII `specflow init` console demo standing in
for the still-deferred animated GIF; five **Why** blocks led by continuity ("your agents forget; the
repo doesn't") with token savings third and its spec-coverage condition stated inline; a mermaid
flowchart of spec → queue → claim → build showing two agents claiming in parallel; and the batch's
file-map deliverable as an annotated tree with `[owner]` tags plus the ownership table. User-added
scope: the README is now **agent-executable** — an HTML comment at the top of the raw file routes an
agent to a `## For AI agents` section of seven ordered steps (ask the user which agents and which
mode rather than guessing, `init` never commits, relay the Claude-Code handoff hook block, read
`AGENTS.md` before any work). Drafted claims were checked against a live `init` in a temp repo, which
corrected the file count (16, not 11) and the `spec/` contents (only `README.md` at install). Key
commit `041f1bc`. Follow-ups: the *How it differs* competitor paragraph is unverified against those
projects' current behavior, and a 1280x640 social-preview image was recommended but not produced.

## Batch SZ — spec-file 600-line hard cap
Turned the size rule in `spec-edit.md` from a nudge that had never fired (largest spec file here:
177 lines) into an enforced stop, and dropped its "~20k tokens" gloss, which overstated the real
figure by about 2x. Before an edit crosses a file's current limit the agent stops and asks the user,
presenting the section headlines, a single-concern claim, and the read-cost warning verbatim; it
never asks for a number. A *keep* is recorded as a `specflow:size-ok` first line with a UTC
timestamp and the next threshold, advancing +200 each time. `archive.md` and `research/` are exempt.
Mirrored into the dogfood procedure and the `spec-edit` SKILL.md. Key commit `f9fa57b`, dogfood
`02e6a41`. Follow-up: `upgrade` still won't refresh a non-managed adapter file whose content merely
went stale in full mode (SL's repair only fires on a spec-only leak), so the stub was hand-mirrored.

## Batch SL — spec-only mode leaks queue/batch language
Fixed a user-reported 0.1.2 defect where `init --spec-only` generated files pointing agents at
`BUILD_QUEUE.md`, `CLAIMS.md`, `claim-batch`, and `finish-batch` — none of which the mode installs.
Gated the six templates that had no mode markers (four adapter rule-files, the `spec-edit` skill
stub incl. its YAML `description:`, and `templates/base/spec/README.md`) with *replacement*
spec-only wording rather than deletion, and widened `CLAUDE.md`'s guards past the step-6 paragraph
to the protocol description and trigger bullets. Added the mode-consistency check baseline hashes
structurally cannot provide (they're taken over the rendered region, so full-mode prose in a
spec-only install matches its own hash — which is why `verify` said "All good" on a broken install):
`kit.ModeLeaks`/`kit.QueueTokens`, shared with the tests, plus the install mode in `verify`'s
output. Made `upgrade` a complete fix path via `staleAdapterFiles`, since non-managed create-once
skill stubs carry no baseline and were being left stale forever. Tests written first and observed
failing across all six files; verified against a real v0.1.2 binary built from `2951d78`. Key commit
`ee4b6b7`, final `3b265e0`. Residue by contract: `spec/README.md` is user-owned so `upgrade` can't
correct existing installs. Follow-up: Batch SZ (600-line cap) unblocked.

## Batch CH — Claude Code batch-boundary hook (opt-in)
Shipped the Claude-only deterministic backstop for the finish-batch step-6 handoff, layered on the
portable FH text. A `PostToolUse(Bash)` hook (`templates/agents/claude/.claude/hooks/specflow-handoff-reminder.sh`)
gates on `git commit` in the command, confirms via the landed HEAD subject `^meta: complete batch-`
(handles `-m`/`-F`/heredoc, proves the commit landed, self-de-dups), and emits
`{"decision":"block","reason":…}` to halt the loop and force the handoff. Soft-deps on `jq`, fails
open so it can never break a commit. The CLI prints an opt-in `.claude/settings.json` paste-block at
the end of `init`/`add-agent` (recommends committed settings; no auto-merge — JSON can't be
marker-merged); the claude `CLAUDE.md` full-only region tells the agent to relay that step. spec-only
omits it (no batch boundary). Also **extended `upgrade`** to place newly-shipped non-managed adapter
files create-once (scoped to `agents/`, never resurrects a deleted base file), so existing installs
receive the hook — previously upgrade only refreshed managed regions. Key commits `c99b734`
(hook + CLI + relay + tests), `2e4b69a` (upgrade convergence), `7b3ffdc` (dogfood upgrade
0.1.0→0.1.1). Design recorded in `spec/open-questions.md` → Quality / enforcement. 4 new tests
(install+notice, add-agent, jq-gated script behavior, upgrade-adds-hook). Follow-up: activating the
hook in this repo's own `.claude/settings.json` is left as a user opt-in.

## Batch RF — Ship the research-flow convention
Made the lightweight research-note flow a shipped part of specflow, so a fresh install carries the
convention (previously only self-hosted). Three template edits, all self-contained (no queue/claim
refs) so **spec-only inherits them**: `templates/base/AGENTS.md` names the optional pre-design
research step + the gate-free `spec/research/` home in its spec-discipline region;
`templates/base/specflow/procedures/spec-edit.md` gains a *Research notes* section (gate-free /
write-as-you-go / graduate-upward, framed as the pre-design exception to the archive rule);
`templates/base/spec/README.md` documents the optional `research/` sub-folder. Key commit `9fbba41`.
2 new tests (full + spec-only) assert `init` ships the convention in all three files; the existing
spec-only banned-word test guards the sections stay queue-free. Propagated the two managed files to
the dogfood copies via `specflow upgrade` (verify clean). No follow-ups.

## Batch 3 — Broaden the test suite
Locked behavior beyond the file-existence smoke checks with 4 new tests. **Content assertions** on
the generated `AGENTS.md` (`TestAgentsMdContentSections`): its key sections, the commit-grammar table
header + `batch-N`/`meta:`/`spec:` rows, and the `specflow/procedures/*.md` path references.
**Picker + adapter coverage**: the interactive agent picker over piped stdin (numeric multi-select +
the "a" all-shortcut) and the `--all` flag, asserting each of the five adapters' instruction files is
written, marker-wrapped, and baseline-hashed, then a follow-up `upgrade` stays a clean no-op — first
coverage of the copilot/bob/antigravity adapters. An **embed-manifest test** in the new root-package
`templates_test.go` walks the on-disk `templates/**` tree and asserts every file is embedded
byte-for-byte, counting dot-paths so the `//go:embed all:` dotfile footgun can't regress (proven to
bite: dropping `all:` fails it, then reverted). Key commit `677b265`. `go vet`/`gofmt` clean. The
suggested `internal/kit/` unit split was skipped as redundant with the end-to-end binary coverage.

## Batch FH — finish-batch step-6 handoff rework
Reworked step 6 of `finish-batch.md` so the end-of-batch context handoff is hard to skip: it now
states the payoff to the user (cheaper + more reliable next batch, a decision point), names and
refutes the "it's noise" rationalization, and requires a fixed terminal handoff line so an omission
is visible. Step 7 clarifies that "continue" authorizes the next claim but doesn't waive the line.
Canonical template edited and propagated to the dogfood copy via `specflow upgrade` (stamp
rebaselined). Key commit `2962ae1`. Portable (text-only, all agents); the Claude-Code deterministic
hook backstop is queued as Batch CH.

## Batch 5 — `--dry-run` (preview)
Shipped a `--dry-run` flag on `init` and `upgrade` that prints the planned file operations and exits
without touching disk. `init --dry-run` reuses `PlanInit` (would create / inject / already-wired /
skip), always non-interactive, previewing the default agent when `--agents` is omitted;
`upgrade --dry-run` previews refresh / add / migrate / drift via a new read-only `PlanUpgrade`. The
per-file upgrade classification was factored into a shared `decideUpgrade`/`upgradeDecisions` pair so
the apply path (`Upgrade`, unchanged in behavior) and the planner can't diverge. Key commit `7f5f70b`.
6 new tests + the existing upgrade suite guard the refactor; `go vet`/`gofmt` clean. **Completes
Milestone v0.1** (code-complete; next is tagging `v0.1.0`).

## Batch 2 — `specflow status`
Shipped `specflow status`: a read-only orientation summary that writes nothing. `kit.Status`
reports the kit version (stamp vs. binary, with an upgrade hint on mismatch), install mode, wired
agents, commit/push levers, active claims parsed from CLAIMS.md's In-progress section (owner shown;
`none` -> unassigned), the un-done batch count from BUILD_QUEUE.md, and a drift flag for any managed
region edited since install. Spec-only installs report the queue as n/a; not-installed exits
non-zero. Key commit `3f67292`. 7 new tests; `go vet`/`gofmt` clean.

## Batch 1 — `specflow add-agent <name>`
Shipped `specflow add-agent <name> [<name>...]`: wires another agent's adapter into an
already-initialized repo. `kit.AddAgent` copies the agent's adapter files (skip-existing,
non-destructive), injects specflow's region into an existing instruction file (content preserved) or
leaves an already-wired one alone, records the agent in `config.agents`, and refreshes the
managed-region baselines; mode-aware, so a spec-only repo doesn't gain the claim/finish skills. The
CLI validates agent names up front, guards the not-installed case, and never commits (review-then-
commit handoff, like `init`). Key commit `fa3aa3d`. 8 new tests; `go vet`/`gofmt` clean. Follow-up
deferred: `remove-agent` (decision pending).

## Batch G2 — Go release + install pipeline
Shipped zero-runtime distribution. **GoReleaser** (`.goreleaser.yaml`) cross-compiles a 5-target
matrix (linux/darwin amd64+arm64, windows amd64) into checksummed `tar.gz`/`zip` archives named
`specflow_<version>_<os>_<arch>`; a **`Release` workflow** on any `v*` tag runs `goreleaser release
--clean` → a **draft** GitHub Release; an **`install.sh`** `curl … | sh` front-end detects OS/arch,
checksum-verifies, and installs the matching binary (`/usr/local/bin` or `~/.local/bin`, `NO_COLOR`
aware). README rewritten to the binary flow (+ `go install …@latest` fallback). Verified via a
`--snapshot` build plus a throwaway `v0.0.1-test` tag that ran the workflow green and produced a draft
with all assets (tag + draft since deleted). Key commit `50c59aa`. Deferred post-v0.1: Homebrew tap,
npm wrapper, Scoop/Winget. Follow-up: the full public `curl|sh` path proves out at the real `v0.1.0`.

## Batch SO — Spec-only install mode (`--spec-only`)
Shipped `specflow init --spec-only`: a lighter install that keeps only the spec discipline
(`AGENTS.md` spec sections + `spec/` + the `spec-edit` procedure/skill + stamp + agent stubs) and
omits `BUILD_QUEUE.md`, `CLAIMS.md`, the `claim-batch`/`finish-batch` procedures + skills, and the
`specflow/history/` archives. Implemented as **section composition** (one source, no forks):
`AGENTS.md` and `spec-edit.md` carry `specflow:full-only`/`spec-only` marker pairs and `renderBody`
keeps/drops them per mode (own-line vs inline markers handled separately so tables/spacing stay
intact); the baseline hash is taken over the *rendered* region. Mode is recorded in `config.mode`
and threaded through `init`/`upgrade`/`verify`. Key commit `93519b0`. Follow-ups deferred: the
spec-only→full graduation path, and section-composing the per-agent stubs (a spec-only Claude install
still ships a `CLAUDE.md` pointer that mentions queue/claim — cosmetic).

## Batch BI — Brownfield-aware `init` (inject-with-consent) + `specflow verify`
Rebuilt `init` into a non-destructive, two-phase, consent-gated flow: Phase 1 injects specflow's
marker-wrapped region into target files that already exist (content preserved), Phase 2 creates the
owned files; `init` tracks its own created/modified/declined list, ends with a "review `git diff`,
then commit" handoff, and **never commits** (non-interactive proceeds and points at `git status`).
Brought all five per-agent instruction files under management (marker-wrapped, refreshed by `upgrade`
for installed agents only) with **idempotent injection** (refresh a region; skip "already wired" when
a file already references `AGENTS.md`) and **tier-aware** decline/missing notices. Added
per-subcommand `--help` and a basic **`specflow verify`** install-integrity check (`verify --batch`
stubbed until Batch E). Relocated the `_DONE` archives to `specflow/history/` and updated every path
reference. Key commits `b766da2` (two-phase init), `b57bd50` (idempotent injection + tier-aware
decline), `f583f39` (`verify`), `6d14a90` (`_DONE` relocation). Follow-up: Batch SO (spec-only)
shares the template/section + marker work.

## Batch CFG — Config file, commit/push levers & safety fixes (v0.1 foundation)
Renamed the stamp to `specflow/config.json` (out of the repo root) with a **`config` block**
(`agents`/`mode`/`commit`/`push`) beside internal state (versions, schema, `managed` hashes). Taught
the procedures the two commit/push levers: `AGENTS.md` defines `config.commit` / `config.push`
(`agent|user`) authoritatively, and `claim-batch.md` / `finish-batch.md` carry a lever-note (when
`commit: user`, alert + supply a suggested message instead of committing; when `push: user`, commit
but don't push); default `agent`/`agent`. Safety: a managed file with **no baseline** is treated as
drift (`.specflow-new` sidecar, never overwrite — risk A); a corrupt `config.json` fails friendly;
`init` **requires git** (refuses otherwise). CLI: no ANSI when stdout isn't a TTY or `NO_COLOR` set.
Key commits `48ac469` (safety/CLI), `ea3bffb` (config.json + block), `5204656` (lever wording).
Follow-ups: the init-time lever prompt + brownfield two-phase flow → Batch BI; `_DONE` archives move
to `specflow/history/` in BI.

## Batch G1 — Port the CLI to Go (full replace)
Replaced the Node `bin/specflow.js` with a single statically-compiled **Go binary** (`cmd/specflow`
+ `internal/kit`), templates embedded via `//go:embed all:templates` (the `all:` prefix keeps the
dotfiles). Ported the smoke suite to `go test` (builds + drives the real binary against temp repos).
Verified: `init` output **byte-identical** to the Node CLI, stamp semantically identical (managed
hashes match), `upgrade` a clean idempotent no-op, and self-hosted by running the Go `upgrade` on
this repo. Node files removed (`bin/specflow.js`, `package.json`, `test/smoke.js`); CI switched to
`go vet`/`build`/`test` on ubuntu + macos; README + architecture file-map updated; pending batches
retargeted to Go. Key commit `422284b` (Go CLI), final `ac82cec`. Follow-up: Batch G2 ships it
(GoReleaser → GitHub Releases + curl|sh + Homebrew).

## Batch U2 — Self-documenting, edit-resistant region markers
Baked a "managed by specflow; do not edit inside" note into the `specflow:start` marker (an HTML
comment — invisible in rendered markdown, visible in the raw file). Made marker matching token-based
(regex on `specflow:start`/`specflow:end`) so the note can evolve without breaking parsing or forcing
a migration; a clean `upgrade` canonicalizes a file's markers to the template's wording. Key commit
`66442e0`. Smoke suite at 19 checks; applied to the repo's own files (self-hosting). No follow-ups.

## Batch U — Non-destructive upgrade redesign
Made `upgrade` non-destructive: managed files (`AGENTS.md` + procedures) wrap their generated content
in `<!-- specflow:start/end -->` markers, `init` records a SHA-256 of each region in the stamp's
`managed` map, and `upgrade` refreshes only the region — preserving everything outside, leaving a
hand-edited (drifted) region untouched with a `.specflow-new` sidecar, and migrating pre-marker
installs via a `.specflow-bak` backup. Key commit `42cd047`. Smoke suite at 18 checks; applied to
specflow's own root files (self-hosting). No follow-ups specific to U.
