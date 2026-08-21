# Build Queue — Completed History

One-paragraph summaries of every shipped batch, newest at the top. Skim this for context when
picking a new claim. The full implementation history is in `git log` + `specflow/history/CLAIMS_DONE.md`.

<!-- Append a summary here when you finish a batch (see specflow/procedures/finish-batch.md).
     Format, e.g.:

## Batch 1 — <title>
Shipped <what> in <where>. Key commit `<sha>`. <One line on any follow-up deferred.>
-->

## Batch RC — drift is a state you can leave
Two defects in one loop, both reported from a downstream install running 0.1.8. **The sidecar was the wrong bytes.** `upDrift` wrote the rendered template whole, which is right for an adapter (every byte is specflow's) and a footgun for a marker-delimited file: the warning invites reconciliation, the obvious reconciliation is `mv`, and `mv` threw away everything outside the region. One install carried 27 managed lines in `CLAUDE.md` and 73 lines of its own project guidance below them, with a single warning string covering both tiers and the correct action opposite in each. The sidecar now holds `before + markers + fresh region + after`, so `mv` is correct everywhere and the CLI says `mv` outright. **Drift was also terminal**, which is sharper than it was reported: the adapter tier has adopted an identical file since Batch AF, the region tier had no such check, and `Upgrade` carries the old baseline forward for a drifted file. So a user who followed the printed advice ended with a region matching the *new* template and a baseline still holding the *old* hash: re-drifted on every upgrade, warned on in every verify, with no exit but discarding their edit. `decideUpgrade` now adopts on identical bytes, which self-heals every already-reconciled install on its next upgrade with no verb involved. **The third piece is `specflow waive <file>` (`--all`, `--clear`), and it is a deliberate departure from what the queue entry specified.** The entry proposed `specflow adopt` as "re-record the baseline over a deliberate local edit", and that shape is a trap: recording the edited bytes as the baseline makes the region read as *clean*, so the very next `upgrade` refreshes it and destroys the edit being blessed. The correct primitive is a waiver, and this repo already had the idiom in the spec-file `specflow:size-ok` marker from Batch SZ. A waiver changes no file bytes; it records `local` (the exact bytes waived, so a later edit resurfaces as drift) and `kit` (the template version it was taken against, so `upgrade` reports a waiver specflow has moved past rather than sitting silent forever) in a new `waived` map in `specflow/config.json`, absent until first use so older stamps read back fine. `upgrade` leaves a waived file alone and writes no sidecar, `verify` reports it as a choice rather than a warning, `status` counts it on its own row, and only a *drifted* file can be waived, since waiving a clean one would silently opt it out of every future refresh for nothing. Mode consistency still overrides a waiver on the adapter tier: a leak proves the content is stale specflow text rather than the user's. Key commit `2bda486`, spec in `30cd293` (`spec/architecture.md` → *init / upgrade*, plus the `waived` map under *Config & state*). Verified by `gofmt`/`go vet`/`go test` with seven new tests (out-of-region text survives in the sidecar, `mv` ends the drift across upgrade + verify + status, waive keeps the edit and silences the report, a second edit resurfaces it, `--clear` restores reporting, clean and unmanaged files are refused, and `--all` covers the adapter tier), and a hand run against a scratch install that reproduced the original `CLAUDE.md` failure and showed it fixed. Follow-ups left alone on purpose: the procedures and `templates/**` say nothing about waivers (Batch FS owns that prose, and Batch ED is about to rewrite those files wholesale), and `waive` deliberately does not delete the stale sidecar it makes redundant, since the command's contract is that it touches no files - it prints a line saying the sidecar can go.

## Batch LW — Ledger weight: bound the entry, not just the count
Pruning bounded the ledgers by **count** — `CLAIMS.md` to its 5 newest completed entries, `BUILD_QUEUE.md` to zero completed batches — and both rules were working. Neither bounded the size of a *single* entry, nor the prose that is not an entry at all. A downstream install running at the prescribed retention of 5 still carried a 27 KB `CLAIMS.md`; this repo reproduced both halves (20 KB `CLAIMS.md`, and 59 of `BUILD_QUEUE.md`'s 149 lines sitting above the first batch heading). Three changes. **(1) The narrative is written once.** `finish-batch` asked for prose about one batch in two files, authored independently, with neither a superset of the other. `CLAIMS.md` now takes a **stub** — metadata, at most 8 lines of "What shipped", and a pointer — and `specflow/history/BUILD_QUEUE_DONE.md` takes the **full narrative**, since nothing reads the archive on the hot path while `CLAIMS.md` is re-read on every claim, finish, and prune. `specflow finish` grows `--stub-file` (`--summary-file` kept as the old name, so an agent on an older procedure copy still files its prose) and refuses an over-length stub *before writing anything*, so the fix is to move prose into the done-file and retry against an untouched repo. Hard reject rather than stop-and-ask: unlike splitting a spec file, moving a paragraph into the archive loses nothing, so there is no judgment to put to the user. Blank lines and the pointer don't count against the cap. **(2) The queue preamble is capped at 45 lines** (the shipped template is 33), reusing the `specflow:size-ok` stop-and-ask from Batch SZ and re-asking every +15, with a new `prune-ledgers` section 3 that sorts preamble paragraphs into delete / relocate-via-`spec-edit` / keep. That section *does* ask, because deciding which spec file owns a stranded paragraph is a judgment call about concerns. The generalisable finding, now recorded in `spec/architecture.md` → *Ledger lifecycle*: the preamble is where an agent parks a durable fact when it cannot decide which `spec/` file owns it — at finish time the queue is already open, writing there is one edit, and no retention rule ever comes back for it, so it fills. **(3) `specflow next` and `specflow verify` report both ledgers' line counts** and warn past a bound (the retention count, and the preamble cap, honoring the waiver). This is reporting, not a second retention rule: the cut stays a count for its determinism, and what a count cannot reveal is a file that grew heavy while its count stayed correct. It absorbs the optional companion Batch NX was carrying, which NX's section no longer claims. Dogfooded in the same batch: this repo's preamble went 59 → 28 lines, with the v0.1.x release history relocated to `spec/roadmap.md` → *Release lines*, and the stale "v0.1.7 is open, not tagged" line deleted. Key commit `5dcced8`. Verified by `gofmt`/`go vet`/`go test` plus four new tests (over-cap refusal writes nothing anywhere, the cap counts prose only, the legacy flag still lands, and weight reporting warns then honors the waiver), and a self-hosted `upgrade` refreshed the managed procedure and skill copies.

## Batch CD — Batch-width and prune discipline in the procedures
Two guidance changes distilled from a session that exhausted a 373 K-token context window. Batches are now sized by **the layers they cross**, not the deliverables they list (`AGENTS.md` → *The work queue*, plus `spec-edit.md` where batch sections get written); a batch spanning more layers than its goal needs is split on the layer seam, which also gives the pieces the disjoint file lists that parallel claiming requires. What counts as a layer stays per project on purpose. And `claim-batch.md` now tests the ledger-retention rule **before** claiming, not only at finish — more than 5 completed `CLAIMS.md` entries means prune first — because pruning only on the way out leaves the next agent reading the overgrown file on the way in, which is where the cost actually lands. A third item from the same report (a write-side rule preferring the file-editing tool over shell edits, overriding harness modes) was discussed and **not adopted**; it is parked with its reasoning in `spec/open-questions.md`. Design: `spec/architecture.md` → *Batch size* and *Ledger lifecycle*.

---

*Relocated from `CLAIMS.md` by `specflow migrate-claims`.*

**What shipped**
- **`AGENTS.md` → *The work queue*** (full-only) now states the sizing rule: a batch is sized by the
  **layers it crosses**, not the deliverables it lists, and one spanning more layers than its goal
  needs gets split on the layer seam. The split pieces declare disjoint file lists, which is what the
  parallelism rule directly above it already required.
- **`specflow/procedures/spec-edit.md`** carries the same rule where batch sections are actually
  written (persisting a decision → step 3), so an agent turning a decision into queue entries meets it
  at the moment it matters, not only when reading `AGENTS.md`.
- **What counts as a layer is deliberately per project.** Seams differ per repo (this one has spec,
  templates, root managed files, CLI, tests); an enumerated list shipped in a template would be wrong
  in most installs and would read as a contract rather than a heuristic.
- **`specflow/procedures/claim-batch.md`** tests the retention rule **before claiming**, in the
  eligibility section next to the heading-grep guidance:
  `sed -n '/^## Completed/,$p' CLAIMS.md | grep -c '^### '` — more than 5 means run `prune-ledgers`
  and commit it on its own before the claim. Pruning at finish alone doesn't help whoever claims next:
  they still read the overgrown ledger on the way in, and that read is the cost. Same 5 that `finish`
  enforces, so no second threshold and no stop-and-ask.
- **Verification:** `init` into temp repos in both modes — full carries both changes, spec-only
  carries neither and still never names the queue. `specflow verify` clean, `config.check` green,
  managed regions refreshed by a self-hosted upgrade.

**Not shipped, on purpose**
- The report's write-side context rule (edit via the file tool, not the shell, outranking harness
  modes) was **dropped after discussion with Matan**: the tool is a proxy for the real cost, which is
  re-emitting a file's contents to change part of it, and the "outranks your harness" clause has
  specflow claiming jurisdiction over the host environment. Parked with its reasoning in
  `spec/open-questions.md` → *CLI / upgrade behavior*, alongside the optional `next` file-spread item
  (**Batch NX**, `[NOT READY]`).

## Batch RN — Authored release notes
**Batch RN — Authored release notes.** The GitHub Release body is now written, not generated:
`.github/release-notes/<tag>.md` is authored in the same commit as the version bump and passed to
GoReleaser via `--release-notes`, so a pushed tag publishes it directly. Writing it in the release
commit is the mechanism rather than a convention — the tag push is irreversible, so the notes are
reviewed in the same diff as the bump, and editing a published body afterwards needs a GitHub API
token that `git push` over SSH doesn't provide. A missing file falls back to the generated changelog
with a job-summary warning and never fails the job. The house style (action first, then what changes
on disk, then behavior changes an agent may be parsing) is specced in `architecture.md` →
Distribution and keyed to the real reader: an agent deciding whether a repo it maintains needs
`specflow upgrade`. `v0.1.6`'s notes are backfilled as the worked example; its published release
keeps the generated body. Repo-internal — no version line. Commit `26b2cbc`.

---

*Relocated from `CLAIMS.md` by `specflow migrate-claims`.*

**What shipped**
- **`.github/workflows/release.yml`** gained a `Resolve release notes` step: it maps the pushed tag
  to `.github/release-notes/<tag>.md` and, when that file exists, runs GoReleaser with
  `--release-notes=<path>`. Confirmed against the real GoReleaser v2 binary that the flag exists and
  "will skip GoReleaser changelog generation", so the authored body replaces the commit list rather
  than sitting beside it.
- **The fallback is the default branch**, not an error branch: no notes file → the exact args the
  workflow ran before, plus a `⚠` line in the job summary. Neither branch can fail the job. A
  release that ships no archives is far worse than one with a plain body, and that failure mode
  (v0.1.3, v0.1.4) is the reason `draft: false` exists in the first place.
- **`CLAIMS.md` → *Releases need the user's approval*** now states that `meta: release vX.Y.Z`
  covers the notes file as well as the version bump, and why a missed file stays missed (fixing a
  published body needs an API token; `git push` over SSH doesn't provide one).
- **`.github/release-notes/v0.1.6.md`** backfilled as the worked example, in the shape the spec
  specifies. The published v0.1.6 release keeps its generated body — the user explicitly didn't want
  it updated.
- **Scope addition:** a comment in `.goreleaser.yaml` above `changelog:` marking that block as the
  fallback path, since `--release-notes` bypasses it entirely and the config otherwise reads as
  live on every release.

**Verification.** Both workflow branches simulated locally against real `GITHUB_OUTPUT` /
`GITHUB_STEP_SUMMARY` files: present → `release --clean --release-notes=.github/release-notes/v0.1.6.md`;
absent → `release --clean` plus the warning. Both YAML files parse. `act` isn't available here, so
the job itself is unproven until the next tag — the first real test is the next release.

**Follow-up.** The notes file is a convention the workflow can't enforce at commit time (it only
sees the tag). If a release ever ships with the generated body again, the next lever is a `ci.yml`
check on `meta: release` commits.

## Batch AF — Adapter files upgrade like everything else
**Batch AF — Adapter files upgrade like everything else.** Closed the create-once hole that froze
every install's skill stubs and handoff hook at whatever shipped on install day: the marker-less
adapter files are now managed as *whole* files on the same drift contract as a region (clean is
replaced, edited is left alone with a `.specflow-new` sidecar), with a one-time adoption path that
carries installs predating the tier across — identical to the template adopts silently, anything
else is backed up to `.specflow-bak` first. `verify` covers the adapters instead of passing a
deleted or mangled one clean, `status` gained a `stale` row that separates "you edited it" from
"specflow moved and this file didn't", and the four skill stubs finally name the queue verbs
v0.1.5 shipped into the procedures. Also fixed `normalizePath`, which stripped the leading dot from
`.claude/`-style paths and hid them from the queue's overlap check. Commit `f5f578f`.

---

*Relocated from `CLAIMS.md` by `specflow migrate-claims`.*

**What shipped**
- **The second managed tier.** `MANAGED` only ever covered files with a marker-wrapped region, so
  the wholly-generated adapters — the four Claude `SKILL.md` stubs and
  `.claude/hooks/specflow-handoff-reminder.sh` — were create-once: `upgrade` placed one when absent
  and otherwise left it alone forever. `internal/kit/kit.go` now hashes each adapter *whole* into
  the stamp's `managed` map (`adapterEntries` / `decideAdapter` / `adapterDecisions`), and both
  tiers funnel through one `applyDecision`, so a region and an adapter in the same state are
  written and reported identically.
- **The one-time adoption.** Every install made before this has no adapter baselines at all. On the
  next `upgrade` each adapter is compared to the current template: identical → record the baseline
  silently; anything else → `.specflow-bak` then replace. That is what carries existing installs
  across; from then on the normal clean/drift rules apply. Verified on this repo: `upgrade` adopted
  all four stubs, the hook was already current, and the four backups were byte-identical to `HEAD`.
- **`verify` and `status` read the same decisions.** `verify` now lists the adapters (a deleted,
  truncated, or hand-mangled one used to pass clean); `status` gained a **stale** row, separating
  "you edited it" from "specflow moved and this file didn't" — it printed the 4 behind stubs on this
  repo before the upgrade and `none` after.
- **The stubs learned the verbs.** A skill loads *before* the procedure, so its "In short:" is what
  gets acted on, and all four described hand-editing markdown only. Each now names its verb
  (`specflow claim`, `specflow finish`, and for `prune-ledgers` the fact that `specflow finish`
  already does the `CLAIMS.md` half); `spec-edit`'s addition is `full-only`-gated so a spec-only
  install still names no queue machinery.
- **Drive-by fix.** `normalizePath` in `internal/kit/queue.go` trimmed a leading `.` along with
  trailing prose punctuation, so `.claude/…`, `.github/…`, and `.cursor/…` compared as different
  files and were invisible to `specflow next`'s overlap check. Caught by this batch's own file list.

**Verification.** 10 new tests in `cmd/specflow/main_test.go` (init baselines the adapters; a stale
adapter refreshes with no backup; a no-baseline install adopts, backing up only what differs; an
edited adapter is untouched with a `.specflow-new` sidecar; a deleted one is caught by `verify` and
restored by `upgrade`; each stub names its verb; the spec-only `spec-edit` stub names none; dotfile
paths keep their leading dot). `gofmt`/`go vet`/`go test ./...` clean, and the whole flow was run
end-to-end against this repo's own install.

**Follow-up.** An existing install gets one `.specflow-bak` per adapter it didn't already match —
expected and reported by `upgrade`, but worth a line in the release notes so users know to delete
them.

## Batch QV — Queue verbs (`next`, `claim`, `finish`)
Shipped the three queue verbs. `specflow next [--json]` answers the whole eligibility section of
`claim-batch.md` in one read-only call (tag, already-claimed, dependency, file overlap), `specflow
claim <id>` writes the In-progress entry, and `specflow finish <id> --commit <sha>` moves the entry
to `## Completed`, deletes the batch from the queue, files both archive paragraphs, and prunes to
the 5 newest. Batch sections now have a declared shape the parser reads, and a batch missing a field
is reported unparseable rather than offered. The CLI owns placement, format, and timestamps; the
agent still writes every word of prose (`--summary-file`, `--done-file`), and no verb commits. The
procedures name the verbs as the fast path while keeping every manual step. Key commits `80df6c4`, `2346480`, `a7df418`.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped**
- **`internal/kit/queue.go`** (new, 500 lines): the declared-batch-shape parser plus the three verbs.
  `ParseQueue` reads the heading (`## Batch <id> [TAG] — <title>`, backticked or bare tag), the
  optional `**Depends on:**` line (parenthetical rationale ignored, "none" understood), and the
  `### Files this batch creates/edits` list (backticked paths, `dir/{a,b}.md` brace-expanded). A
  batch missing the file list, or sharing its id with another section, comes back with `Problem` set
  and is never offered as claimable. `ParseClaims` errors out if either `##` section heading is
  missing, which is what keeps a hand edit from being rewritten.
- **`specflow next [--json]`** — read-only. Applies the whole Eligibility section of
  `claim-batch.md` in one call, in the order the procedure states it: tag → unparseable → already
  claimed → dependency → file overlap with an in-progress batch. Blocked batches print with the
  reason; the JSON form carries id, title, files, and reason.
- **`specflow claim <id> [--as <agent>]`** — writes the In-progress entry (heading, Owner from
  `config.agents`, `Started` in UTC) and refuses any batch `next` would not offer. `--as` is
  required only when several agents are wired.
- **`specflow finish <id> --commit <sha> [--summary-file <path>] [--done-file <path>]`** — does
  steps 2, 3, 4, and 4a of `finish-batch.md`: entry to the top of `## Completed` with
  Finished/Commit/summary, batch section deleted from `BUILD_QUEUE.md`, paragraph filed in
  `BUILD_QUEUE_DONE.md`, `## Completed` pruned to its 5 newest with the overflow moved verbatim to
  `CLAIMS_DONE.md`. Every edit is computed in memory first, so a parse failure writes nothing.
- **No verb commits**, and no verb writes prose: `--summary-file` (or `-` for stdin) and
  `--done-file` carry the agent's words, and the CLI owns only placement, format, and timestamps.
  Omitting either flag still moves the batch and names what the agent owes by hand.
- **Procedures name the verbs as the fast path** (`claim-batch.md` Eligibility + Claim,
  `finish-batch.md` step 1, `prune-ledgers.md` When-to-run), with every manual step kept intact so
  non-CLI agents and hand edits work exactly as before. The queue template documents the declared
  shape.

**Verification.** `test -z "$(gofmt -l cmd internal)" && go vet ./... && go test ./...` green.
Nine new tests: table-driven parser cases (tag forms, dependency forms, brace expansion, missing
file list, empty file list, duplicate ids), the eligibility and overlap rules, the JSON shape,
claim's refusals, a full `next` → `claim` → `finish` round trip asserting the prune boundary at 5
and verbatim archiving, an unparseable-ledger case asserting nothing is rewritten, and a check that
no verb creates a commit. This batch was itself finished with `specflow finish`, which is how the archive-ordering bug fixed in
`a7df418` was caught: both archives are newest-first, and the pruned entry was landing at the end.

**Deviation from the declared file list.** The parser landed in a new `internal/kit/queue.go` rather
than in `kit.go`, which is already 1314 lines. No other batch was in progress, so nothing raced.

**Follow-up.** `next` reports Batch P as unparseable (no declared file list). That is user-owned
queue prose and P is `[NOT READY]`, so it was left alone.

## Batch CE — Context economy + `config.check`
Cut the protocol's recurring per-batch context cost. The four procedures now name the cheap read
(grep the headings, slice the one section) wherever they say what state to check; `AGENTS.md` gained
a *Working economically* section (batch independent reads, never re-read your own write, one check
command), with its queue-specific bullets `full-only` so spec-only installs stay clean; and
`config.check` records the repo's single check command, asked at `init` (skippable, or `--check=`),
shown by `status`, quoted back by `finish-batch`, and never executed by specflow. Named `check`
rather than `verify` to avoid colliding with the `specflow verify` command. Key commit `25f419e`.
Measured basis: 41x on `CLAIMS.md`, 33x on `BUILD_QUEUE.md`, both read 3 to 5 times per batch.
Follow-up: `upgrade` does not backfill `check` into older installs (absent reads as "not set").

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** Three changes aimed at the recurring per-batch context cost, all of them reaching
existing installs through `upgrade`:

1. **Read-shape steps in all four procedures.** Wherever a procedure said *what* state to check, it
   now also names the cheap read: `grep` the headings, then slice the one section by line number.
   `claim-batch` carries the two greps for its eligibility checks, `finish-batch` and `prune-ledgers`
   locate entries by line range, and `spec-edit` extends the read-the-index rule to inside a file.
2. **`AGENTS.md` → *Working economically*.** Read by headings; batch independent reads into one turn;
   never re-read to confirm your own write; run the one check command; read the batch's declared file
   list first. The queue-specific bullets are wrapped `full-only`, so a spec-only install does not
   get told about machinery it doesn't have.
3. **`config.check`.** A new config string: the repo's single check command. `init` asks for it
   (skippable) or takes `--check=`, `status` shows it, and `finish-batch` quotes it back before the
   final commit. specflow never validates or executes it.

**Why.** Instrumented over one long batch in a specflow-managed repo, file reads were 45% of tool
calls and 88% of context spend, and the single biggest read was a `cat` of a 22 KB `CLAIMS.md` to
answer a question its headings settle. Measured here: 419 bytes of headings against 17.2 KB for the
whole file (41x), 224 bytes against 7.5 KB for the queue (33x), and both ledgers are read 3 to 5
times per batch. Rationale in `spec/architecture.md` → *Context economy*.

**Two decisions worth carrying.** The field is named `check`, not `verify`, because `specflow verify`
already means install-integrity and an agent told to "run config.verify" would plausibly type the
wrong command. And this is deliberately *not* a smaller `CLAIMS.md` retention: retention is a count
of 5 for reasons already recorded under *Ledger lifecycle*, and the waste was the read shape, not
the entry count.

**Verification.** `go vet` + `go test ./...` green, including five new tests: `--check=` recorded and
surfaced by `status`; the skipped answer stored as an empty string; a check command containing quotes
and backslashes still producing valid JSON (the config template is text-substituted, so the value is
`json.Marshal`-escaped); the interactive prompt; and a legacy install whose config has no `check` key
at all, where `status` degrades to "not set". Rendered `AGENTS.md` checked in both modes (5 bullets
full, 4 spec-only) with `specflow verify` clean in each. Self-hosted `upgrade` refreshed this repo's
own managed files, `drift none`, and the repo now records its own check command.

**Follow-ups deferred.** `upgrade` does not backfill `check` into installs that predate it: the key
stays absent, which reads as "not set", and `status` prints the one-line hint on how to add it.
Batch QV depends on this batch and is next.

## Batch RD — Release auto-publish, and the user approves every release
Flipped `.goreleaser.yaml` `release.draft` to `false`, so a pushed `v*` tag now publishes a public
release with its archives attached instead of a draft awaiting a manual click. The draft gate had put
the release *copy* in front of the *binaries*: v0.1.3 and v0.1.4 were each created by hand in the
GitHub UI rather than published from GoReleaser's draft, shipping releases with **zero assets** while
the real drafts sat unpublished, and since `install.sh` resolves `releases/latest` the public
`curl … | sh` install 404'd both times until repaired by hand (v0.1.1 has no release at all for the
same reason). The trade is that release notes become GoReleaser's generated commit list, edited after
publish rather than written before. Because a tag push is now instantly public and irreversible, the
human checkpoint moved earlier rather than vanishing: the counterpart rule at the top of `CLAIMS.md`
requires the user's explicit approval for every release, covering the version bumps, the tag, and the
push. Decision recorded in `spec/architecture.md` → artifact host. Verified with `goreleaser check`
against `@latest` v2, matching the workflow's `~> v2`. Key commit `e279fb9`. Follow-ups deferred:
changelog prettification (`release.header`, `changelog.groups`), backfilling a v0.1.1 release, and
the end-to-end proof that the next tag lands published with 6 assets.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** `.goreleaser.yaml` `release.draft` flipped `true` → `false`, so a pushed `v*` tag
now publishes a public release with its archives attached and no manual step. Plus the counterpart
rule at the top of this file (*Releases need the user's approval*) and the decision recorded in
`spec/architecture.md` → artifact host.

**Why.** The draft gate put the release *copy* in front of the *binaries*. v0.1.3 and v0.1.4 were
both created by hand in the GitHub UI instead of published from GoReleaser's draft, so each shipped
a release with **zero assets** while the real draft sat unpublished beside it. `install.sh:55`
resolves `releases/latest` and downloads `specflow_<ver>_<os>_<arch>.tar.gz` from it, so the public
`curl … | sh` install 404'd both times until it was noticed and repaired by hand. v0.1.1 has no
release at all for the same reason. A body can be edited after publish; a missing archive cannot.

**The trade, and where the checkpoint went.** Auto-publish means the release notes are GoReleaser's
generated commit list (the `^meta:`/`^spec:`/`^docs:` filters leave just the `batch-*:` lines — two
of them for v0.1.4) rather than the user's prose, which is now an edit made after publishing. It
also means a tag push is instantly public and effectively irreversible, so the human checkpoint did
not disappear, it moved earlier: from "publish the draft" to "should we tag at all", which is the
user's call every time.

**Verification.** `goreleaser check` passes against `@latest` v2 — matching what the workflow's
`version: "~> v2"` resolves to. (v2.5.0 rejects the config on `archives.formats`, a field added
later; that is a stale-pin artifact, not a config defect.) `go test ./...` green, `specflow verify`
clean on all 7 managed files.

**Follow-ups deferred.** Changelog prettification (`release.header`, `changelog.groups`) — the
generated body is adequate. Backfilling a v0.1.1 release. The end-to-end proof is the next tag push:
confirm it lands published with 6 assets and no manual step.

## Batch PR — Ledger pruning (`prune-ledgers`, the fourth procedure)
Closed the gap that let `CLAIMS.md` grow without bound: `finish-batch` appended every completed
entry and no procedure ever wrote to `specflow/history/CLAIMS_DONE.md`, so the archive shipped with
its 206-byte header untouched in every install. New `specflow/procedures/prune-ledgers.md` keeps the
**5** newest `## Completed` entries and moves older ones there verbatim, never touching
`## In progress`; retention is a count rather than a byte budget so pruning cuts on an entry
boundary and two agents reach the same result (measured on a 26-entry host install: 35 to 126 lines
per entry, median 61). Includes a catch-up pass for installs predating it and a narrow
`BUILD_QUEUE.md` sweep for completed/dissolved/absorbed sections. `finish-batch` delegates as step
**4a** (numbered, not renumbered, so the "step-6 handoff" references survive); the rules live in the
procedure and the Claude skill stays a thin trigger so pruning does not become Claude-only. Go side:
`specOnlyOmits` + `kit.QueueTokens`. Key commit `dd7a1e9`. Follow-up deferred: nothing enforces
pruning until Batch E, where `verify --batch` is the natural home for a retention check.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** A fourth procedure, `specflow/procedures/prune-ledgers.md`, plus its thin Claude
skill. `CLAIMS.md` had no pruning mechanism at all: `finish-batch` appended each completed entry to
`## Completed` and nothing ever reached `specflow/history/CLAIMS_DONE.md`. The archive shipped and
`AGENTS.md` documented it, but no procedure wrote to it. The only "archive when it grows long"
sentence lived *inside* `CLAIMS_DONE.md`, a file agents are told is reference-only and never open,
and it carried no threshold. So this was a missing step, not a skipped one, and no agent was at
fault for the bloat.

**The rule.** `## Completed` keeps its **5** newest entries; older ones move verbatim to
`CLAIMS_DONE.md`, newest at top. `## In progress` is never touched (a stale claim is a handoff
question, not something to archive). Retention is a **count, not a byte budget**, so pruning always
cuts on an entry boundary and two agents pruning independently reach the same result. Includes a
**catch-up pass** for installs that predate the procedure, and a `BUILD_QUEUE.md` sweep for sections
whose batch is already completed, dissolved, or absorbed.

**Grounded in real data, not a guess.** The first draft of the spec hard-coded N=5 on assumption and
the user stopped it. Measured against a long-running host install (`agents/kapara`): 26 completed
entries, 1,789 lines, 125 KB of `CLAIMS.md`, against a 206-byte untouched archive header, byte-for-
byte the same failure as this repo's own 18-entry ledger. Entry lengths there ran 35 to 126 lines,
median 61: a 3.6x spread, which is what settled count over byte budget. That install's queue was
healthy (549 lines of `BUILD_QUEUE_DONE.md` prove `finish-batch` step 4 is followed), so the sweep
is deliberately narrow.

**Design choices worth keeping.** (1) A separate procedure rather than an inlined finish-batch step,
because an overgrown ledger needs a many-entry catch-up unrelated to any batch finishing, and the
user wanted to run it by hand. (2) `finish-batch` delegates as **step 4a**, not a renumber:
"step-6 handoff" is a named concept across the Claude hook, the spec, and the README, and
renumbering would break every reference. (3) Rules live in the **procedure**, skill stays a thin
trigger, per the cross-agent invariant. Putting them in the skill would silently make pruning
Claude-only. (4) No stop-and-ask, unlike the spec 600-line cap: archiving is lossless and mechanical,
and `claim-batch` already resolves dependencies against either location.

**Go side.** `prune-ledgers` joins `specOnlyOmits` (procedure + skill) and `kit.QueueTokens` so a
mode leak is caught. `MANAGED` already covers `specflow/procedures` as a directory, so `upgrade`
picked the new file up with no change there.

**Verified against the real CLI, not assumed.** Built and ran it on this repo: `upgrade --dry-run`
first flagged the hand-edited copies as drifted (correct behavior, since their baselines no longer
matched), so the root copies were removed and rewritten by `upgrade`, which recorded the new
baseline hash. `verify` now lists four procedures. `go test ./...` uncached green, `gofmt`/`go vet`
clean. Two new tests lock the retention number, the no-stop-and-ask framing, the delegation, the
thin-trigger skill (with a line-count ceiling), and the spec-only omission. Dogfooded immediately:
this entry's own finish ran step 4a and archived 13 entries, taking `CLAIMS.md` from 18 completed
entries (~520 lines) to 5 (245 lines).

**Follow-ups deferred.** (1) Nothing enforces pruning. Like the rest of the protocol it is
honor-system until Batch E, and `verify --batch` is the natural home for a "CLAIMS over retention"
check. (2) Host installs need `specflow upgrade` plus one manual `prune-ledgers` run to catch up;
kapara has 21 entries to archive. (3) The kapara agent was never consulted: the message sat unread
behind a cross-session approval gate, so the numbers here were read from its files directly.

## Batch 4 — README badges + file-map
Rewrote `README.md` ahead of a public launch post, directed by the user. Three badges (CI,
auto-tracking release, MIT); a centered hero with an ASCII `specflow init` console demo standing in
for the still-deferred animated GIF; five **Why** blocks led by continuity ("your agents forget; the
repo doesn't") with token savings third and its spec-coverage condition stated inline; a mermaid
flowchart of spec → queue → claim → build showing two agents claiming in parallel; and the batch's
file-map deliverable as an annotated tree with `[owner]` tags plus the ownership table. User-added
scope: the README is now **agent-executable** — an HTML comment at the top of the raw file routes an
agent to a `## For AI agents` section of seven ordered steps (ask the user which agents and which
mode rather than guessing, `init` never commits, relay the Claude-Code handoff hook block, read
`AGENTS.md` before any work). Drafted claims were checked against a live `init` in a temp repo, which
corrected the file count (16, not 11) and the `spec/` contents (only `README.md` at install). Key
commit `041f1bc`. Follow-ups: the *How it differs* competitor paragraph is unverified against those
projects' current behavior, and a 1280x640 social-preview image was recommended but not produced.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** A full README rewrite, directed by the user ahead of a public launch post. Three
badges (CI, auto-tracking release, MIT). New structure: centered hero (badges + nav + a hand-written
ASCII `specflow init` console demo) → causal-chain pitch → five **Why** blocks → Install → Quick
start → How it works (four prose steps + a mermaid flowchart showing two agents claiming in
parallel) → file map → Agents → Who it's for → two `<details>` blocks (upgrade/status/verify,
spec-only) → How it differs → For AI agents.

**File-map deliverable.** An annotated file tree with `[owner]` tags per path, followed by the
existing ownership table. Tree for orientation, table for the contract.

**Agent-accessibility (user-added scope).** The README is now executable by an agent told "install
specflow here": (1) an HTML comment as the **first thing in the raw file** pointing agents at the
right section (invisible when rendered); (2) a `## For AI agents` section with seven ordered
imperative steps — confirm git repo, install binary, **ask the user which agents + which mode rather
than guessing**, run `init` non-interactively, show the diff and commit as `meta: install specflow`,
relay the Claude-Code handoff hook block with its rationale, then read `AGENTS.md` before any work;
(3) step 7 states plainly that the README is the pitch and `AGENTS.md` is the protocol, so an agent
does not try to work from the README.

**Verified, not assumed.** Ran a real `init --agents=claude,cursor` into a temp repo, which corrected
two drafted-from-memory errors: the install writes **16 files, not 11**, and `init` creates only
`spec/README.md` (no `architecture.md` / `research/`), so the tree now shows the near-empty `spec/`
and frames that as intentional. `--version`, `add-agent`, `status`, and `verify` output were each
confirmed before being described; the hook JSON is copied from live `init` output. Anchors, code
fences, and `go vet` / `go test` all check.

**Positioning choices.** Lead is continuity ("your agents forget between sessions; specflow makes
that not matter"), with token savings as the third Why block and its "where the spec covers the
ground" condition stated inline. Added an explicit *honest about enforcement* callout (the protocol
is written guidelines, nothing executable checks it — Batch E territory). No em dashes, per the
user's global writing rule.

**Follow-ups deferred.** (1) The animated demo (GIF/asciinema) stays deferred as the batch specified
— the ASCII console block is the designed visual standing in for it. (2) The *How it differs*
paragraph (Spec Kit / Kiro / OpenSpec / BMAD) was written from
`spec/research/2026-07-competitive-landscape.md` and **not re-verified against those projects' current
behavior**; worth a check before the launch post. (3) A 1280x640 GitHub social-preview image was
recommended to the user and not produced.

## Batch SZ — spec-file 600-line hard cap
Turned the size rule in `spec-edit.md` from a nudge that had never fired (largest spec file here:
177 lines) into an enforced stop, and dropped its "~20k tokens" gloss, which overstated the real
figure by about 2x. Before an edit crosses a file's current limit the agent stops and asks the user,
presenting the section headlines, a single-concern claim, and the read-cost warning verbatim; it
never asks for a number. A *keep* is recorded as a `specflow:size-ok` first line with a UTC
timestamp and the next threshold, advancing +200 each time. `archive.md` and `research/` are exempt.
Mirrored into the dogfood procedure and the `spec-edit` SKILL.md. Key commit `f9fa57b`, dogfood
`02e6a41`. Follow-up: `upgrade` still won't refresh a non-managed adapter file whose content merely
went stale in full mode (SL's repair only fires on a spec-only leak), so the stub was hand-mirrored.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** The size rule in `spec-edit.md` was a nudge ("consider whether the next bite of
content wants its own file") that had never fired: this repo's largest spec file is 177 lines. Its
"~20k tokens" gloss was also about double the real figure (600 lines is roughly 10-11k tokens at
this corpus's 68 chars/line). Rewrote it as a **hard cap**: before an edit crosses the file's
current limit the agent stops and asks the user, presenting the file's **section headlines**, a
**single-concern** claim, and the read-cost warning **verbatim** ("The bigger a spec file is, the
more I read when I need even just a small chunk from it, so it's best the file is small in advance.
But, you're the boss."). It never asks the user to pick a number. A *keep* is recorded as a
`<!-- specflow:size-ok … -->` **first line** of the file with a UTC timestamp and the next
threshold, which advances **+200** each time (600 → 800 → 1000), so a waiver silences one threshold
and never the rule. **`archive.md` and `research/` are exempt** — both grow monotonically by design
and have no concern to split off, so the ask would have no good answer. Mirrored into this repo's
dogfood procedure and into the `spec-edit` SKILL.md (description + body), whose summary loads into
every Claude session. Design ref: `spec/architecture.md` → *Spec organization*.

**Verification.** `go test ./...` uncached green, `go vet` + `gofmt` clean. Three new tests, two of
them asserting the shipped procedure carries all four parts of the ask in both install modes (the
warning matched word-for-word over whitespace-normalized text, since a paraphrase would drop the
"you're the boss" that makes it a question rather than a lecture) and that the superseded nudge
wording and token gloss are gone. The **marker-collision test** proves `specflow:size-ok` is inert
to the region (`specflow:start\b`) and composition (`specflow:full-only:`) regexes on the two
surfaces that can actually parse it: inside a region (the procedure ships the literal as its
example) and above one (a waiver on a managed file). Confirmed it bites by swapping in the
near-miss token `specflow:start-ok`, which **does** match `specflow:start\b` (the `\b` fires before
the hyphen) and silently pushes AGENTS.md onto the drift path. A first draft of that test put the
waiver on `spec/architecture.md` and was **vacuous** — `spec/**` is user-owned, so no specflow
command ever parses it; the exposure is entirely in managed files.

**Known gap, surfaced not fixed.** `upgrade` refreshed the managed procedure here but **not** the
`spec-edit` skill stub: non-managed adapter files are create-once, and SL's `staleAdapterFiles`
repair only fires on a spec-only mode leak. So a *full*-mode install whose stub content merely went
stale still needs a hand-mirror (done here). Worth a batch if stub prose keeps changing.

**Note.** The stub template was rewrapped so its inline composition markers sit at line ends —
stripping them mid-line left a ragged short line in both rendered modes.

## Batch SL — spec-only mode leaks queue/batch language
Fixed a user-reported 0.1.2 defect where `init --spec-only` generated files pointing agents at
`BUILD_QUEUE.md`, `CLAIMS.md`, `claim-batch`, and `finish-batch` — none of which the mode installs.
Gated the six templates that had no mode markers (four adapter rule-files, the `spec-edit` skill
stub incl. its YAML `description:`, and `templates/base/spec/README.md`) with *replacement*
spec-only wording rather than deletion, and widened `CLAUDE.md`'s guards past the step-6 paragraph
to the protocol description and trigger bullets. Added the mode-consistency check baseline hashes
structurally cannot provide (they're taken over the rendered region, so full-mode prose in a
spec-only install matches its own hash — which is why `verify` said "All good" on a broken install):
`kit.ModeLeaks`/`kit.QueueTokens`, shared with the tests, plus the install mode in `verify`'s
output. Made `upgrade` a complete fix path via `staleAdapterFiles`, since non-managed create-once
skill stubs carry no baseline and were being left stale forever. Tests written first and observed
failing across all six files; verified against a real v0.1.2 binary built from `2951d78`. Key commit
`ee4b6b7`, final `3b265e0`. Residue by contract: `spec/README.md` is user-owned so `upgrade` can't
correct existing installs. Follow-up: Batch SZ (600-line cap) unblocked.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** The fix for a user-reported 0.1.2 defect: `init --spec-only` generated files that
instructed agents to use `BUILD_QUEUE.md`, `CLAIMS.md`, `claim-batch`, and `finish-batch`, none of
which the mode installs. (1) **Template gating** — the six templates with no mode markers now carry
`specflow:full-only` / `specflow:spec-only` pairs with *replacement* spec-only wording, not
deletion: the four adapter rule-files (cursor `.mdc` incl. its frontmatter `description:`, copilot,
bob, antigravity), `.claude/skills/spec-edit/SKILL.md` (incl. its YAML `description:`, which loads
into every session's skill listing), and `templates/base/spec/README.md`. (2) **`CLAUDE.md` guards
widened** past the step-6 hook paragraph to the protocol description, "three procedures", and the
claim/finish trigger bullets — it advertised the two skills `specOnlyOmits` had just skipped. (3)
**Mode-consistency check** (`kit.ModeLeaks` + `kit.QueueTokens`, shared with the tests): baseline
hashes are taken over the *rendered* region, so full-mode prose in a spec-only install matches its
own hash and reported clean — hashes prove a region is unmodified, never that it is
mode-appropriate. `verify` now scans managed regions for queue/claim tokens when the stamp records
spec-only, and prints the mode it validated. (4) **`upgrade` made a complete fix path**
(`staleAdapterFiles`): skill stubs and hooks are non-managed create-once files with no baseline, so
upgrade previously left an existing spec-only repo with a wrong `spec-edit` stub forever. The repair
keys off the mode leak itself, which proves the content is stale specflow text; full mode returns
nil from `ModeLeaks`, so a customized stub is never touched.

**Verification.** `go test ./...` uncached green, `go vet` + `gofmt` clean. Four new tests, written
before the fix and observed failing across all six files: a whole-install walk (covers a new
template the day it lands, rather than a hand-maintained file list), a `verify`-catches-leak test,
a legacy-upgrade-repair test, and a full-mode-stub-untouched test guarding the repair from being
destructive. A prose regex on the test side catches queue/claim wording that names no identifier
(`spec/README.md`'s "claiming a batch: read the queue entry first"), which the token list cannot
see. Exercised end-to-end against a **real v0.1.2 binary** built from `2951d78`: the new `verify`
reports 6 problems where 0.1.2 said "All good", and `upgrade` then clears every specflow-owned file.
Propagated to this repo's dogfood install (`upgrade` refreshed `CLAUDE.md`; `verify` clean).

**Known residue, by contract.** `spec/README.md` is user-owned, so `upgrade` must never rewrite it;
an existing spec-only install keeps its line 36 until the user edits it. Fixed at source, so new
installs are clean. Design ref: `spec/architecture.md` → *Install modes*.

**Follow-up.** Batch SZ (600-line cap) was unblocked by this batch and is next in the queue.

## Batch CH — Claude Code batch-boundary hook (opt-in)
Shipped the Claude-only deterministic backstop for the finish-batch step-6 handoff, layered on the
portable FH text. A `PostToolUse(Bash)` hook (`templates/agents/claude/.claude/hooks/specflow-handoff-reminder.sh`)
gates on `git commit` in the command, confirms via the landed HEAD subject `^meta: complete batch-`
(handles `-m`/`-F`/heredoc, proves the commit landed, self-de-dups), and emits
`{"decision":"block","reason":…}` to halt the loop and force the handoff. Soft-deps on `jq`, fails
open so it can never break a commit. The CLI prints an opt-in `.claude/settings.json` paste-block at
the end of `init`/`add-agent` (recommends committed settings; no auto-merge — JSON can't be
marker-merged); the claude `CLAUDE.md` full-only region tells the agent to relay that step. spec-only
omits it (no batch boundary). Also **extended `upgrade`** to place newly-shipped non-managed adapter
files create-once (scoped to `agents/`, never resurrects a deleted base file), so existing installs
receive the hook — previously upgrade only refreshed managed regions. Key commits `c99b734`
(hook + CLI + relay + tests), `2e4b69a` (upgrade convergence), `7b3ffdc` (dogfood upgrade
0.1.0→0.1.1). Design recorded in `spec/open-questions.md` → Quality / enforcement. 4 new tests
(install+notice, add-agent, jq-gated script behavior, upgrade-adds-hook). Follow-up: activating the
hook in this repo's own `.claude/settings.json` is left as a user opt-in.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** The Claude-only deterministic backstop for the finish-batch step-6 handoff, on top
of the portable FH text. (1) **Hook** `templates/agents/claude/.claude/hooks/specflow-handoff-reminder.sh`:
a `PostToolUse(Bash)` script that gates cheaply on `git commit` in the command, then confirms via the
landed **HEAD subject** `^meta: complete batch-` (matching the landed subject, not the command text,
handles `-m`/`-F`/heredoc uniformly, proves the commit succeeded, and self-de-dups since the next
commit is `meta: claim …`), and emits `{"decision":"block","reason":…}` to halt the agentic loop and
feed the step-6 reminder back so the agent must act on it. Soft-deps on `jq`; **fails open** (exit 0)
if jq is absent so it can never break a commit. (2) **CLI** (`cmd/specflow/main.go`): prints an
opt-in paste-notice at the end of `init`/`add-agent` when the hook was just installed (claude + full
mode), recommending committed `.claude/settings.json` (no auto-merge — JSON has no marker-merge
path). (3) **Adapter relay** (`templates/agents/claude/CLAUDE.md`, full-only region): tells the agent
to surface the hook-setup step on install/upgrade instead of burying it. (4) **spec-only omits** the
hook (`internal/kit/kit.go` `specOnlyOmits`) — no batch boundary to backstop. (5) **`upgrade`
extended** to place newly-shipped non-managed adapter files create-once (scoped to the `agents/`
tree, so a user-deleted base file is never resurrected) — without this the backstop reached only
fresh inits, never existing installs.

**Manual prereq.** The hook is inert until registered in `.claude/settings.json`; the CLI/relay
surface the exact block. Activating it in *this* repo is deferred as a user opt-in (see follow-up).

**Verification.** `go test ./...` green, uncached (4 new tests: install+notice, add-agent incl.
no-dupe-on-re-add, jq-gated script behavior asserting block-on-match / silent otherwise, and
upgrade-adds-missing-hook incl. not-resurrecting a deleted base file). `go vet` + `gofmt` clean.
Exercised end-to-end: the shipped script (block/silent cases), `init`/`add-agent`/`upgrade` across
full + spec-only, and the embed-manifest test auto-covers the new file. Propagated to the dogfood
install via `specflow upgrade` (0.1.0→0.1.1, refreshed CLAUDE.md + added the hook here; `verify`
clean, no drift). Design ref: `spec/open-questions.md` → Quality / enforcement.

**Follow-up (deferred).** Registering the hook in this repo's own `.claude/settings.json` to activate
the backstop for agents working here — a user opt-in (committed vs local; loop-blocking behavior),
not bundled into the batch.

## Batch RF — Ship the research-flow convention
Made the lightweight research-note flow a shipped part of specflow, so a fresh install carries the
convention (previously only self-hosted). Three template edits, all self-contained (no queue/claim
refs) so **spec-only inherits them**: `templates/base/AGENTS.md` names the optional pre-design
research step + the gate-free `spec/research/` home in its spec-discipline region;
`templates/base/specflow/procedures/spec-edit.md` gains a *Research notes* section (gate-free /
write-as-you-go / graduate-upward, framed as the pre-design exception to the archive rule);
`templates/base/spec/README.md` documents the optional `research/` sub-folder. Key commit `9fbba41`.
2 new tests (full + spec-only) assert `init` ships the convention in all three files; the existing
spec-only banned-word test guards the sections stay queue-free. Propagated the two managed files to
the dogfood copies via `specflow upgrade` (verify clean). No follow-ups.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** Made the lightweight research-note flow a shipped part of specflow, so a fresh
install carries the convention (not just this self-hosted repo). Three template edits, all
self-contained with **no queue/claim references** so spec-only inherits them: (1)
`templates/base/AGENTS.md` — the spec-discipline region's spec-edit pointer now names the optional
**pre-design research step** and the gate-free `spec/research/` home (dated snapshots, written on the
go, conclusions graduate into `open-questions.md` / `roadmap.md`); (2)
`templates/base/specflow/procedures/spec-edit.md` — a new **"Research notes"** section placed as the
pre-design exception to the archive rule, covering the gate-free / write-as-you-go / graduate-upward
lifecycle; (3) `templates/base/spec/README.md` — documents the optional `research/` sub-folder in the
file map. Design ref: `spec/workflow.md` → *Research notes*, `spec/research/README.md`. The dogfood
`spec/README.md` already carried the convention (it's the project's own spec, not a managed region),
so only the two managed files needed propagation.

**Verification.** `go test ./...` green (2 new tests: `TestResearchFlowConventionShipped` full-mode
and `TestResearchFlowInSpecOnly`, both asserting `init` ships the convention across AGENTS.md +
spec-edit.md + spec/README.md; the pre-existing spec-only banned-word test confirms the research
sections stay queue-free). `go vet` + `gofmt` clean. Propagated the two managed files (root
`AGENTS.md`, root `spec-edit.md`) to the dogfood copies via `specflow upgrade` (dry-run showed exactly
those two refreshing; applied; `specflow verify` clean, no drift). **Follow-up:** none — this is the
last of the queued research-flow work; together with Batch FH's step-6 rework it forms the user-facing
payload of the `v0.1.1` release cut immediately after this batch.

## Batch 3 — Broaden the test suite
Locked behavior beyond the file-existence smoke checks with 4 new tests. **Content assertions** on
the generated `AGENTS.md` (`TestAgentsMdContentSections`): its key sections, the commit-grammar table
header + `batch-N`/`meta:`/`spec:` rows, and the `specflow/procedures/*.md` path references.
**Picker + adapter coverage**: the interactive agent picker over piped stdin (numeric multi-select +
the "a" all-shortcut) and the `--all` flag, asserting each of the five adapters' instruction files is
written, marker-wrapped, and baseline-hashed, then a follow-up `upgrade` stays a clean no-op — first
coverage of the copilot/bob/antigravity adapters. An **embed-manifest test** in the new root-package
`templates_test.go` walks the on-disk `templates/**` tree and asserts every file is embedded
byte-for-byte, counting dot-paths so the `//go:embed all:` dotfile footgun can't regress (proven to
bite: dropping `all:` fails it, then reverted). Key commit `677b265`. `go vet`/`gofmt` clean. The
suggested `internal/kit/` unit split was skipped as redundant with the end-to-end binary coverage.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** Locked behavior beyond the existing file-existence smoke checks, in three areas.
(1) **Content assertions** on the generated full-mode `AGENTS.md` (`TestAgentsMdContentSections`):
its key sections (*Commit & push authority*, *File ownership*, *The work queue*, *The claims file*,
*The procedures*, *Commit message convention*, *Editing rules*), the **commit-grammar table** header
plus its `batch-N` / `meta: claim` / `meta: complete` / `spec:` rows, and that the procedures section
references the real `specflow/procedures/*.md` paths (not bare filenames). (2) **Adapter + picker
coverage**: the interactive agent picker driven over piped stdin — numeric multi-select
(`TestInitInteractivePicksAgentsByNumber`, "3,4,5" → copilot/bob/antigravity, unpicked agents absent)
and the "a" all-shortcut (`TestInitInteractiveAllShortcut`) — plus the **`--all`** flag
(`TestInitAllFlagWiresEveryAdapter`) asserting each of the five adapters' instruction files is
written, carries the managed region markers, and has a baseline hash, then that a follow-up
`upgrade` stays a clean no-op. This is the first coverage of the **copilot / bob / antigravity**
adapters. (3) An **embed-manifest test** in the root package (`templates_test.go`) that walks the
on-disk `templates/**` tree and asserts every file is embedded **byte-for-byte**, explicitly counting
dot-path templates so the `//go:embed all:` dotfile footgun can't regress unnoticed.

**Verification.** `go test ./...` green (4 new tests: 3 in `cmd/specflow/main_test.go`, 1 in the new
root-package `templates_test.go` — the root package had no tests before). `go vet` + `gofmt` clean.
Proved the embed guard actually bites: temporarily dropping `all:` from the `//go:embed` directive
made the manifest test fail on exactly the dropped dotfiles (Claude skills, cursor/copilot adapters),
then reverted to green. **Note:** the embed-manifest test lives in a new root-package file rather than
`cmd/specflow/`, because that is where the `embed.FS` and `Templates()` are — the batch's file list
allowed for splitting tests out. **Deferred (out of scope):** the batch's suggested `internal/kit/`
unit-test split was not done — the behavior is fully covered end-to-end through the built binary, so
a redundant unit layer wasn't warranted now.

## Batch FH — finish-batch step-6 handoff rework
Reworked step 6 of `finish-batch.md` so the end-of-batch context handoff is hard to skip: it now
states the payoff to the user (cheaper + more reliable next batch, a decision point), names and
refutes the "it's noise" rationalization, and requires a fixed terminal handoff line so an omission
is visible. Step 7 clarifies that "continue" authorizes the next claim but doesn't waive the line.
Canonical template edited and propagated to the dogfood copy via `specflow upgrade` (stamp
rebaselined). Key commit `2962ae1`. Portable (text-only, all agents); the Claude-Code deterministic
hook backstop is queued as Batch CH.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** Reworked step 6 of `finish-batch.md` so the end-of-batch context handoff is hard
to skip. Root cause (from a retro): step 6 was the only finish step that produced no artifact, so a
skip was invisible and cheap under throughput pressure and got rationalized away as "noise." The
rewrite (a) states the payoff to the user (cheaper next batch, more reliable next batch, a decision
point), (b) names and refutes the "it's noise" excuse, and (c) requires a fixed terminal handoff
line so an omission is self-evidently non-compliant. Step 7 now clarifies that a user's "continue"
authorizes the next claim but does not waive the step-6 line. Edited the canonical template
(`templates/base/specflow/procedures/finish-batch.md`) and propagated to this repo's dogfood copy
via `specflow upgrade` (stamp rebaselined; `verify` clean). Portable across all agents (text only);
the Claude-Code deterministic backstop is queued separately as Batch CH.

**Verification.** `go test ./...` green, `go vet`/`gofmt` clean. `upgrade --dry-run` showed exactly
one file refreshing; applied; the dogfood copy is back in sync with the template; `specflow verify`
reports all regions intact (no drift).

## Batch 5 — `--dry-run` (preview)
Shipped a `--dry-run` flag on `init` and `upgrade` that prints the planned file operations and exits
without touching disk. `init --dry-run` reuses `PlanInit` (would create / inject / already-wired /
skip), always non-interactive, previewing the default agent when `--agents` is omitted;
`upgrade --dry-run` previews refresh / add / migrate / drift via a new read-only `PlanUpgrade`. The
per-file upgrade classification was factored into a shared `decideUpgrade`/`upgradeDecisions` pair so
the apply path (`Upgrade`, unchanged in behavior) and the planner can't diverge. Key commit `7f5f70b`.
6 new tests + the existing upgrade suite guard the refactor; `go vet`/`gofmt` clean. **Completes
Milestone v0.1** (code-complete; next is tagging `v0.1.0`).

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** A **`--dry-run`** flag on `init` and `upgrade` that prints the exact planned file
operations and exits **without touching disk**. **`init --dry-run`** reuses `PlanInit` and renders
*would create / would inject (content preserved) / already-wired / would skip*; it's always
non-interactive (no agent/consent prompts) and previews the default agent (claude) when `--agents`
is omitted, honoring `--spec-only`/`--all`. **`upgrade --dry-run`** renders *would refresh / add /
migrate (→ `.specflow-bak`) / drift (→ `.specflow-new`, not overwritten)*, and *already current* when
there's nothing to do. **Refactor (no behavior change):** the per-file upgrade classification is now
a shared `decideUpgrade` + `upgradeDecisions` pair consumed by both the apply path (`Upgrade`, kept
byte-for-byte in behavior) and the new read-only `PlanUpgrade` — so the preview and the real run can
never diverge. Help text for both commands documents the flag.

**Verification.** `go test ./...` green (6 new dry-run tests + the full existing upgrade suite,
which guards the refactor). `go vet` + `gofmt` clean. Manually confirmed a fresh-dir `init --dry-run`
and an `upgrade --dry-run` over a drifted+stripped install write nothing (no files, no sidecars, stamp
unchanged).

**Milestone.** This is the **last v0.1 batch** — v0.1's definition-of-done (roadmap.md) is now met in
code. Next: cut the real `v0.1.0` tag (the release pipeline was proven in Batch G2).

## Batch 2 — `specflow status`
Shipped `specflow status`: a read-only orientation summary that writes nothing. `kit.Status`
reports the kit version (stamp vs. binary, with an upgrade hint on mismatch), install mode, wired
agents, commit/push levers, active claims parsed from CLAIMS.md's In-progress section (owner shown;
`none` -> unassigned), the un-done batch count from BUILD_QUEUE.md, and a drift flag for any managed
region edited since install. Spec-only installs report the queue as n/a; not-installed exits
non-zero. Key commit `3f67292`. 7 new tests; `go vet`/`gofmt` clean.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** A read-only **`specflow status`** that orients a user/agent at a glance, writing
nothing. **`kit.Status`** (in `internal/kit/kit.go`) assembles a snapshot from the stamp + repo:
**kit version** (stamp `kitVersion` vs. the running binary, with a *run upgrade* hint on mismatch),
**install mode**, **wired agents**, the **commit/push levers**, **active claims** parsed out of
CLAIMS.md's In-progress section (each `###` heading paired with its `Owner:`; `none`/unset renders as
*unassigned*), the **un-done batch count** (a `^## Batch` count over BUILD_QUEUE.md, since done
batches are removed from that file), and a **drift flag** listing any managed region whose hash no
longer matches its recorded baseline (the same test `verify`/`upgrade` use). The CLI
(`cmd/specflow/main.go`) renders an aligned label/value block (NO_COLOR-aware via the shared paint
helpers), exits non-zero when specflow isn't installed, and marks the queue **n/a** for a spec-only
install (no BUILD_QUEUE.md/CLAIMS.md). Registered in `dispatch` + top-level/`--help` usage.

**Verification.** `go test ./...` green (7 new tests: fresh install, active claims incl.
owner/unassigned, drift flag, version mismatch + upgrade hint, spec-only queue-n/a, not-installed
non-zero exit, and `--help`). `go vet` + `gofmt` clean. Manually exercised across all scenarios.

**Note (not part of this batch).** A parallel session's git worktree appeared at
`.claude/worktrees/` during the batch; a `.gitignore` entry for it would prevent accidental commits
(flagged for the user, left out of scope here).

## Batch 1 — `specflow add-agent <name>`
Shipped `specflow add-agent <name> [<name>...]`: wires another agent's adapter into an
already-initialized repo. `kit.AddAgent` copies the agent's adapter files (skip-existing,
non-destructive), injects specflow's region into an existing instruction file (content preserved) or
leaves an already-wired one alone, records the agent in `config.agents`, and refreshes the
managed-region baselines; mode-aware, so a spec-only repo doesn't gain the claim/finish skills. The
CLI validates agent names up front, guards the not-installed case, and never commits (review-then-
commit handoff, like `init`). Key commit `fa3aa3d`. 8 new tests; `go vet`/`gofmt` clean. Follow-up
deferred: `remove-agent` (decision pending).

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** A new **`specflow add-agent <name> [<name>...]`** command that wires another
agent's adapter into an already-initialized repo. **`kit.AddAgent`** (in `internal/kit/kit.go`) reads
the stamp, no-ops if the agent is already in `config.agents` (reported as *already installed*), then
walks `agents/<key>/` and, per file: **creates** a missing adapter file (rendered for the install
mode); **injects** specflow's marker region into an existing **instruction file** (CLAUDE.md etc.),
preserving the user's content, or **leaves it as-is** when it already carries a region or points at
AGENTS.md; and **skips** any other specflow-owned file already present. It then records the agent in
`config.agents` and refreshes the managed-region baselines for the full agent set (so `upgrade`
tracks the new instruction file). **Mode-aware:** a spec-only install doesn't gain the
claim/finish skills (`specOnlyOmits` filter). The CLI (`cmd/specflow/main.go`) validates every name
up front against the known-agent list (clean error + non-zero exit on a typo), guards the
not-installed case, prints a per-agent summary, and ends with the **review-then-commit** handoff —
**it never commits** (same discipline as `init`). Registered in `dispatch` + top-level/`--help`
usage.

**Verification.** `go test ./...` green (8 new tests: adapter+stamp+managed, multi-add +
already-present no-op, brownfield inject, already-wired left-as-is, spec-only skill omission,
unknown/not-installed guards, post-add `verify`, and `--help`). `go vet` + `gofmt` clean. Manually
exercised end-to-end across all scenarios.

**Follow-ups deferred.** `remove-agent` (still *decision pending* in `open-questions.md` → CLI /
upgrade behavior) is out of scope here.

## Batch G2 — Go release + install pipeline
Shipped zero-runtime distribution. **GoReleaser** (`.goreleaser.yaml`) cross-compiles a 5-target
matrix (linux/darwin amd64+arm64, windows amd64) into checksummed `tar.gz`/`zip` archives named
`specflow_<version>_<os>_<arch>`; a **`Release` workflow** on any `v*` tag runs `goreleaser release
--clean` → a **draft** GitHub Release; an **`install.sh`** `curl … | sh` front-end detects OS/arch,
checksum-verifies, and installs the matching binary (`/usr/local/bin` or `~/.local/bin`, `NO_COLOR`
aware). README rewritten to the binary flow (+ `go install …@latest` fallback). Verified via a
`--snapshot` build plus a throwaway `v0.0.1-test` tag that ran the workflow green and produced a draft
with all assets (tag + draft since deleted). Key commit `50c59aa`. Deferred post-v0.1: Homebrew tap,
npm wrapper, Scoop/Winget. Follow-up: the full public `curl|sh` path proves out at the real `v0.1.0`.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** Zero-runtime distribution for the Go binary. **GoReleaser** (`.goreleaser.yaml`,
v2) cross-compiles a **5-target matrix** (linux amd64/arm64, darwin amd64/arm64, windows amd64;
windows/arm64 ignored) as `-trimpath` + `CGO_ENABLED=0` static builds with `main.version` injected
from the tag; `tar.gz` archives (`.zip` on Windows) named `specflow_<version>_<os>_<arch>` bundle
LICENSE + README, alongside a `sha256` `checksums.txt`. A **`Release` workflow**
(`.github/workflows/release.yml`) fires on any `v*` tag → `goreleaser release --clean` with the
auto `GITHUB_TOKEN` (`contents: write`) → a **draft** release (`prerelease: auto`; review then
publish from the UI). An **`install.sh`** `curl … | sh` front-end detects OS/arch, resolves the
latest published release (or `SPECFLOW_VERSION`), downloads + checksum-verifies + extracts the
matching archive, installs to `/usr/local/bin` (sudo) or `~/.local/bin`, and honors `NO_COLOR`/
non-TTY. README install section rewritten to the binary flow (+ `go install …@latest` fallback).

**Verification.** `goreleaser release --snapshot` produced all 5 archives + `checksums.txt` with the
exact names `install.sh` expects; install.sh's checksum-grep + extract path was replicated locally
(binary runs, injected version correct). A pushed throwaway `v0.0.1-test` tag ran the workflow
**green** ([run 28645497812](https://github.com/MatanKoby/specflow/actions/runs/28645497812)) and
produced a draft release with all assets — draft correctly hidden from the public API, its assets not
anonymously downloadable. Test tag + draft deleted afterward.

**Deferred (post-v0.1).** Homebrew tap (a `brews:` block slots into `.goreleaser.yaml` when the tap
repo lands), npm wrapper (`npx specflow`), Scoop/Winget — see `open-questions.md` → Distribution. The
full public `curl … | sh` path is only exercisable against a *published* release, so it stays
unproven until the real `v0.1.0`.

## Batch SO — Spec-only install mode (`--spec-only`)
Shipped `specflow init --spec-only`: a lighter install that keeps only the spec discipline
(`AGENTS.md` spec sections + `spec/` + the `spec-edit` procedure/skill + stamp + agent stubs) and
omits `BUILD_QUEUE.md`, `CLAIMS.md`, the `claim-batch`/`finish-batch` procedures + skills, and the
`specflow/history/` archives. Implemented as **section composition** (one source, no forks):
`AGENTS.md` and `spec-edit.md` carry `specflow:full-only`/`spec-only` marker pairs and `renderBody`
keeps/drops them per mode (own-line vs inline markers handled separately so tables/spacing stay
intact); the baseline hash is taken over the *rendered* region. Mode is recorded in `config.mode`
and threaded through `init`/`upgrade`/`verify`. Key commit `93519b0`. Follow-ups deferred: the
spec-only→full graduation path, and section-composing the per-agent stubs (a spec-only Claude install
still ships a `CLAUDE.md` pointer that mentions queue/claim — cosmetic).

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** `specflow init --spec-only` — a lighter install that keeps only the **spec
discipline** and omits the batch/claim machinery. It installs `AGENTS.md` (spec sections), `spec/`,
the **spec-edit** procedure + skill, the stamp (mode recorded), and the selected agent stubs; it
omits `BUILD_QUEUE.md`, `CLAIMS.md`, the `claim-batch`/`finish-batch` procedures, those two skills,
and the `specflow/history/` archives. **Section composition (one source, no forks):** `AGENTS.md`
and `spec-edit.md` now carry `specflow:full-only` / `specflow:spec-only` marker pairs *inside* their
managed region; `renderBody` keeps the pair matching the install mode (stripping its markers) and
drops the other pair whole. Own-line markers (paragraphs, list items, table rows) and inline clause
markers are handled separately so a drop never splits a markdown table or leaves a stray blank line;
runs of 3+ newlines collapse back to one blank line. The baseline hash is taken over the **rendered**
region, so drift detection works per-mode. The tag token (`specflow:full-only:…`) never collides with
the region token (`specflow:start`/`end`). **Mode plumbing:** `{{MODE}}` placeholder in the config
template → `config.mode`; `mode` threaded through `PlanInit`/`ApplyInit`/`classifyInit`/`initFiles`/
`managedEntries`/`computeManaged`/`recordManaged`/`fillStamp`; `upgrade` and `verify` read the mode
from the stamp, re-render the managed region for it, and filter the managed/placed-file sets via
`specOnlyOmits`. CLI: `--spec-only` flag, mode-aware Phase-2 / review-handoff text, and updated
`init --help` + top-level usage. **Verification:** `go test ./...` green (six new spec-only tests:
omissions + mode stamp, batch-free managed files with no leftover tags, full-mode completeness, clean
spec-only `upgrade`, spec-only `verify`); `gofmt`/`go vet` clean. Self-hosted: ran `upgrade` on this
(full-mode) repo — root `AGENTS.md` refreshed to the composed wording, `spec-edit.md` byte-identical,
`verify` clean, second `upgrade` a no-op. **Follow-ups deferred (per architecture, in scope for later):**
the graduation path spec-only → full (`enable-batching` / re-run); and the per-agent stubs + the
`spec-edit` skill description are **not** section-composed yet, so a spec-only Claude install still
ships a `CLAUDE.md` whose pointer mentions queue/claim + the two uninstalled skills (cosmetic — the
authoritative composed `AGENTS.md` is correct).

## Batch BI — Brownfield-aware `init` (inject-with-consent) + `specflow verify`
Rebuilt `init` into a non-destructive, two-phase, consent-gated flow: Phase 1 injects specflow's
marker-wrapped region into target files that already exist (content preserved), Phase 2 creates the
owned files; `init` tracks its own created/modified/declined list, ends with a "review `git diff`,
then commit" handoff, and **never commits** (non-interactive proceeds and points at `git status`).
Brought all five per-agent instruction files under management (marker-wrapped, refreshed by `upgrade`
for installed agents only) with **idempotent injection** (refresh a region; skip "already wired" when
a file already references `AGENTS.md`) and **tier-aware** decline/missing notices. Added
per-subcommand `--help` and a basic **`specflow verify`** install-integrity check (`verify --batch`
stubbed until Batch E). Relocated the `_DONE` archives to `specflow/history/` and updated every path
reference. Key commits `b766da2` (two-phase init), `b57bd50` (idempotent injection + tier-aware
decline), `f583f39` (`verify`), `6d14a90` (`_DONE` relocation). Follow-up: Batch SO (spec-only)
shares the template/section + marker work.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** `init` is now brownfield-aware and non-destructive, in two consent-gated phases:
Phase 1, for each target file that **already exists**, injects specflow's marker-wrapped region at
the top (existing content preserved); Phase 2 explains, then creates, the specflow-owned files.
`init` tracks its **own** created/modified/declined list and ends with a "review `git diff`, verify
nothing was damaged, then commit" handoff — it **never commits**; non-interactive runs proceed and
point at `git status` (no `--yes` flag). All five per-agent instruction files (`CLAUDE.md`, copilot,
cursor, bob, antigravity) are now **marker-wrapped and managed**, so `upgrade` refreshes their region
— but only for installed agents. **Idempotent injection:** refresh an existing region, and skip with
an "already wired" note when a per-agent file already references `AGENTS.md` (`AGENTS.md` itself
excepted — it must *carry* the protocol). **Tier-aware notices** on declined/missing pieces (Tier 1
`AGENTS.md`/procedures → can't work properly; Tier 3 per-agent → that agent isn't auto-wired, works
once its file points at `AGENTS.md`). Added **per-subcommand help** (`init`/`upgrade`/`verify --help`)
and a basic **`specflow verify`** install-integrity check (config valid, Tier-1 present with intact
regions + drift warnings, Tier-3 present/wired; exits non-zero on a Tier-1 problem; `verify --batch`
stubbed until Batch E). **Relocated the `_DONE` archives** to `specflow/history/` and updated every
path reference (`AGENTS.md`, procedures, queue/claims headers, the finish-batch skill, README,
tests); templates moved in lockstep so fresh installs place them there. Two bugs fixed along the way:
a shared package-level stdin reader (the per-prompt `bufio.NewReader` dropped buffered input across
prompts), and a guard so `upgrade` never adopts/overwrites an untouched brownfield file (region
absent **and** no recorded baseline). Executed as internal `batch-BI:` chunks — key commits
`ce60443` (per-agent files managed), `b766da2` (two-phase init), `b57bd50` (idempotent injection +
tier-aware decline), `ec31a2d` (per-subcommand help), `f583f39` (`verify`), `6d14a90` (`_DONE`
relocation). ~22 tests pass; `gofmt`/`go vet` clean; self-host `upgrade` idempotent; brownfield-init
/ verify / fresh-init smokes pass. **Follow-up:** Batch SO (spec-only mode) is next and shares the
template/section + marker work.

## Batch CFG — Config file, commit/push levers & safety fixes (v0.1 foundation)
Renamed the stamp to `specflow/config.json` (out of the repo root) with a **`config` block**
(`agents`/`mode`/`commit`/`push`) beside internal state (versions, schema, `managed` hashes). Taught
the procedures the two commit/push levers: `AGENTS.md` defines `config.commit` / `config.push`
(`agent|user`) authoritatively, and `claim-batch.md` / `finish-batch.md` carry a lever-note (when
`commit: user`, alert + supply a suggested message instead of committing; when `push: user`, commit
but don't push); default `agent`/`agent`. Safety: a managed file with **no baseline** is treated as
drift (`.specflow-new` sidecar, never overwrite — risk A); a corrupt `config.json` fails friendly;
`init` **requires git** (refuses otherwise). CLI: no ANSI when stdout isn't a TTY or `NO_COLOR` set.
Key commits `48ac469` (safety/CLI), `ea3bffb` (config.json + block), `5204656` (lever wording).
Follow-ups: the init-time lever prompt + brownfield two-phase flow → Batch BI; `_DONE` archives move
to `specflow/history/` in BI.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** The v0.1 foundation the other batches read. The stamp is renamed to
`specflow/config.json` (out of the repo root) and now carries a **`config` block** —
`agents` / `mode` / `commit` / `push` — alongside internal state (versions, schema, `managed`
hashes); all references + tests updated. The two **commit/push levers** are honored by the
procedures: `AGENTS.md` gains an authoritative *Commit & push authority* section defining
`config.commit` / `config.push` (`agent|user`), and `claim-batch.md` / `finish-batch.md` each carry
a compact lever-note — when `commit: user` the agent alerts at commit points and supplies a suggested
message rather than committing; when `push: user` it commits but never pushes; default stays
`agent` / `agent`. (The *interactive* lever choice at `init` is deferred to Batch BI's init overhaul;
CFG lands the field + the procedure behavior with agent/agent defaults.) **Safety fixes:** a managed
file with no recorded baseline is now treated as **drift** — `upgrade` writes a `.specflow-new`
sidecar and never overwrites (closes risk A); a corrupt/hand-edited `config.json` fails with a
friendly "fix or restore it" message; `init` now **requires a git repo** and refuses otherwise
(nothing written), closing the no-undo risk on brownfield repos. **CLI hygiene:** colour output is
suppressed when stdout isn't a TTY or `NO_COLOR` is set (no escape codes in piped/redirected output).
10 tests pass; `go vet` clean; self-host `upgrade` is a clean idempotent no-op. **Follow-ups:** the
init-time lever prompt + brownfield two-phase flow land in Batch BI; the `_DONE` archives move under
`specflow/history/` in BI.

## Batch G1 — Port the CLI to Go (full replace)
Replaced the Node `bin/specflow.js` with a single statically-compiled **Go binary** (`cmd/specflow`
+ `internal/kit`), templates embedded via `//go:embed all:templates` (the `all:` prefix keeps the
dotfiles). Ported the smoke suite to `go test` (builds + drives the real binary against temp repos).
Verified: `init` output **byte-identical** to the Node CLI, stamp semantically identical (managed
hashes match), `upgrade` a clean idempotent no-op, and self-hosted by running the Go `upgrade` on
this repo. Node files removed (`bin/specflow.js`, `package.json`, `test/smoke.js`); CI switched to
`go vet`/`build`/`test` on ubuntu + macos; README + architecture file-map updated; pending batches
retargeted to Go. Key commit `422284b` (Go CLI), final `ac82cec`. Follow-up: Batch G2 ships it
(GoReleaser → GitHub Releases + curl|sh + Homebrew).

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** The CLI is now a single statically-compiled **Go binary** (`cmd/specflow` +
`internal/kit`), replacing the Node `bin/specflow.js` at functional parity (`init`, `upgrade`,
`--version`, `--help`, unknown-command exit, non-git warning). Templates are embedded via
`//go:embed all:templates` (the `all:` prefix keeps the dotfiles — `.claude/`, `.cursor/`, etc.).
The smoke suite was ported to `go test`, which builds and drives the real binary against temp repos.
Verified parity three ways: `init` output is **byte-identical** to the Node CLI, the stamp is
semantically identical (managed SHA-256 hashes match), and `upgrade` is a clean idempotent no-op;
self-hosted by running the Go `upgrade` on this repo (regions untouched, only the stamp bumped).
Node files removed (`bin/specflow.js`, `package.json`, `test/smoke.js`); CI switched to
`go vet`/`build`/`test` on ubuntu + macos; `.gitignore` drops node_modules, adds `/dist/`; README +
`architecture.md` file-map updated to the Go layout; pending batches (1/2/3/5/W/NB) retargeted from
Node to Go; the obsolete Node-version-guard open question dropped. **Follow-up:** Batch G2 — the
GoReleaser → GitHub Releases pipeline with `curl|sh` + Homebrew front-ends — ships the binary.

<!-- Recent finishes, newest first. Bounded working set: the 5 newest stay here and
     prune-ledgers moves the rest to specflow/history/CLAIMS_DONE.md. -->

## Batch U2 — Self-documenting, edit-resistant region markers
Baked a "managed by specflow; do not edit inside" note into the `specflow:start` marker (an HTML
comment — invisible in rendered markdown, visible in the raw file). Made marker matching token-based
(regex on `specflow:start`/`specflow:end`) so the note can evolve without breaking parsing or forcing
a migration; a clean `upgrade` canonicalizes a file's markers to the template's wording. Key commit
`66442e0`. Smoke suite at 19 checks; applied to the repo's own files (self-hosting). No follow-ups.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** The `specflow:start` marker now carries an inline note — `managed by specflow; do
not edit inside these markers (your edits block specflow upgrade). Add your own notes outside them.`
— so a human editing the raw file is warned (the note is an HTML comment, invisible in rendered
markdown). To keep that safe, `extractRegion` matches markers by their `specflow:start`/`specflow:end`
**token** (regex `START_RE`/`END_RE`) rather than an exact string, and returns the matched marker
text; the clean-path rewrite re-applies the *template's* marker wording, so the note can evolve
without breaking parsing or forcing a migration. Smoke suite at 19 checks (added: marker-wording
change is canonicalized, outside text preserved, no backup). Self-hosting check: ran `upgrade` on the
repo root — only the marker line changed in each managed file. No follow-ups.

## Batch U — Non-destructive upgrade redesign
Made `upgrade` non-destructive: managed files (`AGENTS.md` + procedures) wrap their generated content
in `<!-- specflow:start/end -->` markers, `init` records a SHA-256 of each region in the stamp's
`managed` map, and `upgrade` refreshes only the region — preserving everything outside, leaving a
hand-edited (drifted) region untouched with a `.specflow-new` sidecar, and migrating pre-marker
installs via a `.specflow-bak` backup. Key commit `42cd047`. Smoke suite at 18 checks; applied to
specflow's own root files (self-hosting). No follow-ups specific to U.

---

*Relocated from `specflow/history/CLAIMS_DONE.md` by `specflow migrate-claims`.*

**What shipped.** `upgrade` no longer wholesale-overwrites managed files. Each managed file
(`AGENTS.md` + the three procedures) wraps its generated content in `<!-- specflow:start -->` /
`<!-- specflow:end -->` markers; `init` records a SHA-256 of each region in the stamp's new
`managed` map. On `upgrade`: a clean region (hash matches baseline) has only its between-markers
content replaced (everything outside preserved verbatim); a drifted region (hash differs) is left
untouched, with the fresh version dropped to a `<file>.specflow-new` sidecar and reported; a
pre-marker file is migrated (backed up to `<file>.specflow-bak`, then rewritten with markers).
Implemented in `bin/specflow.js`; markers added to `templates/base/AGENTS.md` + procedures; 18-check
smoke suite green (outside-text-survives, drift-not-clobbered, pre-marker-migration). Self-hosting:
specflow's own root `AGENTS.md` + procedures migrated to the format, stamp now carries `managed`.
Spec updated (`architecture.md`, `open-questions.md`). **Follow-ups deferred:** none specific to U;
`--dry-run` (Batch 5) and `status`/drift-flag (Batch 2) build naturally on this.
