# specflow — Specification

**specflow** is a spec-driven, batch, claim-before-work protocol for AI coding agents, installable
into any repository with one command and honored by any agent (Claude Code, Cursor, Copilot, IBM
Bob, Google Antigravity, and anything that reads `AGENTS.md`). It is a small, file-based,
cross-agent layer: design lives in `spec/`, work is declared as batches in `BUILD_QUEUE.md`, and
every batch is claimed in git (`CLAIMS.md`) before code is written — so multiple agents or people
can share a branch without colliding, and progress survives a crashed machine.

> specflow uses specflow. This `spec/` (and the root `AGENTS.md` / `BUILD_QUEUE.md` / `CLAIMS.md`)
> are the project self-hosting its own protocol. They do **not** ship in the npm package — only
> `bin/` + `templates/` do (see `architecture.md` → Distribution).

The spec is split across concern-focused files. Each file is small and edited as a unit. Edit via
the procedure in `specflow/procedures/spec-edit.md`.

## Files

- **`architecture.md`**: what specflow is: the CLI + `templates/` layout, `init`/`upgrade`, the
  ownership split, the host-repo file-contract, the ledger lifecycle (how `BUILD_QUEUE.md` and
  `CLAIMS.md` stay bounded), spec organization + the 600-line cap, versioning/stamp, distribution,
  the SaaS frontier.
- **`workflow.md`**: the git/autonomy policy model: four orthogonal config dimensions, the profile
  on-ramps, the `config.md` rendering, and the `init` setup flow.
- **`adoption.md`**: bringing specflow onto existing, unspecced code (brownfield): incremental
  spec, optional assisted backfill, and how the agent behaves around code it didn't spec.
- **`open-questions.md`**: decisions still under discussion, kept here so they survive beyond any
  one conversation. Resolved items graduate into the file above whose concern they match.
- **`roadmap.md`**: milestones (near-term release goals + their batch groupings, e.g. v0.1) and
  deferred-but-intended directions (standalone demo repo, the SaaS tier). The single, authoritative
  home for roadmap content; never restated in the root README.
- **`research/`** holds dated research snapshots that feed roadmap/design decisions (competitive
  surveys, option/tradeoff analyses), written on the go so an abruptly ended session is never lost.
  Gate-free; conclusions graduate into `open-questions.md` / `roadmap.md`. See `research/README.md`.

## Reading order

New reader: README → `architecture.md` → `workflow.md`. An agent picking a batch: read the queue
entry, then the 1–2 spec files for its domain. Don't read everything.

## Editing convention

Edit the file matching the concern. If a change crosses files, flag it before duplicating —
cross-reference by path instead. When a decision in `open-questions.md` is settled, move it into
the matching design file (don't leave it in two places). Full procedure: `specflow/procedures/spec-edit.md`.
