# Build Queue

Reference spec: [`spec/`](spec/README.md)
Agent work tracking: `CLAIMS.md` (managed by coding agents)
Completed history: [`specflow/history/BUILD_QUEUE_DONE.md`](specflow/history/BUILD_QUEUE_DONE.md) — the full narrative of each shipped batch.

## How this works

- This file lists only **un-done batches**, in full. A completed batch's section leaves this file
  and its narrative lands whole in `specflow/history/BUILD_QUEUE_DONE.md`; `CLAIMS.md` keeps a stub
  pointing there (git log + `specflow/history/CLAIMS_DONE.md` hold the implementation history).
- Dependencies are listed where they exist — the agent decides execution order.
- Agents claim and track completion in `CLAIMS.md`. **No Owner / Started / Status ever goes here.**
- See `specflow/procedures/claim-batch.md` before claiming.

---

## Un-done batches

> **Pick-order pointer.** Current release: **`v0.1.9`**; no line is open. Which batch shipped in
> which release lives in `spec/roadmap.md` → *Release lines*, and the milestone goals live there
> too, not here. This file holds un-done work only.
>
> **Claimable: FL**, then **OD**, which depends on it. FL stops the preamble filling at finish time
> (subtractive queue edit, bounded replace-only pointer block); OD lets a batch declare it is waiting
> on what an earlier batch *found*, which is the other reason facts get parked in queue prose. The
> queue warning is diagnostic and section 3 greps before it judges, but both of those are cleanup:
> these two are the ones that change the slope.
> **Not ready:** **NX** (`next` file spread) · **W** (workflow config) · **NB** (`--new-batch`) ·
> **E** (enforcement, research-first) · **P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch FL - a finish leaves the queue smaller

**Goal.** `BUILD_QUEUE.md`'s preamble fills at finish time, one appended status paragraph per batch,
under a cap that only reports after the fact. Batch QD made the fill visible and Batch PD made the
cleanup cheap; neither changes the slope. The install both came from accumulated 446 preamble lines
across roughly 50 batches, about 9 lines per finish, so a pruned queue there is back over the
45-line cap inside five batches. Stop the append.

- **`finish-batch` step 4 says the queue edit is subtractive.** Delete the section, rewrite the
  pointer, never append. The narrative already has a home and `finish` already writes it there.
- **The pick-order pointer becomes a bounded, replace-only block**, inside
  `<!-- specflow:pointer:start -->` / `end` markers with its own cap (propose 20 lines), measured by
  `Weigh` and reported beside the preamble count. An append-only region under a cap is the whole
  bug: bound it by a number rather than by discipline. Markers are the same shape the managed-file
  regions already use, and an install without them keeps working (absent markers mean no separate
  measurement, not an error).
- **The queue template gains one line** in `How this works`: a fact that will outlive the batch goes
  to `spec/`, not here.
- **`specflow finish` reports whether the preamble grew** during the batch (it already reads and
  rewrites the file, so this is a diff of two counts). Report, not refuse: the cheapest enforcement
  that needs no new state, and the point where an agent can still fix it.

### Files this batch creates/edits
- `templates/base/specflow/procedures/finish-batch.md` · `templates/base/BUILD_QUEUE.md` ·
  `templates/agents/claude/.claude/skills/finish-batch/SKILL.md` · `internal/kit/queue.go`
  (`Weight`, `Weigh`, the pointer block) · `cmd/specflow/main.go` (`printWeight`, `finish` output) ·
  `cmd/specflow/main_test.go` · `spec/architecture.md` (Ledger lifecycle: the preamble is bounded
  because the append is, run through `spec-edit.md`) · this repo's managed copies + `config.json`
  baselines.

### Does NOT touch
- `prune-ledgers.md` (PD just rewrote section 3) and the batch format itself (Batch OD).

### Verification
- `test -z "$(gofmt -l cmd internal)" && go vet ./... && go test ./...`
- New tests: the pointer block is measured and warned about separately; a queue without markers is
  reported exactly as it is today; `finish` names a preamble that grew.
- `specflow verify` clean after a self-hosted `upgrade`.

---

## Batch OD - a batch can wait on an outcome, not just on a finish

**Depends on:** Batch FL (both edit `Weigh` and `templates/base/BUILD_QUEUE.md`).

**Goal.** `Depends on:` takes batch ids and means *completed*, so there is no way to say a batch
waits on what an earlier batch **found**. Downstream, two batches declare `Depends on: ... Batch 46
having come back answers`; Batch 46 shipped and reported package presence only, so `specflow next`
offers both as claimable and is wrong. The only thing standing between an agent and a wrong claim
there is a prose caveat in the queue preamble, which is itself preamble growth: the format could not
express the fact, so it was parked in prose. That is the second inlet, and FL closes only the first.

**Build a `Blocked on:` field**: free text, makes the batch non-claimable, and `next` prints the
text as the block reason. Preferred over documenting `[NOT READY]` plus a comment, because the gate
stays machine-readable and the reason lands where the agent already looks. Settle the shape in a
`spec-edit` first (the field name, whether it coexists with a tag, what clears it), then build.

### Files this batch creates/edits
- `spec/architecture.md` or `spec/open-questions.md` (the decision, via `spec-edit.md`) ·
  `internal/kit/queue.go` (parse + eligibility) · `cmd/specflow/main.go` (`next` reason) ·
  `templates/base/BUILD_QUEUE.md` (declared shape) · `templates/base/specflow/procedures/claim-batch.md`
  (the eligibility rules the verb implements) · `cmd/specflow/main_test.go` · managed copies +
  baselines.

### Does NOT touch
- `Depends on:` semantics. A batch id still means completed; this is a second, separate gate.

### Verification
- `test -z "$(gofmt -l cmd internal)" && go vet ./... && go test ./...`
- New tests: a `Blocked on:` batch is never offered and its reason is printed verbatim; clearing
  the line makes it claimable; `--json` carries the field.

---

## Batch NX `[NOT READY]` — `specflow next` prints each batch's file spread

**Why `[NOT READY]`:** raised as optional; the decision is open in `spec/open-questions.md` →
*CLI / upgrade behavior*. Promote once the user calls it.

**Goal.** `next` already prints each batch's declared file list verbatim, which goes long and reads as
a wall. Replace it with a **spread**: the file count plus the distinct top-level paths, so an over-wide
batch (`spec/architecture.md` → *Batch size*) is visible at a glance **before** it is claimed. Read-only,
no new state; `--json` keeps the full list. (The companion from the same report — warning when
`CLAIMS.md` `## Completed` holds more than five entries — shipped in Batch LW, so NX no longer
carries it.)

### Files this batch creates/edits
- `cmd/specflow/main.go` (`cmdNext`) · `internal/kit/queue.go` · `cmd/specflow/main_test.go`.

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
