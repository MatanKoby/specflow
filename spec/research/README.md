# Research notes

Dated snapshots of pre-design research: competitive surveys, prior-art scans, option and tradeoff
analyses that feed a future roadmap or design decision. This is the home for the **research** step
that precedes *idea* in the model (see `../workflow.md` → *Research notes (optional pre-idea input)*).

## Why this folder exists

- **Crash-safe backup.** A long research session in a terminal can die with the machine (a forced OS
  update, a crash). The agent writes findings here **as it goes**, so an abruptly ended session
  never loses the work. Same principle as `../../specflow/procedures/spec-edit.md` (the transcript
  is not durable), applied one step earlier in the pipeline.
- **A home for not-yet-design.** Research feeds design but is not design, so it fits neither `spec/`
  proper, `open-questions.md` (crystallized questions), nor the queue (needs ratified design).

## Rules

- **Gate-free.** A research note asserts no design, so the agent may write and update it without
  gate-1 approval. Only *graduating a conclusion* into `open-questions.md` / `roadmap.md` needs the
  user's ratification.
- **Dated snapshots, not living design.** A note is a record of what was true when written, so it is
  **exempt** from the "spec = current design, archive the stale" rule (`spec-edit.md`). Do not rewrite
  a note to stay current; write a new dated note instead.
- **Conclusions graduate upward.** When a candidate direction matures, promote it into
  `open-questions.md` (as a question) or `roadmap.md` (as a direction), exactly as open questions
  graduate into design files. The note stays behind as the evidence trail (like `archive.md` is the
  history trail).
- **Naming.** `YYYY-MM-topic.md`.

## Index

- [`2026-07-competitive-landscape.md`](2026-07-competitive-landscape.md) — landscape of tools for
  better specs / spec maintenance / backfill, with a gap analysis and an import-vs-copy verdict.
