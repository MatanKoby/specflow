# Build Queue — Completed History

One-paragraph summaries of every shipped batch, newest at the top. Skim this for context when
picking a new claim. The full implementation history is in `git log` + `specflow/history/CLAIMS_DONE.md`.

<!-- Append a summary here when you finish a batch (see specflow/procedures/finish-batch.md).
     Format, e.g.:

## Batch 1 — <title>
Shipped <what> in <where>. Key commit `<sha>`. <One line on any follow-up deferred.>
-->

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
