# Build Queue

Reference spec: [`spec/`](spec/README.md)
Agent work tracking: `CLAIMS.md` (managed by coding agents)
Completed history: [`specflow/history/BUILD_QUEUE_DONE.md`](specflow/history/BUILD_QUEUE_DONE.md) - the full narrative of each shipped batch.

## How this works

- This file lists only **un-done batches**, in full. A completed batch's section leaves this file
  and its narrative lands whole in `specflow/history/BUILD_QUEUE_DONE.md`; `CLAIMS.md` keeps a stub
  pointing there (git log + `specflow/history/CLAIMS_DONE.md` hold the implementation history).
- Dependencies are listed where they exist - the agent decides execution order.
- Agents claim and track completion in `CLAIMS.md`. **No Owner / Started / Status ever goes in
  this file** - that's execution state, and it lives in `CLAIMS.md` only.
- Batches are designed so two agents can work different batches at once without file conflicts.
- See `specflow/procedures/claim-batch.md` before claiming.
- Everything above the first `## Batch` heading is the **preamble**, capped at 45 lines. It holds
  the pick-order pointer and these rules - not design facts or release history, which belong in
  `spec/`. `specflow/procedures/prune-ledgers.md` (section 3) audits it.
- **A fact that will outlive the batch goes to `spec/`, not here** (`spec-edit.md`): parking it in
  the preamble, where nothing prunes it, is how this file fills.
- Each batch below follows a **declared shape**: the `## Batch <id> [TAG] - <title>` heading, an
  optional `**Depends on:** Batch X[, Batch Y]` line, an optional `**Blocked on:** <text>` line
  (free text; the batch is not claimable while it is there), and a
  `### Files this batch creates/edits` list. Everything else in a section is free prose.
  `specflow next` reads that shape to answer eligibility, and reports a batch missing a field
  rather than treating it as claimable.

---

## Un-done batches

<!-- specflow:pointer:start - replace-only, 20 lines max: rewrite this block, never append to it. -->
> **Pick-order pointer for "continue".** When the user types "continue" after a context clear,
> **ask** which un-done batch to claim rather than guessing. (Keep a rough priority order here
> as the queue fills.)
<!-- specflow:pointer:end -->

---

## Batch 1 - <short title> (example: replace or delete)

> This is a worked example showing the batch shape. Delete it once you have real batches.

**Depends on:** none. Batch ids only, and each one means *completed*.

To gate on something a batch id cannot express (an outcome an earlier batch was supposed to
establish, an answer you are waiting on), add a `**Blocked on:** <what it waits on>` line. It is free
text, `specflow next` prints it as the reason, and deleting the line is what clears it - which only
the person who can answer it should do.

**Goal.** One or two sentences: what this batch delivers and why.

### Deliverables
- Concrete, checkable outcomes - not "work on X" but "X does Y, verified by Z".

### Files this batch creates/edits
- `path/to/file` - what changes. (This list is what the parallelism check and `specflow next` read,
  so keep it honest. Backticked paths; `dir/{a,b}.md` stands for both files.)

### Does NOT touch
- Files deliberately out of scope, so a parallel batch knows it's safe.

### Verification
- How to confirm it works (test, manual step, query).
