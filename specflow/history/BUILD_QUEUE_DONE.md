# Build Queue — Completed History

One-paragraph summaries of every shipped batch, newest at the top. Skim this for context when
picking a new claim. The full implementation history is in `git log` + `specflow/history/CLAIMS_DONE.md`.

<!-- Append a summary here when you finish a batch (see specflow/procedures/finish-batch.md).
     Format, e.g.:

## Batch 1 — <title>
Shipped <what> in <where>. Key commit `<sha>`. <One line on any follow-up deferred.>
-->

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
