---
name: prune-ledgers
description: Use when CLAIMS.md or BUILD_QUEUE.md has grown long, when a specflow weight warning fires, when finish-batch reaches its prune step, or when the user asks to prune/archive the ledgers by hand. Moves older completed CLAIMS entries into specflow/history/CLAIMS_DONE.md (keeping the 5 most recent), sweeps completed or dissolved sections out of BUILD_QUEUE.md, and audits the queue preamble against its size cap.
---

# Prune the ledgers

Follow **`specflow/procedures/prune-ledgers.md`** in this repo — that file is the authoritative,
up-to-date procedure (kept in sync by `specflow upgrade`; this skill is a thin trigger).

In short: `CLAIMS.md` `## Completed` keeps its 5 newest entries and the rest move verbatim to the
top of `specflow/history/CLAIMS_DONE.md` (never touch `## In progress`) → sweep `BUILD_QUEUE.md` of
sections whose batch is already completed, dissolved, or absorbed, leaving `[NOT READY]` /
`[DEFERRED]` / `[MANUAL]` alone → audit the queue **preamble** (everything above the first
`## Batch`) against its 45-line cap → commit `meta: prune ledgers (N claims entries archived)`.

**`specflow finish` already applies the `CLAIMS.md` half** as part of completing a batch, so after
that verb the only thing left here is the `BUILD_QUEUE.md` sweep, which has no verb and is done by
hand.

An install that predates this procedure may be many entries over: archive them all in one
catch-up pass. Lossless and mechanical, so no stop-and-ask — **except** the preamble audit, which
asks, because deciding which `spec/` file should own a stranded paragraph is a judgment call.
