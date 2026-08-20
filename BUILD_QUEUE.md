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
> **Current release: `v0.1.4`. No version is pending.** **RD** (a pushed tag publishes the release
> directly; the agent needs the user's approval to cut one) landed after v0.1.4 but is **repo-internal
> and ships nothing to users**: it touched `.goreleaser.yaml` plus this repo's own ledgers and spec,
> no `templates/`, no Go source, no `install.sh`. So the binary and the installed kit are unchanged
> and there is nothing for `specflow upgrade` to deliver. Don't open a new version line for a batch
> until it changes something a user installs.
> **Pick next: Batch CE**, then **Batch QV** (which depends on it). Both come from the turn-cost
> analysis in `architecture.md` → *Context economy* and *Queue verbs*, and both change what users
> install, so **a version line opens when CE ships** — the number is the user's call at tag time.
> **Post-v0.1 queue below:**
> **Batch CE** (context economy + `config.verify`) · **Batch QV** (queue verbs) · **Batch W**
> (workflow config) · **Batch NB** (`--new-batch`) · **Batch E** (enforcement — research-first) ·
> **Batch P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch CE — Context economy + `config.verify`

**Spec:** `spec/architecture.md` → *Context economy — the read side of the ledger* and
*Config & state* (`verify`).

**Goal.** Cut the recurring per-batch context cost that the procedures currently cause. Measured on
this repo, an eligibility check that needs 419 bytes of headings reads 17.2 KB when the agent `cat`s
`CLAIMS.md`, and both ledgers are read 3 to 5 times per batch.

1. **Read-shape steps in the procedures.** `claim-batch.md`, `finish-batch.md`, and
   `prune-ledgers.md` name the cheap read inline (grep the headings, then slice the one section)
   wherever they currently say only *what* state to check. `spec-edit.md` says the same for a
   `spec/` file: headings first, then the matching section.
2. **An economy section in `AGENTS.md`.** Batch independent reads into one turn; never re-read to
   confirm your own write; read the batch's declared file list before opening anything else. Keep it
   short: this text loads in every session in every install.
3. **`config.verify`.** A new `config` string: the repo's single check command. `init` asks for it
   (skippable, empty when not configured), `status` shows it, and `finish-batch.md` says to run it
   before the final commit *only* when it is set. specflow never validates or executes it.

**Note.** Item 3 is worth zero in a repo with no check suite, and item 2 is advisory (specflow can
ask an agent to batch its reads; it cannot enforce that). Item 1 is the deterministic one.

### Files this batch creates/edits
- `templates/base/specflow/procedures/{claim-batch,finish-batch,prune-ledgers,spec-edit}.md` ·
  `templates/base/AGENTS.md` · `cmd/specflow/main.go` (init prompt + `status` row) ·
  `internal/kit/kit.go` (stamp field) · `cmd/specflow/main_test.go` · the repo's own `AGENTS.md` +
  `specflow/procedures/**` via self-hosted `upgrade`.

### Verification
- `go test ./...`; `init` into a temp repo with and without a verify answer, asserting the stamp and
  that spec-only mode still omits the queue machinery; `status` renders the new row; the repo's own
  managed regions refresh with no drift.

---

## Batch QV — Queue verbs (`next`, `claim`, `finish`)

**Depends on:** Batch CE (both edit `cmd/specflow/main.go`, `internal/kit/kit.go`, the procedures,
and `main_test.go`; CE also settles the procedure wording these verbs then shortcut).

**Spec:** `spec/architecture.md` → *Queue verbs — the CLI as the agent's hands* + *Declared batch
fields*.

**Goal.** Move the deterministic file surgery out of agent turns and make the ledger format
machine-guaranteed for every agent, not just the one that wrote the last entry.

1. **Declared batch fields.** Parse the fixed shape out of `BUILD_QUEUE.md` (heading + id + tag,
   optional `**Depends on:**`, `### Files this batch creates/edits`). Forgiving and line-oriented;
   a batch missing a field is reported unparseable, never silently claimable.
2. **`specflow next [--json]`** — read-only eligibility: tag, not already claimed, dependencies
   satisfied (`CLAIMS.md` `## Completed` **or** `CLAIMS_DONE.md`), no file overlap with anything in
   progress.
3. **`specflow claim <N>`** — write the `## In progress` entry (Owner from `config.agents`, `Started`
   in UTC).
4. **`specflow finish <N> --commit <sha> [--summary-file <path>]`** — move the entry to `## Completed`
   with `Finished` + `Commit` + the agent's summary, delete the batch from `BUILD_QUEUE.md`, append
   the agent's paragraph to `BUILD_QUEUE_DONE.md`, prune `CLAIMS.md` to its 5 newest.
5. **Procedures reference the verbs as the fast path**, keeping every manual step so non-CLI agents
   and hand edits still work.

**Constraints.** No verb commits (the `commit` / `push` levers own that). No verb writes prose a
human reads. Never lose a user's hand edit: unparseable input stops with a message rather than
rewriting the file.

### Files this batch creates/edits
- `cmd/specflow/main.go` · `internal/kit/kit.go` · `cmd/specflow/main_test.go` ·
  `templates/base/BUILD_QUEUE.md` (demonstrate the declared shape) ·
  `templates/base/specflow/procedures/{claim-batch,finish-batch,prune-ledgers}.md`.

### Verification
- Table-driven parser tests over malformed queues (missing fields, unknown tags, duplicate ids).
- Round-trip on a temp repo: `next` → `claim` → `finish` and assert both ledgers plus both archives
  match what the procedures describe by hand, including the prune boundary at 5.

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
