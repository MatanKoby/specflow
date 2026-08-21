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

> **Pick-order pointer.** Current release: **`v0.1.8`**; **`v0.1.9` is open** and ships **RC** ·
> **MC** · **FS** · **ED**. Which batch shipped in which release lives in `spec/roadmap.md` →
> *Release lines*, and the milestone goals live there too, not here. This file holds un-done work only.
>
> **Claimable:** **RC** (drift reconciliation) · **MC** (`migrate-claims`) · **FS** (stub contract) ·
> **ED** (em dash sweep). Pick order RC, MC, FS, and **ED last** so it sweeps the prose the others
> add. MC and FS share `internal/kit/queue.go`; FS and ED share `templates/base/**`.
> **Not ready:** **NX** (`next` file spread) · **W** (workflow config) · **NB** (`--new-batch`) ·
> **E** (enforcement, research-first) · **P** (npm-wrapper front-end) · Homebrew tap.

---

## Batch RC - drift is a state you can leave

**Goal.** Two defects in one loop: reconciling a drifted managed file is destructive for
marker-delimited files, and it never actually clears the drift.

**(a) The sidecar is the wrong bytes for a region file.** `upDrift` writes the rendered template
*whole* (`internal/kit/kit.go:729`), which for an adapter is right (every byte is specflow's) and for
a marker-delimited file is a footgun: the warning tells the user to reconcile, the obvious
reconciliation is `mv`, and `mv` throws away everything outside the region. A downstream install hit
exactly this: 27 managed lines in `CLAUDE.md` and 73 lines of project guidance below the markers, with
one warning string (`cmd/specflow/main.go:516`) covering both cases and the correct action opposite in
each. Fix: the sidecar carries the *on-disk file with the fresh region spliced in*
(`before + markers + fresh region + after`), so `mv` is correct for both tiers and the warning can say
`mv` outright. Drift in a marker-delimited file is by definition drift inside the markers, which is
what the user then diffs.

**(b) Drift is terminal.** The adapter path adopts a file already identical to the current template
(`kit.go:853`, check 2). The region path has no such check, and `Upgrade` carries the old baseline
forward for a drifted file (`kit.go:976`). So after following the printed advice the region matches
the *new* template while the baseline is still the *old* hash: the next `upgrade` re-drifts it, writes
the sidecar again, and `verify` warns forever. The only exit today is restoring bytes that hash to the
old baseline, which means discarding the edit. Two parts: add adopt-on-identical to `decideUpgrade`
(mirrors the adapter check, self-heals every reconciled file), and add **`specflow adopt <file>...`**
(`--all` for the whole drifted set) to re-record a baseline over a deliberate local edit. `adopt`
changes no bytes; it says "I have reconciled this", which is what makes the warning list mean
something again.

**Spec.** `architecture.md` → *init / upgrade* documents the drift contract but not what the sidecar
contains, nor how drift ends. Both rules land there.

### Files this batch creates/edits
- `internal/kit/kit.go` · `cmd/specflow/main.go` · `cmd/specflow/main_test.go` ·
  `spec/architecture.md` · `README.md`.

---

## Batch MC - migrate-claims, so 0.1.8's ledger shape reaches old entries

**Goal.** The stub + pointer shape applies only to entries written *after* the upgrade, so an install
arriving at 0.1.8 still carries every legacy essay: the exact weight Batch LW exists to remove, in the
file that is re-read on every claim, finish, and prune. Add **`specflow migrate-claims [--dry-run]`**:
rewrite legacy `### Batch` entries in `CLAIMS.md` and `specflow/history/CLAIMS_DONE.md` to metadata
plus a stub within `StubMaxLines` plus the pointer, and **relocate** the displaced prose into
`specflow/history/BUILD_QUEUE_DONE.md` under that batch's heading. It never deletes prose: where a
section for the batch already exists, the relocated text is appended under a divider, never
overwritten. Writes nothing anywhere if either ledger fails to parse, same contract as `finish`.

**The hazard, and the reason this is a verb rather than a snippet in a procedure.** A hand-rolled
retrofit downstream lost a `- Progress note (2026-08-20, ...)` line and its 15 wrapped continuation
lines by reading the continuation as body prose. specflow's own archiver kept a multi-line
`- Overlap note:` intact in the same run, because it copies entries verbatim. So the parsing subtlety
is already solved here and nowhere else: metadata lines and every continuation line survive, and the
stub is cut from the body only. Both halves get a test.

**Note:** shares `internal/kit/queue.go` and `cmd/specflow/main.go` with Batch FS, so the two do not
run in parallel; either order is fine.

### Files this batch creates/edits
- `internal/kit/queue.go` · `cmd/specflow/main.go` · `cmd/specflow/main_test.go` ·
  `templates/base/specflow/procedures/prune-ledgers.md` · `README.md`.

---

## Batch FS - the stub contract says what the code already does

**Goal.** Three loose ends left by Batch LW, all cheap, all reported from a real install.

- **The cap counts prose only.** `stubLines` already skips blank lines and the pointer
  (`internal/kit/queue.go:815`), which is why a 10-line stub file is accepted against an 8-line cap.
  Correct behavior, undocumented: `finish-batch.md` step 3 and `templates/base/CLAIMS.md` state the
  cap without saying what counts, so an agent budgets against the wrong number. Say it where the cap
  is stated.
- **The pointer is unvalidated free text.** It is matched only to exclude it from the count
  (`queue.go:64`), while `finish` writes the done-file heading itself (`queue.go:622`) and therefore
  knows the target. Emit the pointer when the stub omits it, and refuse one that names a different
  batch.
- **The archive's header still promises the old shape.** `templates/base/BUILD_QUEUE.md:5` and
  `prune-ledgers.md:60` describe `BUILD_QUEUE_DONE.md` as "one-paragraph summaries", which is what it
  was before LW made it the home of the full narrative.

**Note:** shares `internal/kit/queue.go` with Batch MC and `templates/base/**` with Batch ED, so it
runs alone against either.

### Files this batch creates/edits
- `internal/kit/queue.go` · `cmd/specflow/main.go` · `cmd/specflow/main_test.go` ·
  `templates/base/BUILD_QUEUE.md` · `templates/base/CLAIMS.md` ·
  `templates/base/specflow/procedures/finish-batch.md` ·
  `templates/base/specflow/procedures/prune-ledgers.md`.

---

## Batch ED - one mechanical pass, no rewording

**Goal.** Everything specflow ships carries em dashes: 87 across the managed set, `AGENTS.md` alone
holding 31. They land in every install, where they collide with a repo rule that forbids them, and a
downstream repo that sweeps them locally is punished twice: the sweep is drift, so `upgrade` stops
refreshing those files (Batch RC is the other half of that story) and the next clean upgrade puts the
dashes back. Fixing it upstream is the only version that holds. Replace every em dash and en dash in
prose with a plain hyphen, a comma, a colon, or a sentence break, choosing per sentence but
**rewording nothing**, across `templates/**` and `specflow/**` plus this repo's own managed copies
(`AGENTS.md`, `.claude/skills/**`, the `CLAUDE.md` region) so a self-hosted `upgrade` agrees.

**Out of scope on purpose:** `spec/**`, `README.md`, and the ledgers. They are this repo's own prose,
not shipped content, and mixing them in makes the diff unreviewable. Sweep them later if wanted.

**Watch item:** the marker-parsing separator list (`internal/kit/queue.go:190`) accepts ` - ` as well
as ` — `, so heading separators may be swept; a dash that is *data* rather than prose must not be.
Verification is a grep for zero em/en dashes under the swept paths, plus `go test ./...`.

**Run last** of the four, so it also sweeps whatever prose RC, MC, and FS add.

### Files this batch creates/edits
- `templates/**` · `specflow/procedures/*.md` · `AGENTS.md` · `.claude/skills/*/SKILL.md` ·
  `CLAUDE.md` (managed region only).

---

## Batch NX `[NOT READY]` — `specflow next` prints each batch's file spread

**Why `[NOT READY]`:** raised as optional; the decision is open in `spec/open-questions.md` →
*CLI / upgrade behavior*. Promote once the user calls it.

**Goal.** `next` already prints each batch's declared file list verbatim, which goes long and reads as
a wall. Replace it with a **spread**: the file count plus the distinct top-level paths, so an over-wide
batch (`spec/architecture.md` → *Batch size*) is visible at a glance **before** it is claimed. Read-only,
no new state; `--json` keeps the full list. (The companion from the same report — warning when
`CLAIMS.md` `## Completed` holds more than five entries — shipped in Batch LW, so NX no longer
carries it.)

### Files this batch creates/edits
- `cmd/specflow/main.go` (`cmdNext`) · `internal/kit/queue.go` · `cmd/specflow/main_test.go`.

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
