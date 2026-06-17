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

> **Pick-order pointer.** Correctness-first: **Batch U** (non-destructive upgrade — `upgrade` is
> provisional/unsafe until this lands). Then claimable: **Batch 1** (add-agent) · **Batch 2**
> (status) · **Batch 3** (broaden tests) · **Batch 4** (README badges + GIF) · **Batch 5** (CLI
> paper-cuts). Blocked on design: **Batch W** (workflow config). Later: **Batch V/H/CI**
> (enforcement), **Batch P** (npm publish).

---

## Batch 1 — `specflow add-agent <name>`

**Goal.** `init` is one-shot; adding an agent later means manual copying. Add a command that copies
one agent's adapter into an already-initialized repo and records it in the stamp's `agents`.

### Deliverables
- `specflow add-agent claude|cursor|copilot|bob|antigravity` copies that adapter (skip-existing),
  updates `.spec-batch.json` `agents`, prints what was written.
- Errors cleanly if specflow isn't installed here, or the agent is unknown/already present.

### Files this batch creates/edits
- `bin/specflow.js` (new command + shared copy helpers) · `test/smoke.js` (coverage).

### Verification
- `npm test`; manual: init with one agent, `add-agent` a second, confirm files + stamp.

---

## Batch 2 — `specflow status`

**Goal.** A read-only summary of the install so a user/agent can orient instantly.

### Deliverables
- Prints: kit version (stamp vs installed), installed agents, workflow profile (once Batch W lands),
  `## In progress` claims, count of un-done batches, and a drift flag (managed file edited).

### Files this batch creates/edits
- `bin/specflow.js` · `test/smoke.js`.

### Verification
- `npm test`; manual: run in a fresh install and in one with an active claim.

---

## Batch 3 — Broaden the test suite

**Goal.** Lock behavior beyond the current smoke checks.

### Deliverables
- Content assertions: generated `AGENTS.md` contains its key sections (work-queue, claims, commit
  grammar table); procedures reference `specflow/procedures/…`.
- Interactive picker (piped stdin), `--all`, and copilot/bob/antigravity adapters covered.
- An `npm pack` manifest test asserting every `templates/**` file ships (guards dropped dotfiles).

### Files this batch creates/edits
- `test/smoke.js` (or split into `test/`).

### Verification
- `npm test` green locally and in CI on Node 18/20/22/24.

---

## Batch 4 — README badges + demo

**Goal.** Make the repo legible at a glance.

### Deliverables
- CI + license badges; once published, an npm-version badge. A short asciinema/GIF of `init`.
- Strengthen the "what each file is + the agent's loop over them" section (replaces the dropped
  demo-repo idea).

### Files this batch creates/edits
- `README.md` · `docs/` asset.

### Verification
- Render check; links resolve.

---

## Batch 5 — CLI paper-cuts: NO_COLOR + Node guard

**Goal.** Two cheap robustness fixes (see `open-questions.md` → CLI).

### Deliverables
- Honor `NO_COLOR` and non-TTY (`!process.stdout.isTTY`) — disable ANSI so piped/CI output is clean.
- Runtime Node-version guard: friendly "specflow needs Node 18+, you have X" instead of a crash.

### Files this batch creates/edits
- `bin/specflow.js` · `test/smoke.js`.

### Verification
- `npm test`; `node bin/specflow.js init | cat` shows no escape codes.

---

## Batch U — Non-destructive upgrade redesign

**Goal.** Enforce the hard invariant: `upgrade` must never remove or overwrite text authored by a
user or another agent, in any file. Replace wholesale file-overwrite with marker-delimited managed
regions + drift detection.

### Deliverables
- specflow's managed content in `AGENTS.md` (and any managed file) is wrapped in
  `<!-- specflow:start -->` … `<!-- specflow:end -->`; `upgrade` replaces only that region and
  preserves everything outside it verbatim.
- `upgrade` detects a hand-edited managed region (drift) and warns / backs up rather than clobbering.
- `init` writes the markers; an existing install without markers is migrated on first upgrade.
- Tests assert: a user paragraph added outside the markers survives an upgrade; edited-region drift
  is reported, not silently overwritten.

### Files this batch creates/edits
- `bin/specflow.js` · `templates/base/AGENTS.md` (markers) · `templates/base/specflow/procedures/*`
  · `test/smoke.js`.

### Verification
- `npm test`; manual: add a user paragraph to `AGENTS.md`, run `upgrade`, confirm it survives.

---

## Batch W `[NOT READY]` — Workflow config model

**Depends on:** `open-questions.md` → Workflow section resolved (profile names + default, dimension
defaults, fifth-lever question, enforcement coupling).

**Goal.** Implement `spec/workflow.md`: the five dimensions, the explicit-choice setup flow (no
default profile; `--profile` required non-interactively), the `workflow` stamp block, and
`config.md` rendering; procedures reference `config.md`.

### Files this batch creates/edits
- `bin/specflow.js` (setup flow + render) · `templates/base/specflow/procedures/*` (policy-dependent
  steps reference `config.md`) · `templates/base/specflow/config.md` (template) · `test/smoke.js`.

### Verification
- Init each profile into a temp repo; confirm stamp + `config.md` + procedures match.

---

## Batch V `[NOT READY]` — `specflow verify`

**Goal.** Read-only protocol validator: every changed file maps to an owned `## In progress` claim,
no state leaked into `BUILD_QUEUE.md`, commit grammar valid. Carve-outs for `meta:`/`spec:`/doc-only.

---

## Batch H `[NOT READY]` — `specflow install-hooks`

**Depends on:** Batch V. Opt-in `commit-msg` + `pre-push` hooks (via `core.hooksPath`) running `verify`.

---

## Batch CI `[NOT READY]` — host-repo CI template

**Depends on:** Batch V. An optional GitHub Action `init` can drop into the host repo to run `verify`.

---

## Batch P `[NOT READY]` — npm publish + automated release

**Goal.** Publish to npm (enables `npx specflow`); GitHub Action publishes on `v*` tag. Claim only
when the user decides to publish.
