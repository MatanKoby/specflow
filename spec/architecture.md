# Architecture

## What specflow is

A single statically-compiled Go binary (`cmd/specflow`) plus a `templates/` tree, with no
third-party dependencies. The CLI scaffolds the protocol files into a host repo and refreshes them
over time. There is no runtime library the host imports — specflow's output is **plain markdown +
git**, which is what makes it cross-agent and language-neutral.

```
specflow/                   (this repo — the tool)
  cmd/specflow/             the CLI entry: init, upgrade, --version  (+ planned: add-agent, status, --new-batch, verify)
  internal/kit/             scaffold (init) + non-destructive refresh (upgrade): markers, region hashing, the stamp
  templates.go              embeds templates/ into the binary (//go:embed all:templates)
  templates/
    base/                   files every install gets (AGENTS.md, queue/claims skeletons, spec/, specflow/)
    agents/<agent>/         per-agent adapter stubs, copied when that agent is selected
  cmd/specflow/main_test.go builds + drives the binary against temp repos, asserts behavior
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

## init / upgrade

- **`init`** — interactive (or `--agents=…`/`--all`): pick agents, scaffold base + selected
  adapters, fill the stamp. Skips any file that already exists (never clobbers). Guards re-init.
- **`upgrade`** — refreshes specflow's managed mechanism to the installed kit version and bumps the
  stamp. **Hard invariant: `upgrade` never removes or overwrites text authored by a user or another
  agent, in any file.** Each managed file wraps its generated content in marker-delimited regions
  (`<!-- specflow:start … -->` … `<!-- specflow:end -->`); the start marker carries a "do not edit
  inside" note (an HTML comment — invisible in rendered markdown). Markers are matched by their
  `specflow:start` / `specflow:end` **token**, not exact text, so the note can evolve without
  breaking parsing or forcing a migration (a clean `upgrade` canonicalizes a file's markers to the
  template's current wording). `init` records a SHA-256 of each region in the stamp's `managed` map.
  On `upgrade`:
  - **Clean region** (on-disk hash matches the stored baseline) → only the content *between* the
    markers is replaced; everything outside is preserved verbatim, and the baseline is re-recorded.
  - **Drifted region** (hash differs — someone edited inside the markers) → left **untouched**; the
    fresh version is written to a `<file>.specflow-new` sidecar and reported, so nothing is clobbered.
  - **Pre-marker file** (an install predating markers) → migrated: backed up to `<file>.specflow-bak`,
    then rewritten with markers.

## Versioning & the stamp

One file: `specflow/.spec-batch.json` carrying `kitVersion` + `schemaVersion`, plus a `managed` map
(per managed file → SHA-256 of its region) that powers the drift detection above. Procedure prose is
*instructions, not data*, so a refresh replaces specflow's managed region (never user/agent text —
see the `upgrade` invariant above) — no migration. `schemaVersion` gates the
rare case where the **format** of the state files changes; only then is a migration needed. No
migration runner is built until a format actually breaks (the file-contract is kept stable on
purpose to keep that rare).

## Distribution

specflow ships as a **single, statically-compiled Go binary** — **no runtime (Node, Python, …)
required** on the user's machine. This is what makes "install into any repo, any language, no
friction" literally true: the binary scaffolds and upgrades the markdown + git protocol, and the
*output* is already language-neutral. (The CLI was a zero-dep Node script through v0.x; it is being
ported to Go — see `BUILD_QUEUE.md`. Decided 2026-06-21.)

- **Artifact host: GitHub Releases.** A `v*` tag triggers **GoReleaser** in a GitHub Action that
  cross-compiles per OS/arch (macOS arm64/x64, Linux x64/arm64, Windows x64), attaches archives +
  checksums to the release, and can update a Homebrew tap / Scoop manifest in the same run. GitHub
  Releases *is* the registry — no npm/PyPI account needed to host the binary.
- **Install front-ends.** v1 ships two, both resolving to the same release binary: a `curl … | sh`
  script that detects OS/arch, and **Homebrew** (`brew install`) for macOS/Linux. Deferred post-v1
  (see `open-questions.md` → Distribution): an **npm wrapper** so `npx specflow` still works for the
  JS ecosystem (esbuild pattern), and **Scoop/Winget** for Windows. `go install …@latest` works as a
  bonus for users who have Go.
- **Templates travel inside the binary** (Go `embed`), so there is no separate `files` allowlist to
  curate and the repo self-hosting the protocol at its own root never leaks into the shipped artifact.
- **Caveat:** a future Claude-plugin packaging must target `templates/agents/claude/.claude/`, never
  a self-hosted `.claude/` at the repo root.

## SaaS frontier (later)

The git tier above is the core and stands alone offline. A hosted tier (`spec_for_ai-as-a-service`)
would be an upstream **producer** that authors/syncs `spec/` + `BUILD_QUEUE.md` into the repo; the
git tier executes them and never depends on it. **The file-contract is the API between the tiers** —
keep it stable and versioned and the SaaS bolts on without a rewrite.
