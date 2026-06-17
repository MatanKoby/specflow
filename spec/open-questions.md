# Open questions

Decisions still under discussion, kept here so they survive beyond any single working session. When
one is settled, **move it into the design file whose concern it matches** (`architecture.md` or
`workflow.md`) and delete it here — don't leave it in two places.

## Workflow / config flow
- **Profile → dimension mapping** — what concrete value each profile (Autonomous / Supervised /
  Reviewed) sets for each of the five levers; this is the starting point a Customize walk edits.
  *(Decided: names are those three; `init` requires an explicit choice — no pre-selected default;
  `commitCadence` is the fifth lever.)*
- **Enforcement coupling** — do strict profiles offer to set up the guardrails (hooks / host CI /
  branch-protection guidance) in the same flow, or is enforcement a wholly separate opt-in step?
  *(Pending: user wants clarification on what this means before deciding.)*

## CLI / upgrade behavior
- **`--dry-run`** (preview init/upgrade) · **`--force`** (overwrite-on-init) · **`--cwd`** (target
  another dir) — keep, or cut as scope creep?
- **NO_COLOR / non-TTY** — real bug: the CLI always emits ANSI; piping embeds escape codes. Fix.
- **`new-batch`** — a scaffolder that appends a batch skeleton to `BUILD_QUEUE.md`. Worth it?
- **Drift detection** — warn (and back up) before `upgrade` overwrites a hand-edited managed file.
- **upgrade refreshes per-agent stubs?** — currently only `AGENTS.md` + procedures refresh; stub
  improvements never reach installed repos. Refresh them too (same drift risk as `CLAUDE.md`)?
- **upgrade changelog** — print "vX → vY changed …" on upgrade.
- **stamp validation** — fail friendly on a corrupt/hand-edited `.spec-batch.json`.
- **runtime Node-version guard** — friendly "needs Node 18+" instead of a cryptic crash.

## Quality / enforcement (later)
- **ESLint + Prettier in CI** — adopt as the code grows.
- **`specflow verify`** — read-only protocol validator (claims match work, no state in the queue,
  commit-grammar lint). Named `verify`, not `check`.
- **`install-hooks`** — opt-in `commit-msg` + `pre-push` running `verify`.
- **host-repo CI template** — an optional Action `init` can drop to run `verify` server-side.
- **npm-pack-manifest test** — assert every template file ships (guards the dropped-dotfile footgun).

## Distribution
- **Automated npm release on tag** — when we decide to publish.
- **Optional Claude plugin** — ship the skills globally for Claude users, in addition to `init`.
