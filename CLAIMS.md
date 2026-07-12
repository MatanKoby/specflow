# Claims

Execution-state ledger, managed by coding agents. Records who is working on what and the recent
completion log. The user does not normally edit this. Procedures:
`specflow/procedures/claim-batch.md` and `specflow/procedures/finish-batch.md`.

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

## Completed

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

<!-- Recent finishes, newest first. Older entries archived to specflow/history/CLAIMS_DONE.md. -->

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
