# Specification

> Replace this paragraph with one or two sentences describing what this project is — the
> product identity a new reader (or agent) needs before reading anything else.

The spec is split across concern-focused files. Each file is small and edited as a unit. When
two sections always change in tandem, they belong in the same file. Edit via the procedure in
`specflow/procedures/spec-edit.md`.

## Files

<!-- One line per spec file, naming its concern. Add files as the design grows; when a
     sub-folder appears, give it its own README and point to it here instead of listing every
     file. Example:

- **`architecture.md`** — tech stack, deployment, the major moving parts.
- **`schema.md`** — data model.
- **`flows.md`** — end-to-end flows.
- **`roadmap.md`** — post-MVP / deferred work.
- **`archive.md`** — historical content no longer reflecting live code.
-->

## Reading order

For someone new: README → (the architecture/overview file) → the files for the area in front
of them. Don't read everything.

For an agent claiming a batch: read the queue entry first, then the 2–4 spec files relevant to
the batch's domain. Pull a sub-folder README before reading individual files there.

## Editing convention

Edit the file matching the concern. If a change crosses multiple files, that's a signal the
concern might be miscarved — flag it before duplicating content. Cross-reference by file path
rather than restating. Move historical context to `archive.md` when it stops being part of the
live system. Full procedure: `specflow/procedures/spec-edit.md`.
