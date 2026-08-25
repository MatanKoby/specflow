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

> **Pick-order pointer.** Current release: **`v0.1.9`** (**RC** · **MC** · **FS** · **ED** · **EM**);
> no line is open. Which batch shipped in which release lives in `spec/roadmap.md` →
> *Release lines*, and the milestone goals live there too, not here. This file holds un-done work only.
>
> **Claimable: QD** (queue diagnostics), then **PD** (prune-ledgers section 3), which depends on it.
> Both came out of a downstream install whose queue preamble reached 446 lines of shipped-batch
> narrative: the warning that caught it was read as a specflow bug, and the procedure that fixes it
> asks for judgment where a grep would do.
> **Not ready:** **NX** (`next` file spread) · **W** (workflow config) · **NB** (`--new-batch`) ·
> **E** (enforcement, research-first) · **P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch QD - the queue warning says what is wrong, not just how long

**Goal.** `specflow next` reports a preamble line count with no anchor, so a downstream install read
its 467-line warning as a bug in specflow rather than a finding about its own queue. Four
diagnostics, all read-only, all in `Weigh` so `verify` gets them too:

- **Address the count.** Name the boundary and the heading holding the bulk: `467 lines above the
  first "## Batch" heading (445 of them under "## Un-done batches")`.
- **Staleness, which is the number that actually diagnoses this.** Count preamble references to
  batch ids archived in `BUILD_QUEUE_DONE.md` against ones live in the queue, and warn when there
  are at least 3 archived and more archived than live. That fires on a short rotten preamble and
  stays quiet on a long current one, which a line count cannot do.
- **Claimability contradiction.** A preamble line saying a batch is claimable when the queue has no
  such section is a hard warning naming the line: it is the one that misleads an agent into
  claiming a batch that shipped weeks ago.
- **Near-miss headings.** A line that reads as a batch heading but fails the declared shape is
  parsed as prose, so it silently inflates the preamble count and never appears in `next`. Name its
  line number and the shape it missed.

`--json` carries every new number. No format change, no new state, nothing written.

### Files this batch creates/edits
- `internal/kit/queue.go` (`Weight`, `Weigh`) · `cmd/specflow/main.go` (`printWeight`) ·
  `cmd/specflow/main_test.go`.

### Does NOT touch
- The batch format, the parsers, or any procedure. Reporting only.

### Verification
- `test -z "$(gofmt -l cmd internal)" && go vet ./... && go test ./...`
- New tests: the address names the dominant heading; staleness fires on archived-heavy and stays
  quiet on a current pointer; a stale claimable line warns with its line number; a near-miss
  heading warns and is counted as prose.

---

## Batch PD - prune-ledgers section 3, duplication-first

**Depends on:** Batch QD (its warnings are what send an agent here, and section 3 should name them).

**Goal.** Section 3 tells the agent to sort every preamble paragraph into keep / relocate / delete
and put the piles to the user. Run against a real 446-line preamble, the relocate pile came back
**empty**: every durable fact was already carried by `BUILD_QUEUE_DONE.md` or `spec/`. The section
asked for 73 judgment calls to reach an answer that a grep settles. Rewrite it around that:

- **Step 1 is duplication, not judgment.** Grep the archives, `CLAIMS.md` and `spec/` for a
  distinctive phrase from each paragraph. A cited duplicate is a delete, mechanically, no ask: that
  is section 3's own losslessness bar, the same one sections 1 and 2 act on without asking.
- **Only uncited paragraphs reach the stop-and-ask**, and they are put as piles with counts, not one
  paragraph at a time.
- **Two presumptions worth stating:** a paragraph naming a `spec/**.md` path is usually pointing at
  the file that already owns it, and a paragraph naming an archived batch is usually history.
  Presumption, not proof; the citation is still required.
- **Prescribe the report shape**: line range, pile, evidence, one-line reason, plus a projected
  preamble line count and an explicit "content that lives nowhere else" section, which is the only
  part whose loss would be real.

### Files this batch creates/edits
- `templates/base/specflow/procedures/prune-ledgers.md` ·
  `templates/agents/claude/.claude/skills/prune-ledgers/SKILL.md` · this repo's managed copies
  (`specflow/procedures/prune-ledgers.md`, `.claude/skills/prune-ledgers/SKILL.md`) ·
  `specflow/config.json` baselines.

### Does NOT touch
- Sections 1, 2 and 4, and no Go code.

### Verification
- `test -z "$(gofmt -l cmd internal)" && go vet ./... && go test ./...` (unchanged, but the managed
  baselines must agree) plus `specflow verify` clean after a self-hosted upgrade.

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
