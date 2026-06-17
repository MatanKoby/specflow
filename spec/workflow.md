# Workflow & autonomy policy

specflow spans a spectrum: a solo developer who lets the agent **commit and push freely**, and a
team raising a new repo that wants **accountability — PRs, review, no direct pushes** — while still
getting the spec-batch, token-saving, spec-guided benefits. The setup flow must make a user
**understand what they are authorizing the agent to do.**

## Model: five orthogonal dimensions (not presets)

Workflow is configured as independent levers, stored in `specflow/.spec-batch.json` under
`workflow`, and rendered into a per-repo **`specflow/config.md`** that both humans and agents read.
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

Real enforcement — branch-protection guidance, a `pre-push` hook, and a CI `verify` check — is
deferred to later research/development (Batches V / H / CI). The model is **start by telling the
agent; add strict enforcement once designed** — never mislead a user that a chosen policy is locked
down when it is currently a written promise.

## Still open

The exact **profile → dimension mapping** (what each profile sets each lever to — the starting
point a Customize walk edits) is the remaining build-time detail, tracked in `open-questions.md`.
**Decided:** profiles are Autonomous / Supervised / Reviewed; `init` requires an explicit choice (no
default); `commitCadence` is the fifth lever; v1 enforces strict profiles by agent guidelines only
(see Enforcement above).
