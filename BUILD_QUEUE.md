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
> **`v0.1.3`** (patch, unreleased) ships **SL** (spec-only mode no longer names the queue/claim
> machinery it omits, plus the `verify` mode-consistency check and an `upgrade` repair path for
> existing installs).
> **Pick next: Batch SZ** (spec-file 600-line hard cap — unblocked by SL).
> **Post-v0.1 queue below:** **Batch SZ** (size cap) ·
> **Batch 4** (badges + file-map) · **Batch
> W** (workflow config) · **Batch NB** (`--new-batch`) · **Batch E** (enforcement — research-first) ·
> **Batch P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch SZ — spec-file 600-line hard cap

**Goal.** Turn the size watch from a never-firing nudge into an enforced stop. Design ref:
`spec/architecture.md` → *Spec organization*; the gate shape is cross-referenced from
`spec/workflow.md` → *Work-admission gates*.

**Why.** `spec-edit.md` currently says "consider whether the next bite of content wants its own
file" past ~600 lines (~20k tokens). It has never fired: this repo's largest spec file is 177 lines.
The token gloss is also wrong — at the corpus's 68 chars/line, 600 lines is roughly 10–11k tokens,
not 20k.

### Deliverables
- Rewrite the **Size watch** section of `templates/base/specflow/procedures/spec-edit.md`: hard cap,
  stop-and-ask (headlines + single-concern claim + the verbatim read-cost warning), never ask the
  user for a number, `specflow:size-ok` first-line waiver with UTC timestamp, +200 threshold
  advance, `archive.md` / `research/` exempt. Drop the incorrect token gloss.
- Mirror into this repo's own `specflow/procedures/spec-edit.md` (dogfood install).
- Update the `spec-edit` SKILL.md trigger text if its size-watch summary goes stale.
- Confirm `specflow:size-ok` cannot collide with the region (`specflow:start\b`) or composition
  (`specflow:full-only:`) regexes in `internal/kit/kit.go`.

### Files this batch creates/edits
- `templates/base/specflow/procedures/spec-edit.md`, `specflow/procedures/spec-edit.md`,
  `templates/agents/claude/.claude/skills/spec-edit/SKILL.md`.

### Verification
- Marker-collision test for `specflow:size-ok` against the region + composition regexes.
- `go test ./...` green; `specflow verify` clean on this repo after the dogfood mirror.

**Dependency: satisfied.** Shared the `spec-edit` SKILL.md and procedure template with **SL**, which
has landed (`3b265e0`). Note that SKILL.md now carries `specflow:full-only` markers around its
queue-persistence sentence — keep them intact when editing the size-watch summary.

---

## Batch 4 — README badges + file-map

**Goal.** Make the repo legible at a glance and the file-contract obvious to a newcomer.

### Deliverables
- CI + license badges (a release/version badge once published).
- A **pretty, simple file-map** in the README: each file specflow creates, what it does, and the
  flow the agent follows over them (spec → queue → claim → build). Directed by the user.
- **Deferred:** an animated demo (GIF/asciinema) — design the visual first before producing it.

### Files this batch creates/edits
- `README.md`.

### Verification
- Render check; links resolve; a new reader can follow the file-map without prior context.

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
