# Build Queue

Reference spec: [`spec/`](spec/README.md)
Agent work tracking: `CLAIMS.md` (managed by coding agents)
Completed history: [`BUILD_QUEUE_DONE.md`](BUILD_QUEUE_DONE.md) — one-paragraph summaries of shipped batches.

## How this works

- This file lists only **un-done batches**, in full. Completed batches collapse to summaries in
  `BUILD_QUEUE_DONE.md` (git log + `CLAIMS_DONE.md` hold the implementation history).
- Dependencies are listed where they exist — the agent decides execution order.
- Agents claim and track completion in `CLAIMS.md`. **No Owner / Started / Status ever goes here.**
- See `specflow/procedures/claim-batch.md` before claiming.

---

## Un-done batches

> **Pick-order pointer.** **Now: Batch G2** (Go release + install pipeline — GoReleaser → GitHub
> Releases + `curl|sh` + Homebrew). The CLI is Go (Batch G1, done), so every batch below targets the
> **Go** CLI (`cmd/specflow` + `internal/kit`). After G2: **Batch 1** (add-agent) · **Batch 2**
> (status) · **Batch 3** (broaden tests) · **Batch 4** (badges + file-map) · **Batch 5** (`--dry-run`).
> Blocked on design: **Batch W** (workflow config) · **Batch NB** (`--new-batch` quick flow). Later:
> **Batch E** (enforcement — research-first), **Batch P** (optional npm-wrapper front-end).

---

## Batch G2 — Go release + install pipeline

**Goal.** Ship the binary with zero runtime: GitHub Releases as the artifact host, plus the v1
install front-ends (curl|sh + Homebrew). See `architecture.md` → Distribution. Depends on G1.

### Deliverables
- **GoReleaser** config: cross-compile matrix (macOS arm64/x64, Linux x64/arm64, Windows x64),
  archives + SHA-256 checksums.
- **GitHub Action** on a `v*` tag → runs GoReleaser → publishes the GitHub Release with all artifacts.
- **`curl … | sh` installer** that detects OS/arch and fetches the matching binary from the latest
  release.
- **Homebrew tap** formula (GoReleaser-generated) → `brew install`.
- README install section rewritten to the binary flow.

### Deferred (post-v1 — see `open-questions.md` → Distribution)
- npm wrapper (`npx specflow` via prebuilt binary, esbuild-style) · Scoop/Winget (Windows).

### Files this batch creates/edits
- `.goreleaser.yaml` · `.github/workflows/release.yml` · `install.sh` · Homebrew tap (separate repo
  or `homebrew/` dir) · `README.md`.

### Verification
- A test tag produces a draft release with every platform archive + a checksums file.
- `curl … | sh` installs a working binary on Linux; `brew install` from the tap works on macOS.

---

## Batch 1 — `specflow add-agent <name>`

**Goal.** `init` is one-shot; adding an agent later means manual copying. Add a command that copies
one agent's adapter into an already-initialized repo and records it in the stamp's `agents`.

### Deliverables
- `specflow add-agent claude|cursor|copilot|bob|antigravity` copies that adapter (skip-existing),
  updates `.spec-batch.json` `agents`, prints what was written.
- Errors cleanly if specflow isn't installed here, or the agent is unknown/already present.

### Files this batch creates/edits
- `cmd/specflow/` + `internal/kit/` (new command + shared helpers) · `cmd/specflow/main_test.go` (coverage).

### Verification
- `go test ./...`; manual: init with one agent, `add-agent` a second, confirm files + stamp.

---

## Batch 2 — `specflow status`

**Goal.** A read-only summary of the install so a user/agent can orient instantly.

### Deliverables
- Prints: kit version (stamp vs installed), installed agents, workflow profile (once Batch W lands),
  `## In progress` claims, count of un-done batches, and a drift flag (managed file edited).

### Files this batch creates/edits
- `cmd/specflow/` · `internal/kit/` · `cmd/specflow/main_test.go`.

### Verification
- `go test ./...`; manual: run in a fresh install and in one with an active claim.

---

## Batch 3 — Broaden the test suite

**Goal.** Lock behavior beyond the current smoke checks.

### Deliverables
- Content assertions: generated `AGENTS.md` contains its key sections (work-queue, claims, commit
  grammar table); procedures reference `specflow/procedures/…`.
- Interactive picker (piped stdin), `--all`, and copilot/bob/antigravity adapters covered.
- An embed-manifest test asserting every `templates/**` file is embedded in the binary (guards the
  `go:embed all:` dotfile footgun).

### Files this batch creates/edits
- `cmd/specflow/main_test.go` (or split into `internal/kit/` tests).

### Verification
- `go test ./...` green locally and in CI (ubuntu + macos).

---

## Batch 4 — README badges + file-map

