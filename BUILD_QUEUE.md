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

> **Pick-order pointer.** Current release: **`v0.1.7`**; **`v0.1.8` is open** (Batch LW).
> Which batch shipped in which release lives in `spec/roadmap.md` → *Release lines*, and the
> milestone goals live there too — not here. This file holds un-done work only.
>
> **Claimable:** **LW** (ledger weight).
> **Not ready:** **NX** (`next` file spread) · **W** (workflow config) · **NB** (`--new-batch`) ·
> **E** (enforcement, research-first) · **P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch LW — Ledger weight: bound the entry, not just the count

**Goal.** Pruning bounds the ledgers by **count** (`CLAIMS.md` to 5 completed entries;
`BUILD_QUEUE.md` to zero completed batches). Nothing bounds the size of a *single* entry, or the
prose that is not an entry at all. Reported from a downstream install running at its prescribed
retention of 5 with a 27 KB `CLAIMS.md`; reproduced here (`CLAIMS.md` 20 KB / 249 lines at
retention 5, and 59 of `BUILD_QUEUE.md`'s 149 lines sit above the first `## Batch`). Three parts:

1. **Write the narrative once.** `finish-batch` currently asks for prose about one batch twice, in
   two files, independently authored. `CLAIMS.md` keeps a **stub** — metadata, at most 8 lines of
   "What shipped", and a pointer — and the full narrative goes to
   `specflow/history/BUILD_QUEUE_DONE.md`. `specflow finish` grows `--stub-file` (with
   `--summary-file` kept as an alias) and rejects an over-length stub. No waiver: moving prose to
   the done-file is lossless, so there is no judgment call to put to the user.
2. **Cap the queue preamble.** Everything above the first `## Batch` heading is capped at 45 lines
   (template baseline is 30) with the `specflow:size-ok` stop-and-ask shipped in Batch SZ, re-asking
   every +15. `prune-ledgers` gains a third section that audits those paragraphs into
   delete / relocate-via-`spec-edit` / keep. The preamble is where an agent parks a durable fact when
   it cannot decide which spec file owns it: at finish time the queue is already open, and nothing
   ever prunes it.
3. **Report weight.** `specflow next` and `specflow verify` report ledger line counts and warn past
   the bounds. Count is checked today; weight is not, and weight is what reached 27 KB while the
   count stayed correct. Absorbs the optional companion carried by **Batch NX** (warn when
   `## Completed` holds more than five entries), which NX no longer needs to carry.

### Files this batch creates/edits
- `spec/architecture.md` (*Ledger lifecycle*) · `spec/roadmap.md` (receives the relocated version
  lines) · `BUILD_QUEUE.md` · `specflow/procedures/{finish-batch,prune-ledgers}.md` ·
  `templates/base/specflow/procedures/{finish-batch,prune-ledgers}.md` ·
  `templates/base/BUILD_QUEUE.md` · `internal/kit/queue.go` · `cmd/specflow/main.go` ·
  `cmd/specflow/main_test.go`.

### Verification
- `test -z "$(gofmt -l cmd internal)" && go vet ./... && go test ./...`
- This repo's own preamble lands under the cap once its version lines move to `spec/roadmap.md`.

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
