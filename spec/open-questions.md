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
- **Consent flow for `init` injection — decided.** Interactive: **batched consent**; the review
  handoff lets the user `git diff` and remove anything, warned specflow may be limited, and points at
  `specflow verify`. **Non-interactive** (`--agents=` / `--all`): `init` **proceeds** with the
  modifications and notifies the user to check `git status` to approve — safe because `init` never
  commits (no `--yes` flag). **Required vs. optional follows the dependency tiers:** Tier 1
  (`AGENTS.md` region, procedures, stamp) missing → hard error ("can't work properly"); Tier 3 (a
  per-agent file) missing → warning (that agent isn't auto-wired; it works once its file points at
  `AGENTS.md` — single source, so we do **not** inline the protocol into per-agent files).
- **`verify` scope** — `specflow verify` re-checks install integrity, grading by tier (Tier 1 =
  error, Tier 3 = warning; a required file or a managed block missing). Open: is that the **same**
  command as the Batch E enforcement validator (claim-before-work, commit grammar), or a separate
  integrity / `doctor` check? See `BUILD_QUEUE.md` → Batch E.
- **Tier 2 file location** — should the working ledgers (`BUILD_QUEUE.md` / `CLAIMS.md` and their
  `_DONE` archives) move under `specflow/` to declutter the repo root? Tradeoff: root tidiness vs.
  visibility/editability of the active queue + claims, and the `specflow/`-is-mechanism ownership
  boundary. *(under discussion; lean: move the `_DONE` archives under `specflow/`, keep the active
  queue + claims at root.)*

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
