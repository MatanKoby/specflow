# Adoption — bringing specflow onto existing code

specflow's gates assume a **greenfield** flow (idea → spec → batch → build). Most real repos are
**brownfield**: code already exists with no spec. This file defines how specflow adopts onto
existing, unspecced code without violating the work-admission gates (`workflow.md`).

## Model: incremental by default, assisted backfill optional

specflow does **not** require reverse-documenting a whole codebase before use. `init` scaffolds an
empty `spec/` as usual; the spec grows to cover only what you actually touch. Even installation is
brownfield-aware: `init` **injects** specflow's marker-wrapped region into existing agent files (with
per-file consent) instead of skipping them, and never commits — see `architecture.md` → init /
upgrade.

- **Just-in-time spec (default).** Before working an area, spec *that area* first (the gate in
  `spec-edit.md` already requires this), then batch + build. Existing code stays as-is until a batch
  reaches it.
- **Assisted backfill (optional, explicit).** A user may ask the agent to draft spec for code that
  already exists. Opt-in and approval-gated — never automatic.

### Spec the intent, not the mechanics
When the agent drafts spec from existing code, it can only read what the code **does**. But what the
code **should** do, and **why** — the design intent — lives only with the user, and it is the **more
valuable thing to capture**: *"what is this file / class / script meant to do?"* is worth far more in
spec than *"this block takes A and returns B."* So backfill **leads with intent questions to the
user** and treats mechanical behavior description as a fallback, not the goal. The agent presents any
drafted spec as *"here is what I read — ratify or correct it,"* never as settled design. This keeps
backfill inside the gates: the agent proposes; the user supplies intent and ratifies.

## Agent behavior on existing code

These extend the work-admission gates (`workflow.md` → Work-admission gates):

- **Stop on contradiction or change.** When a new or approved spec touches or integrates with
  existing code, and the agent notices a **contradiction** with what's there, or finds it would need
  to **change existing code** (beyond purely adding alongside it) — it **stops and asks the user**
  before proceeding. Purely *adding* new code needs no extra ask **only when that addition is itself
  already specced and approved**; contradicting or modifying existing code always stops for a
  question.
- **Ask before crawling — two steps.** The agent does **not** crawl unspecced code on its own. When
  context from existing code would help, it asks the user **first**: (1) *"is there something
  relevant you can point me to?"* — a chance to hand over a shortcut; (2) if the user has nothing to
  offer, the agent may then **ask permission to scout** — scan the unspecced code that looks relevant
  to check relevancy or look ahead. It crawls only on a yes. This keeps it token-frugal and never
  spends the user's tokens on exploration they didn't want.
- **Offer to spec what exists — sparingly.** The agent may *suggest* backfilling spec for relevant
  existing code, but only when there's a meaningful amount of it (if the area is already well-specced
  there's nothing to offer), and only **occasionally** — it must not nag. Suggest-only; it acts on a
  yes.
- **Pull from existing docs.** The agent may ask whether spec or documentation already lives
  elsewhere (READMEs, design docs, wikis). If so, it asks which files/features those describe, then
  drafts spec for them (subject to the ratify-not-assume rule above).

## Roadmap
Remote spec sources — known integrations (e.g. Confluence) that import spec/documentation from
external systems — are a later direction; see `roadmap.md`.
