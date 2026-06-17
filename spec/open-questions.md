# Open questions

Decisions still under discussion, kept here so they survive beyond any single working session. When
one is settled, **move it into the design file whose concern it matches** (`architecture.md` or
`workflow.md`) and delete it here — don't leave it in two places.

## Workflow / config flow
- **Profile → dimension mapping** — what concrete value each profile (Autonomous / Supervised /
  Reviewed) sets for each of the five levers; this is the starting point a Customize walk edits.
  *(Decided: names are those three; `init` requires an explicit choice — no pre-selected default;
  `commitCadence` is the fifth lever; v1 enforces strict profiles by agent guidelines only, with
  `init` honest about the advisory gap — see `workflow.md` → Enforcement.)*

## CLI / upgrade behavior
- **Upgrade redesign (non-destructive invariant) — HIGH PRIORITY.** `upgrade` must never remove or
  overwrite text authored by a user or another agent, in any file. Replace wholesale-overwrite with
  marker-delimited managed regions (`<!-- specflow:start/end -->`) + drift detection (warn/back-up,
  never silently clobber). This supersedes the earlier "managed files are overwritten" framing and
  retires the stub-refresh question. Tracked as **Batch U**.
- **`--dry-run`** (preview only — writes nothing, prints what init/upgrade *would* create / overwrite
  / skip) · **`--force`** (overwrite existing files on init instead of skipping — conflicts with the
  non-destructive principle; lean against) · **`--cwd <dir>`** (target another directory without
  cd-ing) — keep or cut? *(clarifications given; decision pending)*
- **NO_COLOR / non-TTY** — real bug: the CLI always emits ANSI; piping embeds escape codes. Fix.
  *(drafted into Batch 5, not yet user-approved)*
- **upgrade changelog** — print "vX → vY changed …" on upgrade.
- **stamp validation** — fail friendly on a corrupt/hand-edited `.spec-batch.json`.
- **runtime Node-version guard** — friendly "needs Node 18+" instead of a cryptic crash.

## Quality / enforcement (later)
- **ESLint + Prettier in CI** — adopt as the code grows.
- **Enforcement → Batch E (research-first).** Today enforcement is honor-system, exactly as in
  Upside. Batch E researches/discusses how to add it incrementally (validator → hooks → CI → branch
  protection) before building. (`verify` / `install-hooks` / host-CI become sub-batches there.)
- **npm-pack-manifest test** — assert every template file ships (guards the dropped-dotfile footgun).

## Docs / distribution
- **Demo project in-repo?** — leaning **no**: the README file-map (Batch 4) covers the need. Note
  the npm/npx asymmetry: `npm publish` ships only the `files` allowlist (a demo wouldn't ship), but
  `npx github:` clones the **whole repo** (a demo *would* download, though it never lands in a user's
  project since `init` only copies `templates/`). Final call pending.
- **`new-batch` → "quick spec → batch → execute" flow?** — user likes the richer idea (capture a
  short spec, write a batch, optionally hand to the agent to run) over a plain skeleton-appender.
  Pending: scope it as its own feature with a small design step.

## Distribution
- **Automated npm release on tag** — when we decide to publish.
- **Optional Claude plugin** — ship the skills globally for Claude users, in addition to `init`.
