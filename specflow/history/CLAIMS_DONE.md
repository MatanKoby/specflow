# Claims — Archive

Older completed entries archived from `CLAIMS.md`. Reference-only, newest archived batch at
the top. Append-only institutional memory: entries arrive verbatim and are never rewritten.

Written by `specflow/procedures/prune-ledgers.md`, which keeps the 5 newest completed entries in
`CLAIMS.md` and moves everything older here. Don't hand-move entries; run the procedure (Claude:
the `prune-ledgers` skill) so the retention rule stays consistent.

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch QV

### Batch CE — Context economy + `config.check`
- Owner: claude
- Started: 2026-08-20 11:50
- Finished: 2026-08-20 11:56
- Commit: 25f419e

**What shipped.** Three changes aimed at the recurring per-batch context cost, all of them reaching
existing installs through `upgrade`:

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch CE

### Batch RD — Release auto-publish, and the user approves every release
- Owner: claude
- Started: 2026-08-16 13:05
- Finished: 2026-08-16 13:10
- Commit: e279fb9

**What shipped.** `.goreleaser.yaml` `release.draft` flipped `true` → `false`, so a pushed `v*` tag
now publishes a public release with its archives attached and no manual step. Plus the counterpart
rule at the top of this file (*Releases need the user's approval*) and the decision recorded in
`spec/architecture.md` → artifact host.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch RD

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch PR

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch 4

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch SZ

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch SL

### Batch CH — Claude Code batch-boundary hook (opt-in)
- Owner: claude
- Started: 2026-07-12 11:39
- Finished: 2026-07-12 14:45
- Commit: 7b3ffdc

**What shipped.** The Claude-only deterministic backstop for the finish-batch step-6 handoff, on top
of the portable FH text. (1) **Hook** `templates/agents/claude/.claude/hooks/specflow-handoff-reminder.sh`:

**Manual prereq.** The hook is inert until registered in `.claude/settings.json`; the CLI/relay
surface the exact block. Activating it in *this* repo is deferred as a user opt-in (see follow-up).

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch CH

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch RF

### Batch 3 — Broaden the test suite
- Owner: claude
- Started: 2026-07-11 19:52
- Finished: 2026-07-11 20:02
- Commit: 677b265

**What shipped.** Locked behavior beyond the existing file-existence smoke checks, in three areas.
(1) **Content assertions** on the generated full-mode `AGENTS.md` (`TestAgentsMdContentSections`):

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch 3

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch FH

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch 5

### Batch 2 — `specflow status`
- Owner: claude
- Started: 2026-07-05 04:23
- Finished: 2026-07-05 04:39
- Commit: 3f67292

**What shipped.** A read-only **`specflow status`** that orients a user/agent at a glance, writing
nothing. **`kit.Status`** (in `internal/kit/kit.go`) assembles a snapshot from the stamp + repo:

**Verification.** `go test ./...` green (7 new tests: fresh install, active claims incl.
owner/unassigned, drift flag, version mismatch + upgrade hint, spec-only queue-n/a, not-installed
non-zero exit, and `--help`). `go vet` + `gofmt` clean. Manually exercised across all scenarios.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch 2

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch 1

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch G2

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch SO

### Batch BI — Brownfield-aware `init` (inject-with-consent, review handoff)
- Owner: claude
- Started: 2026-06-23 17:08
- Finished: 2026-06-24 06:12
- Commit: 6d14a90

**What shipped.** `init` is now brownfield-aware and non-destructive, in two consent-gated phases:
Phase 1, for each target file that **already exists**, injects specflow's marker-wrapped region at
the top (existing content preserved); Phase 2 explains, then creates, the specflow-owned files.

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch BI

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch CFG

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch G1

### Batch U2 — Self-documenting, edit-resistant region markers
- Owner: claude
- Started: 2026-06-17 15:21
- Finished: 2026-06-17 15:55
- Commit: 66442e0

**What shipped.** The `specflow:start` marker now carries an inline note — `managed by specflow; do
not edit inside these markers (your edits block specflow upgrade). Add your own notes outside them.`

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch U2

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

- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch U
