# Workflow & autonomy policy

specflow spans a spectrum: a solo developer who lets the agent **commit and push freely**, and a
team raising a new repo that wants **accountability — PRs, review, no direct pushes** — while still
getting the spec-batch, token-saving, spec-guided benefits. The setup flow must make a user
**understand what they are authorizing the agent to do.**

## Model: four orthogonal dimensions (not presets)

Workflow is configured as independent levers, stored in `specflow/.spec-batch.json` under
`workflow`, and rendered into a per-repo **`specflow/config.md`** that both humans and agents read.
The **single, universal** procedures reference `config.md` for the policy-dependent steps
(push / branch / merge) — there are no per-mode procedure variants.

| Dimension | Options | Meaning |
|---|---|---|
| `trunkBranch` | any name (init detects current) | the shared base branch |
| `branchPerBatch` | `false` · `"batch-{n}-{slug}"` | work on trunk, or an isolated branch per batch |
| `push` | `agent` · `user` | agent commits **and pushes**, or agent commits only and the user pushes |
| `integration` | `direct` · `pr` · `manual` | land on trunk directly · open a PR on finish · leave the branch for the user to merge |

These four cover the scenarios raised: "commit + push freely" (`push:agent, branchPerBatch:false,
integration:direct`), "agent commits, I push" (`push:user`), "a branch per batch with PRs"
(`branchPerBatch:"batch-{n}-{slug}", integration:pr`). Commit is always the agent's action; the
levers vary **push, branch, merge**.

## Profiles — educational on-ramps

Profiles are **presets over the four dimensions**, not a separate model. Each is shown as a
plain-English "what the agent may do" sentence:

- **Autonomous** — agent commits and pushes to trunk itself. No review gate. (solo/personal)
- **Supervised** — agent commits but never pushes; the user pushes.
- **Reviewed** — agent works a branch per batch and opens a PR; never touches trunk. (team)

## The `init` setup flow

**Start with a default profile, then customize.** `init` pre-selects a profile, displays exactly
what it authorizes, and lets the user accept it or drop into a **Customize** path that walks the
four dimensions individually. It ends with a **plain-English confirmation** of the agent's
permissions before any file is written. The chosen config → the stamp's `workflow` block + a
freshly rendered `config.md`. `upgrade` re-renders `config.md` from the stored config.

## Enforcement (relationship)

Strict profiles (Supervised / Reviewed) are only *real* if enforced — branch protection on trunk
plus the hooks/CI layer (`specflow verify`, `install-hooks`, host CI template). Otherwise "the agent
won't push to trunk" is honor-system. How tightly the setup flow couples to enforcement setup is
**open** — see `open-questions.md` (enforcement coupling).

## Still open

The profile names, **which profile is the default**, the exact defaults per dimension, whether a
fifth lever (commit cadence) exists, and the enforcement-coupling question are unresolved — tracked
in `open-questions.md`.
