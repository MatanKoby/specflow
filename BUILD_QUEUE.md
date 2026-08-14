# Build Queue

Reference spec: [`spec/`](spec/README.md)
Agent work tracking: `CLAIMS.md` (managed by coding agents)
Completed history: [`specflow/history/BUILD_QUEUE_DONE.md`](specflow/history/BUILD_QUEUE_DONE.md) — one-paragraph summaries of shipped batches.

## How this works

- This file lists only **un-done batches**, in full. Completed batches collapse to summaries in
  `specflow/history/BUILD_QUEUE_DONE.md` (git log + `specflow/history/CLAIMS_DONE.md` hold the implementation history).
- Dependencies are listed where they exist — the agent decides execution order.
- Agents claim and track completion in `CLAIMS.md`. **No Owner / Started / Status ever goes here.**
- See `specflow/procedures/claim-batch.md` before claiming.

---

## Un-done batches

> **Pick-order pointer — Milestone v0.1 is shipped** 🚀 (goal/DoD in `roadmap.md`). `v0.1.0` is
> tagged, published, and the public `curl|sh` install is verified end-to-end. All member
> batches are done: **CFG** (config + commit/push levers + safety), **BI** (brownfield `init` +
> `verify` + `_DONE` relocation), **SO** (spec-only mode), **G2** (release: GoReleaser → GitHub
> Releases + `curl|sh`; Homebrew deferred), **1** (add-agent), **2** (status), **5** (`--dry-run`).
> **`v0.1.1`** (patch) ships the user-facing content improvements since: **FH** (finish-batch step-6
> handoff rework) and **RF** (research-flow convention).
> **`v0.1.2`** (patch) ships **CH** (Claude-Code step-6 handoff hook: a `PostToolUse` backstop that
> blocks the loop after a `meta: complete batch-*` commit to force the handoff) plus the `upgrade`
> convergence that delivers newly-shipped non-managed adapter files to existing installs.
> **`v0.1.3`** (patch) ships **SL** (spec-only mode no longer names the queue/claim
> machinery it omits, plus the `verify` mode-consistency check and an `upgrade` repair path for
> existing installs) and **SZ** (the spec-file 600-line cap is now a stop-and-ask, with a
> `specflow:size-ok` waiver that re-asks every +200 lines).
> **`v0.1.4`** (patch) ships **4** (README rewrite: badges, file-map, and an agent-executable
> install section).
> **Pick next: Batch PR** (ledger pruning — the `prune-ledgers` procedure + skill, and the Go/doc
> wiring for a fourth procedure). The only un-blocked entry; everything else is `[NOT READY]`.
> Unblock **Batch W** (needs the profile→dimension mapping in `open-questions.md`) or **Batch NB**
> (needs a design pass) before claiming either.
> **Post-v0.1 queue below:**
> **Batch PR** (ledger pruning) · **Batch W** (workflow config) · **Batch NB** (`--new-batch`) ·
> **Batch E** (enforcement — research-first) · **Batch P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch PR — Ledger pruning (`prune-ledgers`, the fourth procedure)

**Goal.** `CLAIMS.md` grows without bound: `finish-batch` appends every completed entry to
`## Completed` and nothing ever moves to `specflow/history/CLAIMS_DONE.md`. The archive file ships,
`AGENTS.md` documents it, and no procedure writes to it — the only "archive when it grows long"
sentence lives *inside* `CLAIMS_DONE.md`, which agents never open, and carries no threshold. Two
independent long-running installs confirm it: specflow's own ledger is 18 entries / 36 KB and a host
install reached 26 entries / 1,789 lines / 125 KB, both with a 206-byte untouched archive header.

Ship a fourth procedure that prunes both ledgers, delegated to from `finish-batch` and runnable by
hand. Design + rationale: `spec/architecture.md` → *Ledger lifecycle*.

**Rules.** `CLAIMS.md` keeps the **5** most recent `## Completed` entries; older ones move to
`CLAIMS_DONE.md`, newest at top, unedited. Retention is a count, not a byte budget (entries measured
35–126 lines, median 61). No stop-and-ask — archiving is lossless and `claim-batch` already resolves
dependencies against either location. Must handle a **catch-up pass** (an overgrown ledger archives
many entries in one run). Queue side: `finish-batch` keeps its own step-4 deletion; pruning only
sweeps sections for batches already in `## Completed` or marked dissolved/absorbed.

**Cross-agent invariant.** The rules live in the **procedure** (`architecture.md` → *Cross-agent
model*); the Claude skill stays a thin trigger. Pruning must not become Claude-only.

### Files this batch creates/edits
- `specflow/procedures/prune-ledgers.md` + `templates/base/` twin (new) · `finish-batch.md`
  delegation (both copies) · `.claude/skills/prune-ledgers/SKILL.md` +
  `templates/agents/claude/` twin (new).
