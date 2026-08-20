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
> install section) and **PR** (ledger pruning: the `prune-ledgers` procedure + skill that keeps
> `CLAIMS.md` to its 5 newest completed entries, archiving the rest).
> **`v0.1.5`** (patch) ships **CE** (per-batch context cost: the procedures name the cheap read, plus
> the `config.check` field) and **QV** (the `next` / `claim` / `finish` queue verbs). **RD** (a pushed
> tag publishes the release directly; the agent needs the user's approval to cut one) landed between
> v0.1.4 and v0.1.5 but is **repo-internal and ships nothing to users**, so it opened no version line:
> it touched `.goreleaser.yaml` plus this repo's own ledgers and spec. Don't open a new version line
> for a batch until it changes something a user installs.
> **Current release: `v0.1.5`. A version line is now open, unreleased.**
> **Claimable now: Batch AF** (adapter files upgrade like everything else) — the create-once hole
> that freezes the skill stubs and the handoff hook at whatever shipped on install day. It ships to
> users, so the next tag carries it. The number is the user's call at tag time, as is the tag itself.
> Every other batch below is `[NOT READY]`.
> **Post-v0.1 queue below:**
> **Batch W** (workflow config) · **Batch NB** (`--new-batch`) · **Batch E** (enforcement — research-first) ·
> **Batch P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch AF — Adapter files upgrade like everything else

**Depends on:** none.

**Goal.** Close the create-once hole in `upgrade`. Spec: `spec/architecture.md` → *init / upgrade*
→ "Two managed tiers, one drift contract".

`MANAGED` (`internal/kit/kit.go`) covers `AGENTS.md`, `specflow/procedures/`, and the per-agent
instruction file — every one of them a file with a marker-wrapped region. The **adapters** have no
region: the four Claude skill stubs and `.claude/hooks/specflow-handoff-reminder.sh` are wholly
generated, and `upgrade` only ever *places* them when absent (`missingAdapterFiles`) or replaces
them on a spec-only mode leak (`staleAdapterFiles`). So a repo installed months ago still runs the
skill stubs and hook that shipped that day, and no number of upgrades will move them. This repo is
the proof: its stubs predate the queue verbs that v0.1.5 shipped into the procedures.

Five parts, in dependency order — 1 is the mechanism and 2–5 ride on it:

1. **Whole-file management.** `init` records a SHA-256 of each rendered adapter file in the stamp's
   `managed` map. `upgrade` then treats them like regions: clean → replace and re-record; drifted →
   leave, write `.specflow-new`. Mode-leak replacement stays and still overrides drift.
2. **One-time adoption for existing installs.** An adapter with no baseline (every install shipped
   before this batch) is adopted on the next upgrade: byte-identical to the current template →
   record the baseline silently; anything else → back up to `.specflow-bak`, then replace. This is
   the part that makes the fix reach repos already using specflow, and it must be exercised by a
   test that starts from a v0.1.5-shaped stamp (`managed` holding region entries only).
3. **`verify` covers the adapters.** Today a deleted, truncated, or hand-edited stub or hook passes
   clean — the loop walks `managedEntries` only, and the one extra pass (`staleAdapterFiles`)
   `continue`s on an unreadable file. Missing → warning (that agent loses the trigger, specflow
   still works); drifted → warning naming `.specflow-new`; otherwise OK.
4. **`status` distinguishes stale from drift.** `status` reports `drift: none` off the region
   hashes while a stub is versions behind. Add a **stale** list: files whose on-disk hash matches
   their baseline but not the current template — i.e. exactly what `upgrade` would refresh — so the
   summary stops claiming a repo is current when it isn't.
5. **Teach the stubs the verbs.** Each `SKILL.md` carries an "In short:" summary, and a skill is
   what an agent loads *before* the procedure, so that summary is what gets acted on. All four
   describe hand-editing markdown only: `finish-batch`'s says "move the `CLAIMS.md` entry to
   `## Completed` … delete the batch from `BUILD_QUEUE.md`" without mentioning that
   `specflow finish <id> --commit <sha>` does all of it and prunes as well. One fast-path line per
   stub, matching the wording the procedures already use. (Alone this reaches new installs only —
   it needs 1 + 2 to land anywhere else, which is why it's in this batch and not its own.)

### Files this batch creates/edits
- `internal/kit/kit.go` · `cmd/specflow/main.go` · `cmd/specflow/main_test.go` ·
  `templates/agents/claude/.claude/skills/{claim-batch,spec-edit,finish-batch,prune-ledgers}/SKILL.md`
  · `.claude/skills/{claim-batch,spec-edit,finish-batch,prune-ledgers}/SKILL.md` (the dogfood copies)

### Verification
- `init` a temp repo, confirm the stamp's `managed` map now carries the five adapter paths.
- Hand-build a v0.1.5-shaped install (region baselines only, old stub contents); `upgrade`; confirm
  a pristine stub is replaced with no `.specflow-bak` and an edited one is backed up then replaced.
- Edit a stub *after* it has a baseline; `upgrade`; confirm it is left alone with a `.specflow-new`
  sidecar, and that `verify` and `status` both report it.
- Delete a stub; `verify` warns. `go test ./...`.

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
