# Roadmap research: better specs, spec maintenance, backfill (2026-07)

> A dated research snapshot (see `README.md`). It records the survey and reasoning that feed the
> roadmap; it is not ratified design. The only conclusion **adopted** so far is the research-flow
> mechanism this note is filed under. The other candidate directions below await ratification before
> they graduate into `open-questions.md` / `roadmap.md`.

## Purpose

Survey libraries and apps that help create better specs, maintain specs, and backfill specs from
existing code; identify what they solve that specflow does not yet; decide what specflow can adopt
(and how), given specflow's constraints: a single static Go binary, zero runtime deps, output is
markdown + git, and it runs *inside* an agent that already has file/grep/LLM tools.

## Part 1: the landscape (name + one line)

### A. Spec-driven dev frameworks for AI agents (specflow's neighborhood)
- GitHub Spec Kit: CLI (`specify`) running a phased flow (constitution, specify, clarify, plan, tasks, implement), portable across ~28 agents.
- Amazon Kiro: agentic IDE that forces intent into requirements.md (EARS notation), design.md, tasks.md before code.
- OpenSpec: open-source framework enforcing a strict proposal -> apply -> archive state machine before code.
- BMAD-METHOD: role-based agent team (analyst/PM/architect/SM/dev) producing a PRD + architecture, sharded into stories.
- Tessl: AI-native, spec-centric platform + a Spec Registry; the bet that specs regenerate code.
- Augment Cosmos: treats specs as operational infrastructure coordinating agents across the SDLC.
- Task Master (claude-task-master): parses a PRD into a structured tasks.json, manages breakdown/expansion.
- Agent OS (Builder Methods): layers standards/ + product/ + specs/ onto any agent.
- Traycer: a planning layer that drafts a reviewable plan/spec before the agent writes code.
- PRPs (Product Requirement Prompts): context-engineering method bundling spec + context + validation into one runnable prompt.
- SpecStory: captures AI coding chat sessions and derives durable rules/context/specs from them.

### B. Codebase -> wiki/docs auto-generation (backfill neighbors)
- DeepWiki (Cognition/Devin): turns any GitHub repo into an interactive wiki with diagrams + Q&A.
- CodeWiki (open-source, FSoft-AI4Code): open framework for repo-level structured docs across multilingual codebases.
- deepwiki-rs / Code2Tutorial / AIGNE DocSmith / GitDocs AI: repo -> structured docs/tutorials/README, several Git-integrated to stay fresh.
- Swimm: docs coupled to source via snippets; auto-flags and helps fix docs when code drifts.
- Mintlify: AI docs platform that generates and maintains docs from the codebase.
- Komment.ai: automated in-code documentation at scale.
- Driver.ai / Depth AI / Unblocked / Greptile: codebase knowledge graph to explain architecture and answer questions.
- Sourcegraph (+ Cody): code search + AI comprehension across large codebases.
- CodeSee: auto-generated codebase maps and dependency diagrams.
- Doxygen / Sphinx / JSDoc / TypeDoc: classic API-reference generation from code + doc-comments (the pre-AI baseline).
- Backstage TechDocs (Spotify): docs-as-code in a developer portal, tied to the service catalog.

### C. Requirements / PRD authoring (better spec, upstream)
- ChatPRD: AI copilot that interviews a PM and drafts and critiques PRDs.
- Notion AI / Productboard / Chisel: PRD templates + AI assist inside product tools.
- EARS (Easy Approach to Requirements Syntax): constrained notation for unambiguous, testable requirements (what Kiro emits).
- arc42 / C4 model: standardized templates for architecture documentation.

### D. Formal & executable specifications (rigor / testable spec)
- Gherkin / Cucumber: Given/When/Then acceptance specs that are executable tests.
- Specification by Example (Gojko Adzic): living documentation derived from executable examples.
- OpenAPI/Swagger, AsyncAPI, JSON Schema: machine-readable contracts for APIs and data shapes.
- Pact: consumer-driven contract testing between services.
- TLA+ / Alloy: formal specification languages for verifying system design/behavior.
- Design by Contract (Eiffel): preconditions/postconditions/invariants as spec embedded in code.

