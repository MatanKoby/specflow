# Build Queue — Completed History

One-paragraph summaries of every shipped batch, newest at the top. Skim this for context when
picking a new claim. The full implementation history is in `git log` + `specflow/history/CLAIMS_DONE.md`.

<!-- Append a summary here when you finish a batch (see specflow/procedures/finish-batch.md).
     Format, e.g.:

## Batch 1 — <title>
Shipped <what> in <where>. Key commit `<sha>`. <One line on any follow-up deferred.>
-->

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
