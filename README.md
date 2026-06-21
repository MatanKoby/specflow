# specflow

**Spec-driven, batch, claim-before-work development for AI coding agents — drop it into any repo with one command.**

specflow installs a small, file-based protocol that makes agentic development legible and safe
to parallelize:

- **Spec first** — design lives in `spec/`, concern per file, edited as a unit.
- **Batches** — work is declared as discrete units in `BUILD_QUEUE.md`.
- **Claim before work** — every batch is claimed in git (`CLAIMS.md`) *before* code is written,
  so the record of "who's doing what / what's done" survives a crashed laptop and lets multiple
  agents — or people — share one branch without colliding.
- **Three procedures** carry the discipline: `claim-batch`, `spec-edit`, `finish-batch`.

It's plain markdown + git. Every agent (Claude Code, Cursor, Copilot, IBM Bob, Google
Antigravity, and any tool that reads `AGENTS.md`) honors the same protocol.

## Install

specflow is a single binary — **no runtime (Node, Python, …) required**.

```bash
# With Go installed (works today):
go install github.com/MatanKoby/specflow/cmd/specflow@latest
```

Prebuilt binaries with **no Go required** — a `curl … | sh` script and **Homebrew** — land with the
release pipeline (see [Roadmap](#roadmap)).

## Quick start

```bash
# In the repo you want to set up:
specflow init                          # interactive — pick your agents
specflow init --agents=claude,cursor   # non-interactive
specflow init --all
```

`init` asks which agents will work in the repo and scaffolds the protocol files.

## What it installs

| Path | Role | Owner |
|---|---|---|
| `AGENTS.md` | The full protocol every agent reads first | specflow (overwritten on upgrade) |
| `specflow/procedures/*.md` | The three procedures (claim / spec-edit / finish) | specflow (overwritten on upgrade) |
| `specflow/.spec-batch.json` | Version + schema stamp | specflow |
| `BUILD_QUEUE.md` / `BUILD_QUEUE_DONE.md` | Work declaration + completed history | you |
| `CLAIMS.md` / `CLAIMS_DONE.md` | Execution-state ledger | agents |
| `spec/README.md` | Spec skeleton | you |
| per-agent stubs (e.g. `CLAUDE.md`, `.cursor/rules/…`) | Point each agent at `AGENTS.md` | you |

The boundary is deliberate: **specflow owns the mechanism (`AGENTS.md` + `specflow/`), you own
the content and state.** That's what makes `upgrade` safe.

## Agents

`init` lets you pick any of:

| Agent | What it writes |
|---|---|
| **Claude Code** | `CLAUDE.md` + `.claude/skills/{claim-batch,spec-edit,finish-batch}` (auto-triggering) |
| **Cursor** | `.cursor/rules/specflow.mdc` |
| **GitHub Copilot** | `.github/copilot-instructions.md` |
| **IBM Bob** | `.bob/rules/specflow.md` (Bob also reads `AGENTS.md` natively) |
| **Google Antigravity** | `.agents/rules/specflow.md` (Antigravity reads `AGENTS.md` natively) |

`AGENTS.md` is always written as the universal base — any agent that reads it is covered even
without a dedicated stub.

## Upgrade

```bash
specflow upgrade
```

Refreshes only the managed files (`AGENTS.md` + `specflow/procedures/`) and bumps the version
stamp. Your queue, claims, and spec are never touched. The stamp's `schemaVersion` gates future
state-file migrations — none exist yet, and the file-contract is kept stable on purpose so they
stay rare.

## Setting it up via your agent

You don't have to run the CLI yourself — point your agent at this repo and say:
*"Set up specflow in this project — install it (`go install
github.com/MatanKoby/specflow/cmd/specflow@latest`), run `specflow init`, and pick Claude Code +
Cursor."* Then ask it to read `AGENTS.md` and claim a batch.

## Roadmap

- **Now:** the authoring loop (spec → queue → claim → build), cross-agent.
- **Next:** prebuilt binaries — `curl | sh` + Homebrew — published to GitHub Releases, so no Go is
  needed to install.
- **Later (opt-in):** `specflow verify` — an executable that enforces the invariants
  (claim-before-work, commit grammar, no-state-in-queue) at git-hook / CI chokepoints.
- **Later:** a hosted tier that authors/syncs spec + queue into the repo (the file-contract is
  the API; the git tier never depends on it).

## License

MIT
