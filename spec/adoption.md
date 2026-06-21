# Adoption — bringing specflow onto existing code

specflow's gates assume a **greenfield** flow (idea → spec → batch → build). Most real repos are
**brownfield**: code already exists with no spec. This file defines how specflow adopts onto
existing, unspecced code without violating the work-admission gates (`workflow.md`).

## Model: incremental by default, assisted backfill optional

specflow does **not** require reverse-documenting a whole codebase before use. `init` scaffolds an
empty `spec/` as usual; the spec grows to cover only what you actually touch.

- **Just-in-time spec (default).** Before working an area, spec *that area* first (the gate in
  `spec-edit.md` already requires this), then batch + build. Existing code stays as-is until a batch
  reaches it.
- **Assisted backfill (optional, explicit).** A user may ask the agent to draft spec for code that
  already exists. Opt-in and approval-gated — never automatic.

### Reverse-documentation is not design intent
When the agent drafts spec from existing code, it can only describe what the code **does** (read
from the source). What the code **should** do, and **why** — the design intent — lives only with the
user. The agent presents drafted spec as *"here is what I read — ratify or correct it,"* never as
settled design. This keeps backfill inside the gates: the agent proposes observed behavior; the user
turns it into intent.

## Agent behavior on existing code

These extend the work-admission gates (`workflow.md` → Work-admission gates):

- **Stop on contradiction or change.** When a new or approved spec touches or integrates with
  existing code, and the agent notices a **contradiction** with what's there, or finds it would need
  to **change existing code** (beyond purely adding alongside it) — it **stops and asks the user**
  before proceeding. Adding new code beside existing code needs no extra ask; contradicting or
  modifying existing code does.
- **Token-frugal exploration.** The agent does **not** crawl unspecced code by default — that burns
  tokens for someone who may not want it. It explores existing code **only when needed** for the
  work at hand, or when the user asks.
- **Periodically offer to spec what exists.** From time to time the agent may *suggest* backfilling
  spec for relevant existing code — but only suggest; it acts only on a yes.
- **Pull from existing docs.** The agent may ask whether spec or documentation already lives
  elsewhere (READMEs, design docs, wikis). If so, it asks which files/features those describe, then
  drafts spec for them (subject to the ratify-not-assume rule above).

## Roadmap
Remote spec sources — known integrations (e.g. Confluence) that import spec/documentation from
external systems — are a later direction; see `roadmap.md`.