### E. Decisions, memory & context (keep intent durable)
- adr-tools / MADR / Log4brains: Architecture Decision Records; capture WHY a decision was made, versioned in-repo.
- Cline / Roo Code Memory Bank: markdown files that persist project intent/context across agent sessions.
- Cursor Rules / Windsurf Rules / AGENTS.md / CLAUDE.md: repo-scoped instruction files that steer agents.
- Repomix / Gitingest: pack a repo into one prompt-ready context blob.
- Aider repo map: auto-built tree-sitter map of the repo for agent context.
- Dendron / Foam / Obsidian: markdown-first knowledge bases for engineering notes.

### F. Import from existing docs (already on specflow's roadmap as "remote spec sources")
- Confluence / Notion / GitBook: where teams already keep design; import targets for backfill.

## Part 2: gap analysis (what they solve that specflow does not yet)

Numbering matches the working discussion. Flags note where a "gap" conflicts with specflow's
deliberate philosophy (token-frugal, intent-over-mechanics, human-owns-design, non-crawling,
honor-system).

- #1 No spec template or quality rubric. Adopt EARS (requirement lines) + arc42/C4 (architecture skeleton). Cheap; pure upside.
- #2 No explicit clarify/critique phase. Principle exists (`workflow.md`: the spec phase is a dialogue + gate-1); the mechanism (a structured clarify checklist) does not. Home: Batch NB planning phase.
- #3 No spec lint. Splits: structural lint (broken cross-refs, size rule, orphan files, dead links) is cheap, deterministic, no-LLM, and belongs next to `specflow verify`; semantic lint (ambiguous/testable/contradictory) is LLM-expensive and should be opt-in.
- #4 No spec<->code traceability. Cheapest: harvest what batches already declare (files + spec ref) and persist the join on finish. Richer: file/dir-level inline anchors (NOT line-level; that is Swimm's brittleness).
- #5 No drift detection. Extend the SHA-hashing specflow already ships: record (spec-section-hash, linked-code-files, code-commit-SHA) at finish; detect via pure git (did linked code change since that commit with no later `spec:` commit on the section?). Catches edits made without specflow because git is the witness. Mark-stale is the cheap deterministic default; auto-update cannot be silent (gate-1 forbids), so it becomes opt-in "draft the fix, user ratifies" = the same machinery as backfill.
- #6 No spec coverage view. Falls out of the #4/#5 index for near-zero marginal cost.
- #7 No spec Q&A. DO NOT BUILD. Specs are small, concern-per-file, under 600 lines: they fit in context, so asking the invoking agent is already free. Embeddings / DeepWiki-style Q&A only pays off at a scale specflow avoids.
- #8 Backfill manual and non-crawling (intentional). Adapt, not copy: an opt-in agent-generated skeleton from #1 templates that the user ratifies, reusing the drift draft-and-ratify flow.
- #9 No repo-packing primitive. Largely a NON-NEED: the invoking agent already reads files and greps; Repomix/Gitingest exist to feed raw LLM pipelines that lack repo access, which specflow does not have.
- #10 No visual artifact. Clean import = C4 + Mermaid (text diagrams that render natively in GitHub markdown, zero runtime dep).
- #11 Doc import (Confluence/Notion/GitBook). Already on the roadmap as "remote spec sources"; worth prioritizing (brownfield intent often lives in a wiki).
- #12 CORRECTED / DROPPED. Not a gap: specflow already decomposes spec -> batches (`spec-edit.md` step 2 "add new batches that flow from the decision"; Batch NB is the quick spec-and-queue flow). It already has `Depends on:` + files-overlap parallelism.
- #13 No executable/acceptance-criteria linkage. Lightweight version: batches carry checkable acceptance criteria (Gherkin-shape), not full Cucumber. Needs thought.
- #14 ADRs as a first-class spec type. Adopt the MADR markdown template as a spec file type (e.g. spec/decisions/ADR-NNN.md). Near-zero cost, no dependency.

