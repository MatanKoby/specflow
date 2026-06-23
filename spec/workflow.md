# Workflow & autonomy policy

specflow spans a spectrum: a solo developer who lets the agent **commit and push freely**, and a
team raising a new repo that wants **accountability — PRs, review, no direct pushes** — while still
getting the spec-batch, token-saving, spec-guided benefits. The setup flow must make a user
**understand what they are authorizing the agent to do.**

## Work-admission gates — the agent's authoring authority

Distinct from the git levers below (which govern how *code lands*), two gates govern whether the
agent may *author design and work* at all. **Both are always on — they are the baseline protocol,
not a profile knob.** No profile, Autonomous included, relaxes them; profiles govern git/build
autonomy only.

1. **Approval → spec.** A design decision may not be persisted to `spec/` until the user has
   **explicitly approved** it. The agent *recommends and proposes* how something should work and
   surfaces the tradeoffs, then waits — it does not write a design it settled on its own. A vague
   "continue" authorizes finishing **already-approved, in-scope** work; it is **not** authorization
   to open new scope.
2. **Spec → batch.** A batch may not be created in `BUILD_QUEUE.md` unless the design it builds is
   **already in `spec/`**. The batch entry *references* the spec; it never carries design that lives
   nowhere else.

Together: **idea → (approval) → spec → batch → claim → build.**

### Who decides what

- **The user owns the design.** Broad strokes, wanted functionality, and *how a thing should
  operate* are the user's call. The agent recommends, lays out options/tradeoffs, and asks — then
  implements what the user approves.
- **The agent owns the code.** Once the design is settled, *how* it is implemented (internal
  structure, naming, algorithm, file layout) is the agent's call — unless the user specified
  otherwise.
- **A clash is a stop.** If new design collides with something already decided, the agent raises it
  and asks for direction rather than resolving it itself.

### The two phases

- **Spec phase — dialogue; the user decides.** The agent asks about anything unfilled and raises
  every question that arises; the user answers; they converge until the spec is complete.
- **Implementation phase — conditional autonomy.** When the spec is complete enough to build
  *without further questions*, the agent implements autonomously. The moment a real question
  surfaces mid-build, it **stops and asks** rather than guessing.

**The test for "is this mine to decide?"** — if a choice affects *what the thing does or how it
behaves* (anything the user would plausibly have a view on), it is design: propose and get approval.
If it only affects *how the code achieves an already-approved behavior*, it is the agent's. When
genuinely unsure which side a choice falls on, treat it as design and ask.

These gates live in the procedures (`spec-edit.md` → gate 1, `claim-batch.md` → gate 2). How
strictly they are *enforced* — written guidelines vs. an executable check — is the one piece still
open (see `open-questions.md` → *Speccing & approval discipline*); v1 is guidelines-only, consistent
with enforcement generally (see Enforcement below / Batch E).

**Brownfield repos** (existing code, no spec) adopt these gates incrementally rather than
spec-everything-up-front; the contradiction/change-existing-code stop is part of the gates. See
`adoption.md`.

## Model: five orthogonal dimensions (not presets)

Workflow is configured as independent levers, stored in `specflow/config.json` under `config`
(`commit` / `push` ship in v0.1; the rest land with Batch W), and rendered into a per-repo
**`specflow/config.md`** that both humans and agents read.
The **single, universal** procedures reference `config.md` for the policy-dependent steps
(push / branch / merge / commit cadence) — there are no per-mode procedure variants.

| Dimension | Options | Meaning |
|---|---|---|
| `trunkBranch` | any name (init detects current) | the shared base branch |
| `branchPerBatch` | `false` · `"batch-{n}-{slug}"` | work on trunk, or an isolated branch per batch |
| `push` | `agent` · `user` | agent commits **and pushes**, or agent commits only and the user pushes |
| `integration` | `direct` · `pr` · `manual` | land on trunk directly · open a PR on finish · leave the branch for the user to merge |
| `commitCadence` | `incremental` · `squash` | many `batch-N:` commits per batch, or one commit per batch |

