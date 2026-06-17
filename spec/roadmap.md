# Roadmap

Deferred-but-intended directions — *not* open questions (these are decided in principle), just not
now.

- **Standalone demo repo.** A separate GitHub repo demonstrating a real specflow-managed project,
  kept **outside** the specflow repo on purpose — so it never rides along in the `npx github:` clone
  or the npm package. To be designed later. The root `README.md` file-map (Batch 4) covers the
  immediate "what do these files do / how does the agent move through them" need.
- **`new-batch` / `--nb` quick flow.** A "now-to-now" command (Batch NB) — see `BUILD_QUEUE.md`.
- **Hosted / SaaS tier.** See `architecture.md` → SaaS frontier: a producer that authors/syncs
  `spec/` + `BUILD_QUEUE.md` into a repo; the file-contract is the API between the tiers.
