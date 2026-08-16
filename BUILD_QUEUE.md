# Build Queue

Reference spec: [`spec/`](spec/README.md)
Agent work tracking: `CLAIMS.md` (managed by coding agents)
Completed history: [`specflow/history/BUILD_QUEUE_DONE.md`](specflow/history/BUILD_QUEUE_DONE.md) — one-paragraph summaries of shipped batches.

## How this works

- This file lists only **un-done batches**, in full. Completed batches collapse to summaries in
  `specflow/history/BUILD_QUEUE_DONE.md` (git log + `specflow/history/CLAIMS_DONE.md` hold the implementation history).
- Dependencies are listed where they exist — the agent decides execution order.
- Agents claim and track completion in `CLAIMS.md`. **No Owner / Started / Status ever goes here.**
- See `specflow/procedures/claim-batch.md` before claiming.

---

## Un-done batches

> **Pick-order pointer — Milestone v0.1 is shipped** 🚀 (goal/DoD in `roadmap.md`). `v0.1.0` is
> tagged, published, and the public `curl|sh` install is verified end-to-end. All member
> batches are done: **CFG** (config + commit/push levers + safety), **BI** (brownfield `init` +
> `verify` + `_DONE` relocation), **SO** (spec-only mode), **G2** (release: GoReleaser → GitHub
> Releases + `curl|sh`; Homebrew deferred), **1** (add-agent), **2** (status), **5** (`--dry-run`).
> **`v0.1.1`** (patch) ships the user-facing content improvements since: **FH** (finish-batch step-6
> handoff rework) and **RF** (research-flow convention).
> **`v0.1.2`** (patch) ships **CH** (Claude-Code step-6 handoff hook: a `PostToolUse` backstop that
> blocks the loop after a `meta: complete batch-*` commit to force the handoff) plus the `upgrade`
> convergence that delivers newly-shipped non-managed adapter files to existing installs.
> **`v0.1.3`** (patch) ships **SL** (spec-only mode no longer names the queue/claim
> machinery it omits, plus the `verify` mode-consistency check and an `upgrade` repair path for
> existing installs) and **SZ** (the spec-file 600-line cap is now a stop-and-ask, with a
> `specflow:size-ok` waiver that re-asks every +200 lines).
> **`v0.1.4`** (patch) ships **4** (README rewrite: badges, file-map, and an agent-executable
> install section) and **PR** (ledger pruning: the `prune-ledgers` procedure + skill that keeps
> `CLAIMS.md` to its 5 newest completed entries, archiving the rest).
> **Pick next: Batch RD** — the only un-blocked entry; everything else is `[NOT READY]`. Unblock
> **Batch W** (needs the profile→dimension mapping in `open-questions.md`) or **Batch NB** (needs a
> design pass) before claiming either of those.
> **Post-v0.1 queue below:**
> **Batch RD** (release auto-publish) · **Batch W** (workflow config) · **Batch NB** (`--new-batch`) ·
> **Batch E** (enforcement — research-first) · **Batch P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch RD — Release auto-publish, and the user approves every release

**Goal.** Remove the manual publish step that has now broken two releases, and record that cutting a
release is the user's call, not the agent's.

**Why.** `.goreleaser.yaml` sets `release: draft: true`, so a `v*` tag builds the archives into a
*draft* the user must publish by hand. Twice (v0.1.3, v0.1.4) a release was instead created directly
in the GitHub UI, which produces a published release carrying **zero assets** while the real draft
sits unpublished beside it. `install.sh` resolves `releases/latest` and downloads
`specflow_<ver>_<os>_<arch>.tar.gz` from whatever it finds, so both times the public `curl … | sh`
install 404'd until it was noticed and repaired by hand. v0.1.1 has no release at all for the same
reason. The manual gate puts the release *copy* in front of the *binaries*; inverting that is the
fix, since a body can be edited after publish but a missing archive breaks installs immediately.

**Scope.**
- Flip `release.draft` to `false` so a pushed tag publishes the archives directly, and rewrite the
  comment above it to say why (and that the body is edited after the fact, not before).
- Record the counterpart rule: **the agent never cuts a release without the user's explicit
  approval each time.** Auto-publish removes the human checkpoint from the *pipeline*, so the
  checkpoint moves earlier, to the decision to tag at all. Per the user, this goes in `CLAIMS.md`.
- Note the auto-publish behavior in `spec/architecture.md` → artifact host, which currently
  describes the tag→GoReleaser→release flow without saying draft or published.

**Not in scope.** Changelog prettification (`release.header`, `changelog.groups`) — the generated
body is two clean `batch-*:` lines today. Backfilling a v0.1.1 release.

### Files this batch creates/edits
- `.goreleaser.yaml` · `CLAIMS.md` · `spec/architecture.md`.

### Verification
- `goreleaser check` passes.
- Confirm on the next tag push that the release appears published with 6 assets and no manual step.

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
- Survey the layers, cheapest → most authoritative: a read-only **batching/enforcement** validator
  (distinct from the installation `specflow verify` shipped in Batch BI — changed files map to an
  owned claim, no state leaked into `BUILD_QUEUE.md`, commit grammar), local git hooks
  (`commit-msg` / `pre-push` via `core.hooksPath`), CI running the validator, and GitHub branch
  protection.
- For each: what it binds, prevents-vs-detects, bypassability, and the carve-outs needed
  (`meta:` / `spec:` / doc-only) so it never fights legitimate work — **including the install
  bootstrap**: a fresh `specflow init` creates many specflow-owned files with no claim, so the check
  must exempt the specflow-owned scaffold (and `init` commits the install as its own `meta:` commit);
  otherwise running `init` then the check before committing false-positives.
- Output a short `spec/enforcement.md` design doc + a proposed **incremental** sub-batch sequence.
  Generalizes Upside's `docs/process/agent-discipline.md`.

### Then — sub-batches (only after Phase 1 is agreed with the user)
- the batching validator → opt-in `install-hooks` → optional host-repo CI template → branch-protection guidance.

### Files this batch creates/edits (Phase 1)
- `spec/enforcement.md` · queue updates for the agreed sub-batches.

---

## Batch P `[NOT READY]` — optional npm-wrapper front-end

**Superseded 2026-06-21.** Primary distribution is now a Go binary on **GitHub Releases** via Batch
G2 (GoReleaser) — not npm. This batch is reduced to the **optional npm wrapper**: an npm package that
fetches the prebuilt binary so `npx specflow` still works for the JS ecosystem (esbuild pattern).
Claim only if we decide to also serve `npx`. See `open-questions.md` → Distribution.