These five cover the scenarios raised: "commit + push freely" (`push:agent, branchPerBatch:false,
integration:direct`), "agent commits, I push" (`push:user`), "a branch per batch with PRs"
(`branchPerBatch:"batch-{n}-{slug}", integration:pr`), and clean one-commit-per-batch history
(`commitCadence:squash`). Commit is always the agent's action; the levers vary **push, branch,
merge, and commit cadence**.

## Profiles — educational on-ramps

Profiles are **presets over the four dimensions**, not a separate model. Each is shown as a
plain-English "what the agent may do" sentence:

- **Autonomous** — agent commits and pushes to trunk itself. No review gate. (solo/personal)
- **Supervised** — agent commits but never pushes; the user pushes.
- **Reviewed** — agent works a branch per batch and opens a PR; never touches trunk. (team)

## Commit & push authority — the user decides in advance

Two **independent** levers the user sets **at init**, governing what the agent may do with code it
has written:

- **`commit`: `agent` | `user`** — may the agent create commits? When **`user`**, the agent does
  **not** commit: on reaching a sensible commit point it **alerts the user** and **supplies a short
  suggested commit message** (in the `spec:` / `meta:` / `batch-N:` grammar), and the user creates
  the commit.
- **`push`: `agent` | `user`** — may the agent push? (Only meaningful when `commit: agent`.) When
  **`user`**, the agent commits but never pushes; the user pushes on their own terms.

So **Autonomous** = `commit: agent, push: agent` (writes per batch, commits, pushes). A user who
wants more control sets `commit: user` and/or `push: user`. This mirrors the CLI's own rule that
`init` / `upgrade` never commit (see `architecture.md`).

These two levers are part of the broader workflow config (Batch W); **v0.1 ships just these two
choices at init** (stored in the stamp), with the full dimension/profile system deferred. The
procedures (`claim-batch.md` / `finish-batch.md`) honor the levers instead of always committing +
pushing.

## The `init` setup flow

**The user must choose — there is no pre-selected default to blind-accept.** `init` runs an
interactive setup flow that presents the profiles, each as a plain-English "what the agent may do"
sentence; the user picks one, then may drop into a **Customize** path that walks the five dimensions
individually from that starting point. It ends with a **plain-English confirmation** of the agent's
permissions before any file is written. Non-interactive use must pass `--profile` explicitly (there
is no implicit default). The chosen config → the stamp's `workflow` block + a freshly rendered
`config.md`. `upgrade` re-renders `config.md` from the stored config.

## Enforcement (relationship)

**v1 enforces strict profiles by agent guidelines only.** The procedures + `config.md` tell the
agent what it may do; nothing physically blocks a misbehaving agent. specflow is **honest about
this**: on a strict profile (Supervised / Reviewed), `init` prints one line noting the policy is
advisory-until-enforced and offers the branch-protection command a user can run today. The
Autonomous profile shows nothing (no noise for solo users).

Real enforcement — a `verify` validator, `pre-push`/`commit-msg` hooks, a CI check, and
branch-protection guidance — is deferred to **Batch E**, a research-first track that keeps
enforcement exactly as in Upside (honor-system) today and *designs how to add it incrementally
before building it*. The model is **start by telling the agent; add strict enforcement once
designed** — never mislead a user that a chosen policy is locked down when it is currently a written
promise.

## Still open

The exact **profile → dimension mapping** (what each profile sets each lever to — the starting
point a Customize walk edits) is the remaining build-time detail, tracked in `open-questions.md`.
**Decided:** profiles are Autonomous / Supervised / Reviewed; `init` requires an explicit choice (no
default); `commitCadence` is the fifth lever; v1 enforces strict profiles by agent guidelines only
(see Enforcement above).
