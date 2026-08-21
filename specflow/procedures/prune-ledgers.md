<!-- specflow:start - managed by specflow; do not edit inside these markers (your edits block specflow upgrade). Add your own notes outside them. -->
# Procedure: prune the ledgers into their archives

`BUILD_QUEUE.md` and `CLAIMS.md` are read on every batch, so both are **bounded working sets**, not
growing logs. This procedure keeps them that way. `AGENTS.md` carries only the pointer to this file.

> **Commit & push follow the configured levers** (`config.commit` / `config.push` in
> `specflow/config.json`; see `AGENTS.md` → *Commit & push authority*). Wherever a step below says
> "commit" or "push": if `commit: user`, don't commit, alert the user and hand them the suggested
> message; if `push: user`, commit but don't push. Default is `agent` / `agent`.

## When to run

- **`finish-batch` delegates here** at the end of every batch (its step 4a). That is the normal path.
  (`specflow finish` already applies section 1 as part of completing the batch, so after the verb the
  only thing left here is the section 2 queue sweep.)
- **When a weight warning fires.** `specflow next` and `specflow verify` report both ledgers' line
  counts and warn when one is past its bound — the count of completed entries, or the preamble cap in
  section 3. The warning names the section to run.
- **By hand, any time**, when a ledger has already overgrown: `prune-ledgers` as a skill, or just
  follow this file. An install that predates this procedure can be many entries over, so the first
  run is a **catch-up pass** that archives them all at once.

Pruning is never gated behind a stop-and-ask. Archiving is lossless and mechanical: the entry moves
one file away, unedited, and `claim-batch` already resolves a dependency against `CLAIMS.md`
`## Completed` **or** `specflow/history/CLAIMS_DONE.md`. Don't ask the user to approve it, and don't
ask them to pick the retention number.

## 1. `CLAIMS.md`: keep the 5 most recent completed entries

Count the `### Batch …` entries under `## Completed`. If there are **more than 5**, move every entry
past the newest 5 into `specflow/history/CLAIMS_DONE.md`.

`grep -nE '^## |^###|^- Finished:' CLAIMS.md` gives you the count, the ordering, and the line range
of every entry in one call. Slice from there; don't read the file whole to count its headings.

- **Order.** `## Completed` is newest-first, so the entries to move are the ones at the **bottom**.
  If the ordering has drifted, sort by `Finished:` before counting.
- **Move, never rewrite.** Copy each entry verbatim, including its `Owner` / `Started` / `Finished` /
  `Commit` lines, its "What shipped" summary, and any `Handoff note:` / `Reclaim note:` lines. Don't
  summarize, shorten, or reformat. The archive is the historical record.
- **Destination order.** `CLAIMS_DONE.md` is also newest-first: insert directly under its header, so
  the newest archived batch stays at the top.
- **Never touch `## In progress`.** An active claim is live state, however old it looks. A stale one
  is a handoff question for the user, not something to archive.

Retention is a **count, not a line or byte budget**: entries vary several-fold in length, a count
cuts on an entry boundary instead of severing a record, and two agents pruning independently reach
the same result. Rationale in `spec/architecture.md` → *Ledger lifecycle*.

## 2. `BUILD_QUEUE.md`: sweep what leaked past

`finish-batch` already deletes a finished batch from the queue as part of completing it (its step 4),
so the queue's retention is zero and this is only a sweep for what got left behind. Delete any
section for a batch that is:

- already present in `CLAIMS.md` `## Completed` or `specflow/history/CLAIMS_DONE.md`; or
- marked **dissolved**, **absorbed** into another batch, or otherwise no longer work to be done.

For each one, confirm `specflow/history/BUILD_QUEUE_DONE.md` carries its one-paragraph summary
(add it if `finish-batch` missed it), then remove the section and drop the batch from any
**pick-order pointer** at the top of the file.

**Leave `[NOT READY]`, `[DEFERRED]`, `[MANUAL]`, and standing non-batch sections alone.** They are
un-done work, which is exactly what the queue is for. Length is not a reason to prune them.

## 3. `BUILD_QUEUE.md`: audit the preamble

Everything above the **first `## Batch` heading** is the queue's preamble: the header links, the
"How this works" rules, and the pick-order pointer. Sections 1 and 2 bound *entries*; nothing bounds
this. It is where a durable fact gets parked when whoever wrote it could not decide which `spec/`
file owns it — at finish time the queue is already open, writing there is one edit, and no rule ever
comes back for it. So it fills.

**The cap is 45 lines**, counted from the top of the file to the line before the first batch:

```
awk '/^## Batch /{print NR-1; exit}' BUILD_QUEUE.md
```

Under the cap there is nothing to do here. Over it, sort each preamble paragraph into one of three
piles and put the result to the user:

- **Keep** — the pick-order pointer, and the rules that tell an agent how to read the file. That is
  what a preamble is for.
- **Relocate** — a durable design fact, a decision, a release history. It belongs in `spec/`: run
  `spec-edit.md` to find the file whose concern owns it, move it there, and leave behind a link only
  if a reader of the queue actually needs one.
- **Delete** — stale status (a version line saying "open, not tagged" after the tag was pushed),
  notes about a batch that has since shipped, anything the archives already carry.

**This section is a stop-and-ask**, unlike sections 1 and 2. Archiving an entry is mechanical, but
deciding which spec file should own a stranded paragraph is a judgment call about concerns — the same
call `spec-edit.md` never makes on its own authority. Show the three piles, then act on the answer.

**If the user chooses to keep it over the cap**, record the waiver as the **first line of the file**,
above the `#` heading, exactly as the spec-file cap does:

```
<!-- specflow:size-ok - user approved this preamble over 45 lines on 2026-01-31 14:05 UTC; next check at 60. -->
```

Timestamp in UTC, and set `next check` to the limit you just asked about **plus 15**. That is the
preamble's new limit. A waiver silences one threshold, never the rule.

## 4. Commit

Commit the prune on its own, so the diff stays readable and reviewable:

```
meta: prune ledgers (N claims entries archived)
```

Cover `CLAIMS.md`, `specflow/history/CLAIMS_DONE.md`, and, if section 2 or 3 changed anything,
`BUILD_QUEUE.md` + `specflow/history/BUILD_QUEUE_DONE.md` + whatever `spec/` file received a
relocated paragraph. Push.

When `finish-batch` delegated here, folding this into its `meta: complete batch-N` commit is fine
too. One commit or two, but never leave the prune uncommitted.
<!-- specflow:end -->
