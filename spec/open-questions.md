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
- **Commit & push authority — decided.** Two independent levers (`commit`, `push`) the user sets at
  init; **v0.1 ships these two init-time choices** (stored in the stamp), the rest of the workflow
  config (other dimensions + profiles) deferred to Batch W. When `commit: user`, the agent alerts at
  commit points and supplies a suggested message. `claim-batch.md` / `finish-batch.md` get reworded
  to honor the levers. See `workflow.md` → Commit & push authority. *(Open only: exact init prompt
  wording.)*

## CLI / upgrade behavior
- **`--force`** (overwrite existing files on init instead of skipping — conflicts with the
  non-destructive principle; lean against) · **`--cwd <dir>`** (target another directory without
  cd-ing) — keep or cut? *(clarifications given; decision pending)*
- **`remove-agent`** — build it? Would delete only specflow-authored stub files for an agent
  (`.cursor/rules/specflow.mdc`, the Claude skills) and strip only specflow's marked region from
  shared files like `CLAUDE.md` (never user content). *(decision pending)*
- **NO_COLOR / non-TTY — approved for v0.1.** Real bug: the CLI always emits ANSI, so piped/
  redirected output embeds escape codes. Fix: plain text when stdout isn't a TTY (or `NO_COLOR` set).
- **`next` prints each batch's file spread** — the declared file list's count and the top-level paths
  it touches, printed beside each claimable batch, so a batch that is too wide (see `architecture.md`
  → *Batch size*) is visible before it is claimed rather than after. Read-only, no new state.
  *(raised 2026-08-21 from a context-exhaustion report; decision pending — optional as raised.)*
- **upgrade changelog** — print "vX → vY changed …" on upgrade. *(defer past v0.1.)*
- **No-baseline safety fix + stamp validation — v0.1 (closes risk A).** When a managed file has no
  recorded baseline (stamp missing/corrupt, or the file predates being managed), `upgrade` treats it
  as **drift** — writes `.specflow-new`, does **not** overwrite — instead of refreshing. The
  no-clobber default. Plus friendly failure on a corrupt/hand-edited `config.json`.

## init UX (brownfield)
- **Consent flow for `init` injection — decided.** Interactive: **batched consent**; the review
  handoff lets the user `git diff` and remove anything, warned specflow may be limited, and points at
  `specflow verify`. **Non-interactive** (`--agents=` / `--all`): `init` **proceeds** with the
  modifications and notifies the user to check `git status` to approve — safe because `init` never
  commits (no `--yes` flag). **Required vs. optional follows the dependency tiers:** Tier 1
  (`AGENTS.md` region, procedures, stamp) missing → hard error ("can't work properly"); Tier 3 (a
  per-agent file) missing → warning (that agent isn't auto-wired; it works once its file points at
  `AGENTS.md` — single source, so we do **not** inline the protocol into per-agent files).
- **`verify` is split — decided.** Two separate checks, because they answer different questions at
  different moments:
  - **`specflow verify` — installation integrity.** Tier 1 present & valid (`AGENTS.md` + its block,
    the procedures + blocks, a valid stamp), managed blocks intact (markers found; drift reported),
    Tier 3 agent files present & pointing at `AGENTS.md`. Reads the **working tree**, so it passes on
    a fresh, *uncommitted* `init`. Built in Batch BI.
  - **batching / enforcement check** — Batch E (research-first, later): changed files map to an owned
    claim; no execution-state leaked into `BUILD_QUEUE.md`; commit grammar. Inspects git/diff state,
    so it **must carve out the install bootstrap** (see Batch E).
  **CLI surface — decided:** bare `specflow verify` runs the **installation** check; the batching
  check is `specflow verify --batch` (stubbed "in a later release" until Batch E); `-h` for help.
- **Tier 2 file location — decided.** The `_DONE` archives move under `specflow/history/`
  (`BUILD_QUEUE_DONE.md`, `CLAIMS_DONE.md`); the active `BUILD_QUEUE.md` + `CLAIMS.md` and `spec/`
  stay at the repo root (visibility). Built as part of Batch BI.

## Quality / enforcement (later)
- **Linters** — `gofmt` + `go vet` already run in CI; add `golangci-lint` as the code grows.
- **Enforcement → Batch E (research-first).** Today enforcement is honor-system, exactly as in
  Upside. Batch E researches/discusses how to add it incrementally (validator → hooks → CI → branch
  protection) before building. (`verify` / `install-hooks` / host-CI become sub-batches there.)
- **Step-6 handoff backstop → Batch CH (Claude-only, ships now).** The first concrete slice of the
  "hooks" layer, shipped narrowly ahead of Batch E's general research: a `PostToolUse` hook that
  blocks the loop (`decision:block`) right after a `meta: complete batch-*` commit to force the
  finish-batch step-6 handoff. Claude-only (hooks don't port to other agents), opt-in (pasted into
  `.claude/settings.json`), layered on the portable step-6 text (Batch FH). See `BUILD_QUEUE.md` →
  Batch CH.
- **Embed-manifest test** — assert every `templates/**` file is embedded in the binary (guards the
  `go:embed all:` dotfile footgun) — folded into Batch 3.

## Distribution
- **Additional install front-ends (post-v1).** *(Decided 2026-06-21: the Go binary fully replaces
  the Node CLI; v1 ships `curl|sh` + Homebrew; the rewrite proceeds now — see `architecture.md` →
  Distribution and `BUILD_QUEUE.md` → Batches G1/G2.)* Open: whether to add an **npm wrapper**
  (`npx specflow` via prebuilt binary, esbuild-style) and **Scoop/Winget** (Windows) after v1.
- **Automated npm release on tag** — when we decide to publish.
- **Optional Claude plugin** — ship the skills globally for Claude users, in addition to `init`.
