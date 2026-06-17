# Open questions

Decisions still under discussion, kept here so they survive beyond any single working session. When
one is settled, **move it into the design file whose concern it matches** (`architecture.md` or
`workflow.md`) and delete it here — don't leave it in two places.

## Speccing & approval discipline  `[HIGH]`
- **The speccing/planning flow must force clarify-and-approve gates — the agent should ask, not
  wing it.** Today the agent can read the queue, design a batch, and ship it end-to-end off a vague
  "continue"; in practice it has (Batches U/U2 were designed *and* shipped — including a self-invented
  marker-matching refinement — from a single "let's continue"). The protocol should make the agent
  **stop, ask for clarification, and get an explicit OK before** (a) persisting a design decision to
  `spec/`, and (b) claiming/building a batch that wasn't already approved. **"Continue" must not be
  read as blanket authorization to design new scope.**
- **Where it lives.** `spec-edit.md` and `claim-batch.md` already half-state this (`spec-edit.md` →
  "don't quietly make and persist a decision" for mid-execution forks), but it's advisory and easy to
  skip past. Decide how to harden it: sharper procedure wording, an explicit **approval-gate step** in
  `claim-batch` (a batch that introduces new design can't be claimed until the user has OK'd the
  design), and/or building the clarify+approve loop into the **Batch NB `--new-batch` planning flow**
  (the planning phase asks the questions and ends on an explicit user OK before anything is written or
  claimed).
- **Open.** How strict by default — always-ask, or ask-only-when-new-scope-is-introduced (small
  in-scope follow-ups still flow)? Does it couple to the workflow profile (Autonomous waives some
  gates; Supervised / Reviewed don't)? And is the mechanism guidelines-only (honor-system, like Batch
  E enforcement today) or an actually-enforced step?

## Workflow / config flow
- **Profile → dimension mapping** — what concrete value each profile (Autonomous / Supervised /
  Reviewed) sets for each of the five levers; this is the starting point a Customize walk edits.
  *(Decided: names are those three; `init` requires an explicit choice — no pre-selected default;
  `commitCadence` is the fifth lever; v1 enforces strict profiles by agent guidelines only, with
  `init` honest about the advisory gap — see `workflow.md` → Enforcement.)*

## CLI / upgrade behavior
- **`--force`** (overwrite existing files on init instead of skipping — conflicts with the
  non-destructive principle; lean against) · **`--cwd <dir>`** (target another directory without
  cd-ing) — keep or cut? *(clarifications given; decision pending)*
- **`remove-agent`** — build it? Would delete only specflow-authored stub files for an agent
  (`.cursor/rules/specflow.mdc`, the Claude skills) and strip only specflow's marked region from
  shared files like `CLAUDE.md` (never user content). *(decision pending)*
- **NO_COLOR / non-TTY** — real bug: the CLI always emits ANSI; piping embeds escape codes. Fix.
  *(not yet user-approved)*
- **upgrade changelog** — print "vX → vY changed …" on upgrade.
- **stamp validation** — fail friendly on a corrupt/hand-edited `.spec-batch.json`.
- **runtime Node-version guard** — friendly "needs Node 18+" instead of a cryptic crash.

## Quality / enforcement (later)
- **ESLint + Prettier in CI** — adopt as the code grows.
- **Enforcement → Batch E (research-first).** Today enforcement is honor-system, exactly as in
  Upside. Batch E researches/discusses how to add it incrementally (validator → hooks → CI → branch
  protection) before building. (`verify` / `install-hooks` / host-CI become sub-batches there.)
- **npm-pack-manifest test** — assert every template file ships (guards the dropped-dotfile footgun).

## Distribution
- **Automated npm release on tag** — when we decide to publish.
- **Optional Claude plugin** — ship the skills globally for Claude users, in addition to `init`.
