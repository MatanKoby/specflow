# Open questions

Decisions still under discussion, kept here so they survive beyond any single working session. When
one is settled, **move it into the design file whose concern it matches** (`architecture.md` or
`workflow.md`) and delete it here — don't leave it in two places.

## Speccing & approval discipline  `[HIGH]`
The two-gate **baseline is decided** and now lives in `workflow.md` → *Work-admission gates*
(**approval → spec → batch**, both gates always on). The U/U2 incident — design *and* ship two
batches off one "continue" — is the failure this closes. What remains open is the **mechanism**:
- **Profile coupling** — may the **Autonomous** profile relax gate 1 (let a solo user's agent
  self-approve *in-scope* design), or is the approval gate always-on regardless of profile?
- **Strictness of gate 1** — always-ask, or ask-only-when-*new scope* is introduced (small in-scope
  follow-ups flow without a fresh OK)? Needs a crisp line between "in scope" and "new scope."
- **Mechanism** — guidelines-only (honor-system, like Batch E enforcement) or an enforced step
  (e.g. `verify` flags a batch whose design isn't in `spec/`). Harden `spec-edit.md` /
  `claim-batch.md` wording first; consider enforcement later. Build home: the `claim-batch` precheck
  and/or **Batch NB**'s planning phase.

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