**Goal.** Make the repo legible at a glance and the file-contract obvious to a newcomer.

### Deliverables
- CI + license badges (a release/version badge once published).
- A **pretty, simple file-map** in the README: each file specflow creates, what it does, and the
  flow the agent follows over them (spec → queue → claim → build). Directed by the user.
- **Deferred:** an animated demo (GIF/asciinema) — design the visual first before producing it.

### Files this batch creates/edits
- `README.md`.

### Verification
- Render check; links resolve; a new reader can follow the file-map without prior context.

---

## Batch 5 — `--dry-run` (preview)

**Goal.** A preview flag for `init` and `upgrade` that writes nothing and prints exactly what
*would* be created / overwritten / skipped.

### Deliverables
- `specflow init --dry-run` and `specflow upgrade --dry-run` list the planned file operations and
  exit without touching disk.

### Files this batch creates/edits
- `cmd/specflow/` · `internal/kit/` · `cmd/specflow/main_test.go`.

### Verification
- `go test ./...`; manual: `--dry-run` in a fresh dir creates no files.

---

## Batch W `[NOT READY]` — Workflow config model

**Depends on:** the profile→dimension mapping in `open-questions.md` → Workflow (the only remaining
build-time detail; the rest of the workflow design — 5 dimensions, no-default explicit-choice flow,
guidelines-only enforcement — is settled).

**Goal.** Implement `spec/workflow.md`: the five dimensions, the explicit-choice setup flow (no
default profile; `--profile` required non-interactively), the `workflow` stamp block, and
`config.md` rendering; procedures reference `config.md`.

### Files this batch creates/edits
- `cmd/specflow/` + `internal/kit/` (setup flow + render) · `templates/base/specflow/procedures/*`
  (policy-dependent steps reference `config.md`) · `templates/base/specflow/config.md` (template) ·
  `cmd/specflow/main_test.go`.

### Verification
- Init each profile into a temp repo; confirm stamp + `config.md` + procedures match.

---

## Batch NB `[NOT READY]` — `--new-batch` quick flow

**Goal.** A "now-to-now" command for when the user wants something specced and queued immediately:
it **initiates a short planning phase**, writes the result into `spec/`, then appends it to
`BUILD_QUEUE.md` as a batch (optionally handing it to the agent to claim + execute).

**Why `[NOT READY]`:** the flow needs a small design pass first — what the planning phase asks, how
much it writes to `spec/` vs the batch, and the hand-off to execution. Design, then build.

**Related (open):** the clarify-and-approve discipline in `spec/open-questions.md` → *Speccing &
approval discipline*. NB's planning phase is the natural home for the explicit clarify+approve gate
(ask the questions, end on a user OK before anything is written or claimed), but the gate principle
is broader than NB — it also bears on `claim-batch` for batches not created through NB.

### Files this batch creates/edits
- `cmd/specflow/` + `internal/kit/` · a `spec/` write + `BUILD_QUEUE.md` append · `cmd/specflow/main_test.go`.

---

## Batch E `[NOT READY]` — Enforcement (research-first)

**For now, enforcement is exactly as in Upside: honor-system / written guidelines** — the procedures
+ `AGENTS.md` tell the agent what to do; nothing executable checks it. This batch does **not** jump
to implementation. It **starts with research + discussion** of how to add enforcement incrementally,
then drafts the sub-batches.

### Phase 1 — research & discuss (the deliverable)
- Survey the layers, cheapest → most authoritative: a read-only validator (`specflow verify` —
  changed files map to an owned claim, no state leaked into `BUILD_QUEUE.md`, commit grammar), local
  git hooks (`commit-msg` / `pre-push` via `core.hooksPath`), CI running the validator, and GitHub
  branch protection.
- For each: what it binds, prevents-vs-detects, bypassability, and the carve-outs needed
  (`meta:` / `spec:` / doc-only) so it never fights legitimate work.
- Output a short `spec/enforcement.md` design doc + a proposed **incremental** sub-batch sequence.
  Generalizes Upside's `docs/process/agent-discipline.md`.

### Then — sub-batches (only after Phase 1 is agreed with the user)
- `verify` validator → opt-in `install-hooks` → optional host-repo CI template → branch-protection guidance.

### Files this batch creates/edits (Phase 1)
- `spec/enforcement.md` · queue updates for the agreed sub-batches.

---

## Batch P `[NOT READY]` — optional npm-wrapper front-end

**Superseded 2026-06-21.** Primary distribution is now a Go binary on **GitHub Releases** via Batch
G2 (GoReleaser) — not npm. This batch is reduced to the **optional npm wrapper**: an npm package that
fetches the prebuilt binary so `npx specflow` still works for the JS ecosystem (esbuild pattern).
Claim only if we decide to also serve `npx`. See `open-questions.md` → Distribution.
