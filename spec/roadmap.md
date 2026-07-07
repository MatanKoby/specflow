# Roadmap

Two things live here: **milestones** (near-term release goals + which batches make them up) and
**deferred directions** (decided in principle, just not now).

## Milestones

A **milestone** groups batches toward a release (v0.1, v0.2, …). To stay DRY, specflow keeps **no
separate milestone artifact that restates the queue** — a milestone lives in the same two-place split
every batch uses: its **goal + definition-of-done here**, and its **member batches grouped under a
`Milestone:` heading in `BUILD_QUEUE.md`** (the queue stays the single source of which batches exist
and their status). A milestone maps to a release tag (v0.1 → `v0.1.0`).

### v0.1 — first live-testable release
**Goal:** a user can adopt specflow on an **empty or small repo** (any language) and trust it with
real code. **Done when:**
- `init` is brownfield-aware (inject-with-consent, never commits, review-the-`git diff` handoff,
  requires git) and **spec-only** mode works;
- a **released binary** (GitHub Releases + `curl|sh`) — no build-from-source;
- the safety floor holds (non-destructive; a missing baseline never overwrites; git required);
- supporting commands land: `add-agent`, `status`, `verify` (installation), `--dry-run`; commit/push
  levers honored; clean piped output (NO_COLOR).

Medium/large-repo optimization is explicitly **out of scope**. Member batches: `BUILD_QUEUE.md` →
*Milestone v0.1* (CFG · BI · SO · G2 · add-agent · status · `--dry-run`).

## Deferred directions

- **Standalone demo repo.** A separate GitHub repo demonstrating a real specflow-managed project,
  kept **outside** the specflow repo on purpose — so it never rides along in the `npx github:` clone
  or the npm package. To be designed later. The root `README.md` file-map (Batch 4) covers the
  immediate "what do these files do / how does the agent move through them" need.
- **`--new-batch` quick flow.** A "now-to-now" command (Batch NB) — see `BUILD_QUEUE.md`.
- **Remote spec sources.** Known integrations (e.g. Confluence, and other doc/wiki systems) that
  import spec or documentation from external systems into `spec/`. Feeds the adoption flow
  (`adoption.md` → Pull from existing docs) for teams whose design already lives elsewhere.
- **Hosted / SaaS tier.** See `architecture.md` → SaaS frontier: a producer that authors/syncs
  `spec/` + `BUILD_QUEUE.md` into a repo; the file-contract is the API between the tiers.

## Research inputs

Pre-design research that feeds these milestones and directions lives in `research/` (dated
snapshots; see `research/README.md`). Current: `research/2026-07-competitive-landscape.md` (the
better-specs / maintenance / backfill landscape, gap analysis, and import-vs-copy verdict).
