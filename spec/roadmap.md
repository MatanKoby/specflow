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

## Release lines

Which batches shipped in which patch release, kept here rather than in `BUILD_QUEUE.md`: the queue
holds *un-done* work, and a version history parked in its preamble is exactly the sink
`architecture.md` → *Ledger lifecycle* bounds. The queue keeps only the pick-order pointer.

**Milestone v0.1 shipped** 🚀 — `v0.1.0` tagged and published, `curl|sh` verified end to end.
Member batches: **CFG** (config + commit/push levers + safety), **BI** (brownfield `init` + `verify`
+ `_DONE` relocation), **SO** (spec-only mode), **G2** (release: GoReleaser → GitHub Releases +
`curl|sh`; Homebrew deferred), **1** (add-agent), **2** (status), **5** (`--dry-run`).

| Release | Ships |
| --- | --- |
| `v0.1.1` | **FH** (finish-batch step-6 handoff rework) · **RF** (research-flow convention) |
| `v0.1.2` | **CH** (Claude-Code step-6 handoff hook: a `PostToolUse` backstop that blocks the loop after a `meta: complete batch-*` commit), plus the `upgrade` convergence that delivers newly-shipped non-managed adapter files to existing installs |
| `v0.1.3` | **SL** (spec-only mode no longer names the queue/claim machinery it omits, plus the `verify` mode-consistency check and an `upgrade` repair path) · **SZ** (the spec-file 600-line cap became a stop-and-ask with a `specflow:size-ok` waiver that re-asks every +200) |
| `v0.1.4` | **4** (README rewrite: badges, file-map, agent-executable install) · **PR** (ledger pruning: the `prune-ledgers` procedure + skill) |
| `v0.1.5` | **CE** (per-batch context cost + `config.check`) · **QV** (the `next` / `claim` / `finish` queue verbs) |
| `v0.1.6` | **AF** (the adapters — skill stubs and the handoff hook — are managed as whole files, carried across by a one-time adoption on the next `upgrade`) |
| `v0.1.7` | **CD** (batches are sized by the layers they cross; the prune check runs at claim as well as at finish) |
| `v0.1.8` | **LW** (ledger weight: the `CLAIMS.md` stub + archived narrative, the queue-preamble cap, and weight reporting in `next` / `verify`) |

**A batch only opens a version line when it changes something a user installs.** Two batches landed
without opening one: **RD** (a pushed tag publishes the release directly) between v0.1.4 and v0.1.5,
and **RN** (the release body is authored in the release commit) after v0.1.6. Both are repo-internal
— `.goreleaser.yaml`, this repo's own ledgers and spec — and ship nothing to users.

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
