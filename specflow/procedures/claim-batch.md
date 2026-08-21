<!-- specflow:start - managed by specflow; do not edit inside these markers (your edits block specflow upgrade). Add your own notes outside them. -->
# Procedure: claim a batch from `BUILD_QUEUE.md`

Run this before starting any new batch. `AGENTS.md` carries only the pointer to this file.

> **Commit & push follow the configured levers** (`config.commit` / `config.push` in
> `specflow/config.json`; see `AGENTS.md` → *Commit & push authority*). Wherever a step below says
> "commit" or "push": if `commit: user`, don't commit — alert the user and hand them the suggested
> message; if `push: user`, commit but don't push. Default is `agent` / `agent`.

## Pre-flight

1. `git pull --ff-only` on the shared working branch — if it fails, resolve before claiming.

## Eligibility

**Read both ledgers by their headings, not in full.** Every question in this section is answered by
headings plus one or two field lines, and both files get read again at finish and at prune, so
reading them whole is the largest recurring context cost in the protocol:

```
grep -nE '^## Batch' BUILD_QUEUE.md        # the candidates, with their tags
grep -nE '^###|^- Owner:' CLAIMS.md        # what is claimed, and by whom
```

Then slice the one section you actually need by line number (`sed -n '120,180p' BUILD_QUEUE.md`).
Read a file whole only when the headings genuinely don't answer the question.

**Prune first if `CLAIMS.md` has overgrown.** Pruning at finish alone doesn't help the agent that
claims next — it still reads the overgrown file on the way in, and that read is the cost. So test the
retention rule here too, before claiming:

```
sed -n '/^## Completed/,$p' CLAIMS.md | grep -c '^### '   # more than 5 → prune first
```

More than **5** completed entries means run `specflow/procedures/prune-ledgers.md` and commit the
prune on its own, then claim. This is the same retention count `finish` enforces — there is no second
threshold, and nothing to ask the user.

**Fast path: `specflow next`.** When the specflow CLI is on the machine, one read-only call answers
this entire section: it applies the tag, already-claimed, dependency, and overlap rules together and
prints every blocked batch with the reason (`specflow next --json` for the machine-readable form).
A batch missing its declared fields is reported as unparseable rather than quietly offered. The
steps below stay authoritative: they are what the verb is doing, and what to do without it.

2. Pick a candidate batch in `BUILD_QUEUE.md` (under "Un-done batches"):
   - **Skip** if it has an exclusionary tag: `[MANUAL]`, `[NOT READY]`, or any tag you don't recognize.
   - **Skip** if it's already listed in `CLAIMS.md` `## In progress` or `## Completed`.

3. **Dependency check.** If the batch lists `Depends on: Batch X[, Batch Y]`, verify each
   listed batch appears in `CLAIMS.md` `## Completed` (or `specflow/history/CLAIMS_DONE.md`). If any are
   missing, pick a different candidate.

4. **Parallelism check.** If any batch is currently `## In progress`, compare your candidate's
   "Files this batch creates/edits" against that batch's same field. If they overlap, pick a
   different candidate or wait.

## Claim

5. Edit `CLAIMS.md`. Add an entry to the **top** of `## In progress`:

   ```
   ### Batch N — <title>
   - Owner: <your agent name>
   - Started: YYYY-MM-DD HH:MM
   ```

   Use UTC for the timestamp (the convention every entry uses).

   **Fast path: `specflow claim <N>`** writes exactly that entry (heading, `Owner` from
   `config.agents`, `Started` in UTC) at the top of `## In progress`, and refuses any batch
   `specflow next` would not offer, so the eligibility rules hold either way. It does **not** commit:
   step 6 is still yours.

6. Commit `meta: claim batch-N (<agent>)` and push to the shared branch.

## Push-race recovery (rejected push on the claim commit)

If `git push` is rejected as non-fast-forward, another agent committed first. Recover
**without force-pushing**:

1. `git fetch` the shared branch.
2. `git reset --hard <remote>/<branch>` — drops your local claim commit. Safe because the only
   change was `CLAIMS.md`.
3. Re-read `CLAIMS.md`:
   - If your target batch is now `## In progress`, someone else has it — pick a different
     claimable batch and start over.
   - If your target is still unclaimed (they raced for a *different* batch), re-run this whole
     procedure from step 1 with the same target.

For a rejected push on a *work* commit (`batch-N: ...`), **don't reset** — `git pull --rebase`,
resolve conflicts, push again. **Never** `git push --force` on the shared branch.

## Mid-batch handoff (rare)

If you must stop before finishing:

1. Edit the batch's `## In progress` entry: change `Owner:` to `none`, add a `Handoff note:`
   line (what's done, what's left, files touched, gotchas).
2. Commit `meta: handoff batch-N` and push.

The next agent runs this same claim procedure but only updates `Owner:` (the original
`Started:` timestamp stays).

## Stale-claim recovery

If a batch has been `## In progress` with no new commits for >24h and you want to take over:

1. Update `Owner:` in the existing entry; add a `Reclaim note:` explaining why.
2. Commit `meta: reclaim batch-N from <prior owner>` and push.

Use sparingly — prefer to wait or ask the user.

## Doing the work

After step 6 you're the owner. Commit incrementally with `batch-N: <imperative>` messages and
push at sensible checkpoints. On any rejected push during work commits, `git pull --rebase`,
never force. When finishing, follow `finish-batch.md`.
<!-- specflow:end -->
