# Architecture

## What specflow is

A zero-dependency Node CLI (`bin/specflow.js`) plus a `templates/` tree. The CLI scaffolds the
protocol files into a host repo and refreshes them over time. There is no runtime library the host
imports — specflow's output is **plain markdown + git**, which is what makes it cross-agent.

```
specflow/                  (this repo — the tool)
  bin/specflow.js          the CLI: init, upgrade, --version  (+ planned: add-agent, status, new-batch/--nb, verify)
  templates/
    base/                  files every install gets (AGENTS.md, queue/claims skeletons, spec/, specflow/)
    agents/<agent>/        per-agent adapter stubs, copied when that agent is selected
  test/smoke.js            spawns the CLI against temp repos, asserts behavior
```

## Cross-agent model

`AGENTS.md` is the **universal base** every agent reads — it carries the full protocol. Per-agent
**stubs** are generated only for agents with their own mandatory config home: Claude Code
(`CLAUDE.md` + native skills), Cursor (`.cursor/rules`), Copilot (`.github/copilot-instructions.md`),
IBM Bob (`.bob/rules/`), Google Antigravity (`.agents/rules/`). Bob and Antigravity also read
`AGENTS.md` natively, so their stubs are reinforcement, not load-bearing. The three procedures live
once in `specflow/procedures/`; Claude's skills are thin triggers that point at them (single source).

specflow is deliberately **not** packaged as a Claude plugin — a plugin is Claude-only, and the
requirement is that every agent honors the same protocol.

## The host-repo file-contract

`init` writes, and the ownership boundary is the load-bearing decision:

| Path | Owner | On `upgrade` |
|---|---|---|
| `AGENTS.md`, `specflow/procedures/**` | **specflow** (mechanism) | managed region refreshed — **never user/agent text** |
| `specflow/.spec-batch.json` | specflow (stamp) | version bumped |
| `BUILD_QUEUE.md` / `BUILD_QUEUE_DONE.md`, `CLAIMS.md` / `CLAIMS_DONE.md` | user/agents (state) | untouched |
| `spec/**` | user (content) | untouched |
| per-agent stubs (`CLAUDE.md`, `.cursor/rules`, …) | user (after first write) | untouched |

**specflow owns the mechanism; the host owns content and state.** Hard invariant: `upgrade` is
**non-destructive — it never removes or overwrites text authored by a user or another agent**, in
any file. It refreshes only specflow's own marker-delimited region; everything outside is preserved.
(The current code still wholesale-overwrites — that's the redesign in Batch U.)

## init / upgrade

- **`init`** — interactive (or `--agents=…`/`--all`): pick agents, scaffold base + selected
  adapters, fill the stamp. Skips any file that already exists (never clobbers). Guards re-init.
- **`upgrade`** — refreshes specflow's managed mechanism to the installed kit version and bumps the
  stamp. **Hard invariant: `upgrade` never removes or overwrites text authored by a user or another
  agent, in any file.** The current code wholesale-overwrites `AGENTS.md` + `specflow/procedures/`,
  which violates this if those files were edited — so the mechanism is being **redesigned** around
  marker-delimited managed regions (`<!-- specflow:start -->` … `<!-- specflow:end -->`) + drift
  detection: only specflow's own region is replaced, everything else is preserved, and a hand-edited
  region is reported rather than clobbered. **Treat `upgrade` as provisional until Batch U lands.**

## Versioning & the stamp

One file: `specflow/.spec-batch.json` carrying `kitVersion` + `schemaVersion`. Procedure prose is
*instructions, not data*, so a refresh replaces specflow's managed region (never user/agent text —
see the `upgrade` invariant above) — no migration. `schemaVersion` gates the
rare case where the **format** of the state files changes; only then is a migration needed. No
migration runner is built until a format actually breaks (the file-contract is kept stable on
purpose to keep that rare).

## Distribution

- **Now:** runnable from GitHub — `npx github:MatanKoby/specflow init`. No registry needed.
- **Later:** publish to npm for `npx specflow init`. The package ships **only the `files`
  allowlist** (`bin/` + `templates/` + auto LICENSE/README/package.json). Because it's an allowlist,
  this repo dogfooding the protocol at its own root does not leak those files into the package.
- **Caveat:** a future Claude-plugin packaging must target `templates/agents/claude/.claude/`, never
  a dogfood `.claude/` at the repo root.

## SaaS frontier (later)

The git tier above is the core and stands alone offline. A hosted tier (`spec_for_ai-as-a-service`)
would be an upstream **producer** that authors/syncs `spec/` + `BUILD_QUEUE.md` into the repo; the
git tier executes them and never depends on it. **The file-contract is the API between the tiers** —
keep it stable and versioned and the SaaS bolts on without a rewrite.