## Part 3: import vs copy (the decisive constraint)

specflow is a single static Go binary, zero runtime deps, output is markdown + git, running inside an
agent that already has file/grep/LLM tools. That reshapes "import vs copy" into three buckets:

- **Formats / notations / templates (just text): IMPORT THESE.** MADR (#14), EARS (#1), C4 + Mermaid (#10), arc42 (#1), Gherkin shape (#13). Highest ROI. Text you vendor and pin yourself: no runtime, no breaking changes ever, honors the single-binary promise. This directly answers the "pin a specific version to avoid breaking changes" concern: formats cannot break you.
- **Go libraries (link + pin in go.mod): vendor if we build the traceability spine.** goldmark (markdown AST -> structural lint #3a, cross-ref + anchor parsing for #4/#5); optionally go-tree-sitter (symbol-level repo map for backfill). Compiles into the binary, keeps zero runtime deps, version-locked = no surprise breakage.
- **Runtime tools (Node/Python services): DO NOT IMPORT.** Repomix, Gitingest, CodeWiki, DeepWiki-OSS, Task Master, Log4brains. Shelling out reintroduces a runtime (kills "install into any repo, no friction") and inherits exactly the breaking-changes/scope-drift to be avoided; and the invoking agent already covers what they do.

Crisp answer to "can we use any of these instead of copying": the FORMATS yes (MADR, EARS,
C4/Mermaid, Gherkin); the TOOLS no.

## Part 4: which path each improves most

- **Spec generation** (cheap, mostly format imports): #1 templates + #2 clarify pass in NB.
- **Spec maintenance** (the differentiated build): the TRACEABILITY SPINE = #4 link -> #5 drift -> #6 coverage, built on existing SHA-hashing infra; + #3a structural lint + #14 ADRs. This is the one real build and the most defensible ("a source-of-truth tool that cannot tell when the truth went stale is missing its point").
- **Backfill** (no library; reuse machinery): an agent-generated skeleton from #1 templates, ratified via the drift draft-and-ratify flow; + #11 doc-import. #9 repo-packing is a non-need.
- **spec->batch->queue** (already agent-authored): the trace index CLOSES THE LOOP. Persisting the batch -> (spec section, files) join on finish makes the queue auditable (which spec each batch served, which spec is un-built, which shipped code has no spec). + #2 better batches from a clarify pass + lightweight #13 acceptance criteria for a checkable definition-of-done. Same index as maintenance: build once, both paths benefit.

## Part 5: recommendations (candidates, awaiting ratification)

- ONE build worth doing: the spec<->code<->batch INDEX on top of existing SHA-hashing -> unlocks drift (#5), coverage (#6), traceability (#4), and the queue audit loop at once. Deterministic, git-native, cheap; the expensive LLM part (drafting the fix) is opt-in and ratified, consistent with the gates.
- FOUR cheap format imports: MADR (ADRs), EARS (requirements), C4+Mermaid (diagrams), Gherkin-shape (acceptance criteria). All pin-proof.
- ONE explicit non-build: spec Q&A/embeddings (#7). The agent already does it.
- Runtime tools: skip. They break the single-binary + no-runtime promise and reintroduce version churn.

## Status / next steps

- **Adopted:** the lightweight research-flow mechanism (this folder + `workflow.md` -> *Research notes*). Batch RF ships it to new installs.
- **Pending user ratification (still candidates):** graduate the traceability spine, the format imports, the clarify-pass (into Batch NB), and ADRs into `open-questions.md` / `roadmap.md` when the user picks them.

## Sources
- MarkTechPost, "9 Best AI Tools for Spec-Driven Development in 2026" (2026-05-08).
- Augment Code, "6 Best Spec-Driven Development Tools".
- arXiv 2510.24428, "CodeWiki: Automated Repository-Level Documentation".
- eliteai.tools, "DeepWiki alternatives".
