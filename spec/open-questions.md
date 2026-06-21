# Open questions

Decisions still under discussion, kept here so they survive beyond any single working session. When
one is settled, **move it into the design file whose concern it matches** (`architecture.md` or
`workflow.md`) and delete it here — don't leave it in two places.

## Speccing & approval discipline
The gates, the division of authority, the two phases, and the "is this mine to decide?" test are
all **decided** and live in `workflow.md` → *Work-admission gates*. Profile coupling is resolved
(gates are always-on, profile-independent) and gate-1 strictness is resolved (the
in-scope-vs-new-scope test). The **one open piece** is the *mechanism*:
- **Guidelines-only vs. enforced.** v1 is guidelines-only (honor-system, consistent with Batch E).
  Open: whether/when to add an executable check — e.g. `verify` flags a batch whose design isn't in
  `spec/`, or a claim-time precheck. Harden `spec-edit.md` / `claim-batch.md` wording first; fold
  any executable enforcement into **Batch E**. Build home for the wording: a procedure-hardening
  batch (+ optionally **Batch NB**'s planning phase).

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

## init UX (brownfield)
- **Consent flow for `init` injection.** Two-phase inject-with-consent is decided (see
  `architecture.md` → init / upgrade). **Settled: batched consent** — `init` shows everything it
  would inject + why and asks once; the user may allow, then review `git diff` and remove anything,
  warned that specflow may be limited if required pieces go; `specflow verify` re-checks integrity.
  Still open:
  - **What counts as required vs. per-agent-optional** — being worked from the dependency list; the
    chosen handling is **warn-not-abort** (confirm).
  - **Non-interactive consent** — flag-driven `init` (`--agents=` / `--all`, used by agents/CI with
    no human to prompt): grant via a `--yes` flag and refuse to modify existing files without it?
    *(under discussion.)*
- **`verify` scope** — `specflow verify` is referenced by the init review handoff to re-check
  install integrity (required files present, managed blocks intact). Open: is that the **same**
  command as the Batch E enforcement validator (claim-before-work, commit grammar), or a separate
  integrity / `doctor` check? See `BUILD_QUEUE.md` → Batch E.

## Quality / enforcement (later)
- **ESLint + Prettier in CI** — adopt as the code grows.
- **Enforcement → Batch E (research-first).** Today enforcement is honor-system, exactly as in
  Upside. Batch E researches/discusses how to add it incrementally (validator → hooks → CI → branch
  protection) before building. (`verify` / `install-hooks` / host-CI become sub-batches there.)
- **npm-pack-manifest test** — assert every template file ships (guards the dropped-dotfile footgun).

## Distribution
- **Additional install front-ends (post-v1).** *(Decided 2026-06-21: the Go binary fully replaces
  the Node CLI; v1 ships `curl|sh` + Homebrew; the rewrite proceeds now — see `architecture.md` →
  Distribution and `BUILD_QUEUE.md` → Batches G1/G2.)* Open: whether to add an **npm wrapper**
  (`npx specflow` via prebuilt binary, esbuild-style) and **Scoop/Winget** (Windows) after v1.
- **Automated npm release on tag** — when we decide to publish.
- **Optional Claude plugin** — ship the skills globally for Claude users, in addition to `init`.
