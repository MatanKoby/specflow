# specflow

**Spec-driven design for your AI coding agents. Small, single-subject spec files, so each agent reads only what the task needs and stays on topic.**

Spec-driven design is how humans have built serious software for years: decide, write it down,
build, then look again when something needs to change. Often that spec just lived in a person's
head. That doesn't carry over to agents, though: holding a whole project's worth of context you
don't currently need is expensive. specflow prepares the spec ahead of time as small,
single-subject files in your repo, so an agent consumes only the piece it needs and can enter
any part of the project at any time. It comes in two layers: one keeps the spec current; the
other turns work into batches and runs a git-backed, async-friendly queue that multiple agents
(say Claude Code and Cursor at once) can claim from to work on different things in tandem.

### 1. Spec (the core)

- **Specs that update as you decide:** design lives in a handful of short files, one subject
  each. Every time you spec something or lock a decision, the relevant file updates, so the spec
  reflects your latest thinking instead of rotting.
- **Clarity for every agent:** one shared protocol (`AGENTS.md`) plus native integrations per
  agent (auto-triggering Claude Code skills, Cursor/Copilot rules, and so on), so they all read
  the same small, current picture.
- **Token savings for humans:** because each file covers one subject, an agent pulls in the one
  or two specs a task needs, not a giant document. Less context loaded, less repetition, fewer
  tokens per task.

### 2. Build loop (default)

- **Declare, claim, build:** put work in a queue as batches, claim each in git *before* you touch
  code, then wrap it up cleanly. Your repo always records what's done and what's next, so agents
  work different batches in tandem without colliding and a crashed laptop never loses the thread.

All plain markdown + git. No runtime, no service, no lock-in, just one binary. `specflow init`
installs both layers; add `--spec-only` if you want just the spec layer, without the build loop.

### Who it's for

Best when you're **starting a new project** or working in a **small repo**, solo or a small team.

The spec has to come from a human who knows the code; specflow doesn't backfill it for you (yet).
On a new project that's effortless and compounds as you go. On an existing repo it's a
**one-time investment**: write the spec once, then get the full token-saving agentic-coding
experience from there on.

## Install

specflow is a single binary: **no runtime (Node, Python, …) required**.

```bash
# Prebuilt binary, no Go needed (Linux & macOS):
curl -fsSL https://raw.githubusercontent.com/MatanKoby/specflow/main/install.sh | sh
```

This detects your OS/arch, downloads the matching binary from the latest
[GitHub Release](https://github.com/MatanKoby/specflow/releases), verifies its checksum, and
installs it to `/usr/local/bin` (falling back to `~/.local/bin`). Pin a version with
`SPECFLOW_VERSION=v0.1.0`, or change the target with `SPECFLOW_INSTALL_DIR=…`.

```bash
# Or, with Go installed:
go install github.com/MatanKoby/specflow/cmd/specflow@latest
```

Windows binaries are attached to each release; **Homebrew** (`brew install`) is coming.

## Quick start

```bash
# In the repo you want to set up:
specflow init                          # interactive, pick your agents
specflow init --agents=claude,cursor   # non-interactive
specflow init --all
```

`init` asks which agents will work in the repo and scaffolds the protocol files.

## What it installs

| Path | Role | Owner |
|---|---|---|
| `AGENTS.md` | The full protocol every agent reads first | specflow (overwritten on upgrade) |
| `specflow/procedures/*.md` | The three procedures (claim / spec-edit / finish) | specflow (overwritten on upgrade) |
| `specflow/config.json` | Config + state (versions, schema, region hashes) | specflow |
| `BUILD_QUEUE.md` / `specflow/history/BUILD_QUEUE_DONE.md` | Work declaration + completed history | you |
| `CLAIMS.md` / `specflow/history/CLAIMS_DONE.md` | Execution-state ledger | agents |
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

`AGENTS.md` is always written as the universal base, so any agent that reads it is covered even
without a dedicated stub.

## Upgrade

```bash
specflow upgrade
```

Refreshes only the managed files (`AGENTS.md` + `specflow/procedures/`) and bumps the version
stamp. Your queue, claims, and spec are never touched. The stamp's `schemaVersion` gates future
state-file migrations; none exist yet, and the file-contract is kept stable on purpose so they
stay rare.

## Setting it up via your agent

You don't have to run the CLI yourself. Point your agent at this repo and say:
*"Set up specflow in this project: install it (`go install
github.com/MatanKoby/specflow/cmd/specflow@latest`), run `specflow init`, and pick Claude Code +
Cursor."* Then ask it to read `AGENTS.md` and claim a batch.

## License

MIT
