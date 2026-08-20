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
> **`v0.1.5`** (patch) ships **CE** (per-batch context cost: the procedures name the cheap read, plus
> the `config.check` field) and **QV** (the `next` / `claim` / `finish` queue verbs). **RD** (a pushed
> tag publishes the release directly; the agent needs the user's approval to cut one) landed between
> v0.1.4 and v0.1.5 but is **repo-internal and ships nothing to users**, so it opened no version line:
> it touched `.goreleaser.yaml` plus this repo's own ledgers and spec. Don't open a new version line
> for a batch until it changes something a user installs.
> **`v0.1.6`** (patch) ships **AF** (the adapters — the skill stubs and the handoff hook — are
> managed as whole files, so a fix to one finally reaches an install that already has it, carried
> across by a one-time adoption on the next `upgrade`).
> **Current release: `v0.1.6`. No version line is open.**
> **Claimable now: Batch RN** (authored release notes) — repo-internal, ships nothing to users, so
> it opens no version line.
> **Nothing is claimable right now** — every batch below is `[NOT READY]`, so the next move is the
> user's: cut the tag, or promote one of them.
> **Post-v0.1 queue below:**
> **Batch W** (workflow config) · **Batch NB** (`--new-batch`) · **Batch E** (enforcement — research-first) ·
> **Batch P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch RN — Authored release notes

**Depends on:** none.

**Goal.** Make a pushed tag publish the release body an agent actually needs. Spec:
`spec/architecture.md` → *Distribution* → "The release body is authored, not generated".

Today the body is GoReleaser's filtered commit list. v0.1.6 is the case in point: the one thing its
reader needs is "run `specflow upgrade`, here is what it does to your tree, expect up to five
`.specflow-bak` files" — and none of that is derivable from commit subjects. Editing the body after
the fact needs a GitHub API token, which the agent cutting the release does not have (`git push`
authenticates over SSH; the REST API does not accept that), so in practice the body stays as
generated forever.

The fix keeps the "tag publishes" invariant and adds nothing to the agent's release-time burden that
isn't already in the release commit:

1. **`.github/release-notes/vX.Y.Z.md`**, written in the same commit as the version bump. Reviewed
   in the same diff, before the irreversible tag push.
2. **`.github/workflows/release.yml`** resolves the file for the pushed tag and passes
   `--release-notes` to GoReleaser when it exists. **A missing file must not fail the job** — it
   falls back to the generated changelog, because a release that ships no archives is far worse
   than one with a plain body. Warn in the job summary so the omission is visible.
3. **Record it in the release procedure** so the next agent writes the file as part of cutting a
   release rather than rediscovering this. `CLAIMS.md` → *Releases need the user's approval* is
   where the release convention already lives.
4. **Backfill `v0.1.6`'s notes file** from the draft already written this session, so the directory
   starts with a worked example in the specified shape. This does not touch the published v0.1.6
   release (deliberately — the user's call).

Repo-internal: no `templates/`, no Go source, no `install.sh`. It ships nothing to users and opens
no version line.

### Files this batch creates/edits
- `.github/workflows/release.yml` · `.github/release-notes/v0.1.6.md` (new) · `CLAIMS.md`

### Verification
- `act` isn't available here, so verify by reading: the step must resolve the tag name to a path,
  pass `--release-notes` only when the file exists, and otherwise run exactly the args it runs today.
- Confirm the fallback path is the *default* branch of the shell logic, not an error branch.

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