- Go: `internal/kit/` (file list + spec-only carve-out), stamp `managed` map so `upgrade` refreshes
  the new procedure, `verify` Tier 1 presence check, `cmd/specflow/main_test.go` path enumerations
  (lines ~137, ~612).
- Docs: `AGENTS.md` + `templates/base/` twin (file table, procedure list) · `CLAIMS.md` header note
  + template · "three procedures" → four in `README.md` (3 spots), `CLAUDE.md` + template, and the
  bob / copilot / antigravity agent stubs · `spec/README.md` index line.

### Verification
- `go test ./...` green; init into a temp repo lists four procedures; `verify` passes.
- Upgrade an install whose procedures are clean → the new procedure appears.
- Dogfood: run it on this repo's own `CLAIMS.md` (18 entries → 5 kept, 13 archived).

---

## Batch W `[NOT READY]` — Workflow config model

**Depends on:** the profile→dimension mapping in `open-questions.md` → Workflow (the only remaining
build-time detail; the rest of the workflow design — 5 dimensions, no-default explicit-choice flow,
guidelines-only enforcement — is settled).

**Goal.** Implement `spec/workflow.md`: the five dimensions, the explicit-choice setup flow (no
default profile; `--profile` required non-interactively), the `workflow` stamp block, and
`config.md` rendering; procedures reference `config.md`.

### Files this batch creates/edits
- `cmd/specflow/` + `internal/kit/` (setup flow + render) · `templates/base/specflow/procedures/*`
  (policy-dependent steps reference `config.md`) · `templates/base/specflow/config.md` (template) ·
  `cmd/specflow/main_test.go`.

### Verification
- Init each profile into a temp repo; confirm stamp + `config.md` + procedures match.

---

## Batch NB `[NOT READY]` — `--new-batch` quick flow

**Goal.** A "now-to-now" command for when the user wants something specced and queued immediately:
it **initiates a short planning phase**, writes the result into `spec/`, then appends it to
`BUILD_QUEUE.md` as a batch (optionally handing it to the agent to claim + execute).

**Why `[NOT READY]`:** the flow needs a small design pass first — what the planning phase asks, how
much it writes to `spec/` vs the batch, and the hand-off to execution. Design, then build.

**Related (open):** the clarify-and-approve discipline in `spec/open-questions.md` → *Speccing &
approval discipline*. NB's planning phase is the natural home for the explicit clarify+approve gate
(ask the questions, end on a user OK before anything is written or claimed), but the gate principle
is broader than NB — it also bears on `claim-batch` for batches not created through NB.

### Files this batch creates/edits
- `cmd/specflow/` + `internal/kit/` · a `spec/` write + `BUILD_QUEUE.md` append · `cmd/specflow/main_test.go`.

---

## Batch E `[NOT READY]` — Enforcement (research-first)

**For now, enforcement is exactly as in Upside: honor-system / written guidelines** — the procedures
+ `AGENTS.md` tell the agent what to do; nothing executable checks it. This batch does **not** jump
to implementation. It **starts with research + discussion** of how to add enforcement incrementally,
then drafts the sub-batches.

### Phase 1 — research & discuss (the deliverable)
- Survey the layers, cheapest → most authoritative: a read-only **batching/enforcement** validator
  (distinct from the installation `specflow verify` shipped in Batch BI — changed files map to an
  owned claim, no state leaked into `BUILD_QUEUE.md`, commit grammar), local git hooks
  (`commit-msg` / `pre-push` via `core.hooksPath`), CI running the validator, and GitHub branch
  protection.
- For each: what it binds, prevents-vs-detects, bypassability, and the carve-outs needed
  (`meta:` / `spec:` / doc-only) so it never fights legitimate work — **including the install
  bootstrap**: a fresh `specflow init` creates many specflow-owned files with no claim, so the check
  must exempt the specflow-owned scaffold (and `init` commits the install as its own `meta:` commit);
  otherwise running `init` then the check before committing false-positives.
- Output a short `spec/enforcement.md` design doc + a proposed **incremental** sub-batch sequence.
  Generalizes Upside's `docs/process/agent-discipline.md`.

### Then — sub-batches (only after Phase 1 is agreed with the user)
- the batching validator → opt-in `install-hooks` → optional host-repo CI template → branch-protection guidance.

### Files this batch creates/edits (Phase 1)
- `spec/enforcement.md` · queue updates for the agreed sub-batches.

---

## Batch P `[NOT READY]` — optional npm-wrapper front-end

**Superseded 2026-06-21.** Primary distribution is now a Go binary on **GitHub Releases** via Batch
G2 (GoReleaser) — not npm. This batch is reduced to the **optional npm wrapper**: an npm package that
fetches the prebuilt binary so `npx specflow` still works for the JS ecosystem (esbuild pattern).
Claim only if we decide to also serve `npx`. See `open-questions.md` → Distribution.
