<!-- specflow:start - managed by specflow; do not edit inside these markers (your edits block specflow upgrade). Add your own notes outside them. -->
# Procedure: finish a claimed batch

Run this when wrapping up. `AGENTS.md` carries only the pointer to this file.

> **Commit & push follow the configured levers** (`config.commit` / `config.push` in
> `specflow/config.json`; see `AGENTS.md` → *Commit & push authority*). Wherever a step below says
> "commit" or "push": if `commit: user`, don't commit — alert the user and hand them the suggested
> message; if `push: user`, commit but don't push. Default is `agent` / `agent`.

## Wrap-up

1. Make the final work commit and push. **Note its short SHA** — you'll record it in `CLAIMS.md`.

   ```
   git log --oneline -1     # capture the SHA
   ```

2. Edit `CLAIMS.md`. Move the batch's entry from `## In progress` to the **top** of
   `## Completed`, adding:

   ```
   - Finished: YYYY-MM-DD HH:MM
   - Commit: <short SHA>
   ```

   Same UTC convention as `Started:`. Keep any `Handoff note:` / `Reclaim note:` lines — they're
   part of the historical record.

3. Add a "What shipped" summary under the entry — what changed, where, any manual prereqs,
   verification steps, follow-ups deferred. Match the format of existing `## Completed` entries.
   The point: a future agent (or you, after a context reset) can reconstruct the batch's outcome
   from this entry alone.

4. **Move the batch out of `BUILD_QUEUE.md`.** That file lists only *un-done* batches — a
   completed batch must not linger there or the next agent re-reads it as open work. Three edits:
   - Delete the batch's full section from `BUILD_QUEUE.md`.
   - Add a one-paragraph summary to `specflow/history/BUILD_QUEUE_DONE.md` (match the existing compact style —
     what shipped + key commit).
   - Drop the batch from any **pick-order pointer** line at the top of `BUILD_QUEUE.md`.

5. Commit `meta: complete batch-N` — covering `CLAIMS.md` **+ `BUILD_QUEUE.md` +
   `specflow/history/BUILD_QUEUE_DONE.md`** — and push.

## Hand the context back (if your agent supports context compaction)

6. If you're about to compact/clear context, suggest keeping the *durable* takeaways and
   dropping the execution detail:

   - **Keep:** durable artifacts shipped (new patterns, infra, decisions reaffirmed) + a
     forward pointer to the next likely batch and why.
   - **Drop:** blow-by-blow execution detail, specific file paths / SHAs (git + `CLAIMS.md`
     own those), resolved debug threads, intermediate states.

   If nothing about the closed batch is worth preserving, say so plainly.

## Next

7. Decide: claim the next eligible batch (run `claim-batch.md`) or stop. Either is fine — don't
   auto-chain unless the user asked you to.
<!-- specflow:end -->
