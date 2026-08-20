// Package kit implements specflow's scaffold (init) and non-destructive refresh (upgrade).
// It operates on a target directory using a templates filesystem (embedded by the caller), so the
// same logic is exercised by tests without depending on a real install.
package kit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MANAGED lists the base files/dirs whose specflow-managed *region* upgrade refreshes. Each managed
// file wraps its generated content in START…END markers; upgrade replaces only what's between them
// and preserves everything outside. The per-agent instruction files (see agentInstructionFile) are
// also managed, but only for the agents actually installed. Everything else (queue, claims, spec)
// is written once at init and never touched.
var MANAGED = []string{"AGENTS.md", "specflow/procedures"}

// agentInstructionFile maps an agent key to the repo-relative instruction file specflow manages for
// that agent. Each carries a marker-wrapped region that `upgrade` refreshes — but only for installed
// agents, so an uninstalled agent's file is never created or touched. The template source lives at
// agents/<key>/<path>.
var agentInstructionFile = map[string]string{
	"claude":      "CLAUDE.md",
	"cursor":      ".cursor/rules/specflow.mdc",
	"copilot":     ".github/copilot-instructions.md",
	"bob":         ".bob/rules/specflow.md",
	"antigravity": ".agents/rules/specflow.md",
}

// Markers are matched by their specflow:start / specflow:end token, not an exact string, so the
// human-readable note inside a marker can evolve without breaking parsing or forcing a migration.
var (
	startRe = regexp.MustCompile(`(?s)<!--\s*specflow:start\b.*?-->`)
	endRe   = regexp.MustCompile(`(?s)<!--\s*specflow:end\b.*?-->`)
)

// Section-composition tags. A managed file's region can carry sub-sections wrapped in
// specflow:full-only / specflow:spec-only marker pairs; render keeps the pair that matches the
// install mode (stripping its markers) and drops the other pair entirely (markers + content). This
// is how one source (AGENTS.md, spec-edit.md) serves both modes without forked variants. The tag
// token (specflow:full-only:…) never collides with the region token (specflow:start/end).
//
// A tag pair is authored either on its own lines (the common case — wrapping whole paragraphs,
// list items, or table rows) or inline within a line (wrapping a clause). The two are handled
// separately so dropping a block leaves adjacent content contiguous (no table-splitting blank
// line) while an inline drop removes only the clause, not the surrounding line breaks.
type composeRe struct {
	wholeBlock  *regexp.Regexp // a start…end pair on their own lines → match incl. both marker lines
	wholeMarker *regexp.Regexp // a lone marker on its own line → match incl. its newline
	inlineBlock *regexp.Regexp // a start…end pair within text → match markers + content, no newlines
	inlineMark  *regexp.Regexp // a lone marker token
}

func newComposeRe(tag string) composeRe {
	return composeRe{
		wholeBlock:  regexp.MustCompile(`(?ms)^[ \t]*<!--\s*specflow:` + tag + `:start\b.*?-->[ \t]*\n.*?^[ \t]*<!--\s*specflow:` + tag + `:end\b.*?-->[ \t]*\n`),
		wholeMarker: regexp.MustCompile(`(?m)^[ \t]*<!--\s*specflow:` + tag + `:(?:start|end)\b.*?-->[ \t]*\n`),
		inlineBlock: regexp.MustCompile(`(?s)<!--\s*specflow:` + tag + `:start\b.*?-->.*?<!--\s*specflow:` + tag + `:end\b.*?-->`),
		inlineMark:  regexp.MustCompile(`<!--\s*specflow:` + tag + `:(?:start|end)\b.*?-->`),
	}
}

var (
	fullOnly   = newComposeRe("full-only")
	specOnly   = newComposeRe("spec-only")
	blankRunRe = regexp.MustCompile(`\n{3,}`)
)

// renderBody composes a managed file's content for the given mode: the non-matching tag's blocks
// are removed whole, the matching tag's markers are stripped (content kept), and the runs of blank
// lines left behind are collapsed back to a single blank line. A file with no composition tags is
// returned byte-for-byte unchanged (the common case), so this is safe to run over every template.
func renderBody(s, mode string) string {
	if !strings.Contains(s, "specflow:full-only:") && !strings.Contains(s, "specflow:spec-only:") {
		return s
	}
	drop, keep := fullOnly, specOnly // spec-only mode: drop full-only blocks, keep spec-only
	if mode != "spec-only" {
		drop, keep = specOnly, fullOnly
	}
	s = drop.wholeBlock.ReplaceAllString(s, "")
	s = drop.inlineBlock.ReplaceAllString(s, "")
	s = keep.wholeMarker.ReplaceAllString(s, "")
	s = keep.inlineMark.ReplaceAllString(s, "")
	return blankRunRe.ReplaceAllString(s, "\n\n")
}

// renderFile applies renderBody to raw template bytes for the given mode.
func renderFile(content []byte, mode string) []byte { return []byte(renderBody(string(content), mode)) }

// specOnlyOmits reports whether a repo-relative file belongs to the batch/claim machinery that
// spec-only mode leaves out: the queue, the claims ledger, their history archives, the
// claim/finish/prune procedures, their skills, and the finish-batch handoff hook (for any agent).
// Everything else — AGENTS.md, spec/, the spec-edit procedure + skill, the stamp, agent instruction
// files — installs in both modes.
//
// prune-ledgers is full-only for the same reason claim/finish are: it operates on BUILD_QUEUE.md and
// CLAIMS.md, neither of which a spec-only install has.
func specOnlyOmits(rel string) bool {
	switch rel {
	case "BUILD_QUEUE.md", "CLAIMS.md",
		"specflow/procedures/claim-batch.md", "specflow/procedures/finish-batch.md",
		"specflow/procedures/prune-ledgers.md":
		return true
	}
	if strings.HasPrefix(rel, "specflow/history/") {
		return true
	}
	// The handoff-reminder hook backstops finish-batch step 6, which spec-only doesn't have.
	if strings.Contains(rel, ".claude/hooks/specflow-handoff-reminder.sh") {
		return true
	}
	return strings.Contains(rel, "skills/claim-batch/") ||
		strings.Contains(rel, "skills/finish-batch/") ||
		strings.Contains(rel, "skills/prune-ledgers/")
}

// QueueTokens are the queue/claim identifiers that only exist in a full install. A spec-only repo
// has none of the files or skills they name, so a generated file mentioning one is pointing the
// agent at machinery that isn't there. Shared by Verify's mode-consistency check and the
// composition tests so the two can't drift apart.
var QueueTokens = []string{"BUILD_QUEUE", "CLAIMS.md", "claim-batch", "finish-batch", "prune-ledgers"}

// ModeLeaks returns the QueueTokens present in content that the given install mode omits. Full mode
// omits nothing, so it always returns nil.
//
// This exists because baseline hashes structurally cannot catch a mode mismatch: the baseline is
// taken over the *rendered* region, so a full-mode paragraph wrongly shipped into a spec-only
// install still matches its own recorded hash and reports as clean. Hashes prove a region is
// unmodified; only this proves it is mode-appropriate.
func ModeLeaks(mode, content string) []string {
	if mode != "spec-only" {
		return nil
	}
	var found []string
	for _, tok := range QueueTokens {
		if strings.Contains(content, tok) {
			found = append(found, tok)
		}
	}
	return found
}

// modeOf reads the install mode from the stamp's config block, defaulting to "full".
func modeOf(stamp map[string]any) string {
	if cfg, ok := stamp["config"].(map[string]any); ok {
		if m, _ := cfg["mode"].(string); m != "" {
			return m
		}
	}
	return "full"
}

type regionParts struct {
	before, startMarker, region, endMarker, after string
}

// extractRegion splits a managed file around its single specflow region. ok is false when the
// markers are absent or malformed (a pre-marker install, or hand-mangled). startMarker/endMarker
// carry the matched marker text verbatim so the template's wording can be re-applied on a refresh.
func extractRegion(content string) (regionParts, bool) {
	sm := startRe.FindStringIndex(content)
	em := endRe.FindStringIndex(content)
	if sm == nil || em == nil {
		return regionParts{}, false
	}
	if em[0] < sm[1] {
		return regionParts{}, false
	}
	return regionParts{
		before:      content[:sm[0]],
		startMarker: content[sm[0]:sm[1]],
		region:      content[sm[1]:em[0]],
		endMarker:   content[em[0]:em[1]],
		after:       content[em[1]:],
	}, true
}

func hashRegion(region string) string {
	sum := sha256.Sum256([]byte(region))
	return hex.EncodeToString(sum[:])
}

// hasRegionMarkers reports whether content already carries a specflow region.
func hasRegionMarkers(content string) bool {
	_, ok := extractRegion(content)
	return ok
}

// referencesAgents reports whether a file already points at AGENTS.md (a markdown link, a plain
// mention, or an `@AGENTS.md` import) — the cheap heuristic for "already wired to the single source".
func referencesAgents(content string) bool {
	return strings.Contains(content, "AGENTS.md")
}

func today() string { return time.Now().UTC().Format("2006-01-02") }

func configPath(targetDir string) string {
	return filepath.Join(targetDir, "specflow", "config.json")
}

func destPath(targetDir, rel string) string {
	return filepath.Join(targetDir, filepath.FromSlash(rel))
}

// managedEntry pairs a managed file's repo-relative dest path with its template source path.
type managedEntry struct {
	rel string // repo-relative dest (e.g. "AGENTS.md", "CLAUDE.md")
	src string // template path within tpl (e.g. "base/AGENTS.md", "agents/claude/CLAUDE.md")
}

// managedEntries expands the base-managed set plus the per-agent instruction files for the given
// installed agents into concrete (dest, template-source) pairs — the authoritative managed set for
// a repo with those agents. In spec-only mode the batch/claim procedures are dropped (the queue and
// claims machinery isn't installed), so they're never part of the managed set.
func managedEntries(tpl fs.FS, agentKeys []string, mode string) ([]managedEntry, error) {
	var out []managedEntry
	add := func(rel, src string) {
		if mode == "spec-only" && specOnlyOmits(rel) {
			return
		}
		out = append(out, managedEntry{rel: rel, src: src})
	}
	for _, rel := range MANAGED {
		src := "base/" + rel
		info, err := fs.Stat(tpl, src)
		if err != nil {
			continue
		}
		if info.IsDir() {
			err := fs.WalkDir(tpl, src, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					add(strings.TrimPrefix(p, "base/"), p)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			add(rel, src)
		}
	}
	for _, key := range agentKeys {
		rel, ok := agentInstructionFile[key]
		if !ok {
			continue
		}
		src := "agents/" + key + "/" + rel
		if _, err := fs.Stat(tpl, src); err != nil {
			continue
		}
		add(rel, src)
	}
	return out, nil
}

// baselineMap pulls the managed-region baseline hashes out of the stamp.
func baselineMap(stamp map[string]any) map[string]string {
	out := map[string]string{}
	if raw, ok := stamp["managed"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

// isPerAgentFile reports whether a managed rel is a per-agent instruction file (Tier 3) rather than
// a base mechanism file (Tier 1).
func isPerAgentFile(rel string) bool {
	for _, v := range agentInstructionFile {
		if v == rel {
			return true
		}
	}
	return false
}

// installedAgents reads the comma-separated agent list from the stamp's config block.
func installedAgents(stamp map[string]any) []string {
	cfg, ok := stamp["config"].(map[string]any)
	if !ok {
		return nil
	}
	s, _ := cfg["agents"].(string)
	var out []string
	for _, k := range strings.Split(s, ",") {
		if k = strings.TrimSpace(k); k != "" {
			out = append(out, k)
		}
	}
	return out
}

// placedFile pairs a file init would place with its template source.
type placedFile struct {
	rel string // repo-relative dest (e.g. "AGENTS.md", "spec/README.md")
	src string // template path within tpl (e.g. "base/AGENTS.md", "agents/claude/CLAUDE.md")
}

// initFiles enumerates every file `init` would place: the base set plus the selected agents'
// adapters, as (dest, template-source) pairs. In spec-only mode the queue/claims/batch files (and
// the claim-batch/finish-batch skills) are filtered out — only the spec discipline is installed.
func initFiles(tpl fs.FS, agentKeys []string, mode string) ([]placedFile, error) {
	var out []placedFile
	add := func(root string) error {
		return fs.WalkDir(tpl, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				rel := strings.TrimPrefix(p, root+"/")
				if mode == "spec-only" && specOnlyOmits(rel) {
					return nil
				}
				out = append(out, placedFile{rel: rel, src: p})
			}
			return nil
		})
	}
	if err := add("base"); err != nil {
		return nil, err
	}
	for _, key := range agentKeys {
		root := "agents/" + key
		if _, err := fs.Stat(tpl, root); err != nil {
			continue
		}
		if err := add(root); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// injectRegion inserts specflow's marker-wrapped region (taken from templateContent) at the top of
// an existing file, preserving the file's current content below it. It returns ok=false when the
// file already carries a specflow region (already wired — caller leaves it for upgrade to refresh)
// or the template has no region. This is the non-destructive model applied at init time.
func injectRegion(existing, templateContent string) (string, bool) {
	if _, ok := extractRegion(existing); ok {
		return existing, false // already wired
	}
	src, ok := extractRegion(templateContent)
	if !ok {
		return existing, false
	}
	region := src.startMarker + src.region + src.endMarker
	return region + "\n\n" + existing, true
}

func writeFile(dest string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, content, 0o644)
}

// fillStamp substitutes the version/date/agents/mode/check placeholders in the freshly copied
// config template. check is the repo's single check command and is legitimately empty (the user
// skipped the prompt, or init ran non-interactively without --check).
func fillStamp(targetDir, version string, agentKeys []string, mode, check string) error {
	p := configPath(targetDir)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil // no stamp present (e.g. skipped) — nothing to fill
	}
	s := string(b)
	s = strings.Replace(s, "{{VERSION}}", version, 1)
	s = strings.Replace(s, "{{INIT_DATE}}", today(), 1)
	s = strings.Replace(s, "{{AGENTS}}", strings.Join(agentKeys, ","), 1)
	s = strings.Replace(s, "{{MODE}}", mode, 1)
	s = strings.Replace(s, "{{CHECK}}", jsonEscape(check), 1)
	return os.WriteFile(p, []byte(s), 0o644)
}

// computeManaged returns the baseline hash of each managed file's region as currently on disk (the
// base set plus the installed agents' instruction files). Stored in the stamp so a later upgrade can
// tell a pristine region (safe to refresh) from a hand-edited one (drift).
func computeManaged(targetDir string, tpl fs.FS, agentKeys []string, mode string) (map[string]string, error) {
	entries, err := managedEntries(tpl, agentKeys, mode)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, e := range entries {
		b, err := os.ReadFile(destPath(targetDir, e.rel))
		if err != nil {
			continue
		}
		if parts, ok := extractRegion(string(b)); ok {
			m[e.rel] = hashRegion(parts.region)
		}
	}
	return m, nil
}

// recordManaged reads the stamp, attaches/refreshes the managed-region baseline map (and any extra
// fields), and writes it back.
func recordManaged(targetDir string, tpl fs.FS, agentKeys []string, mode string, extra map[string]any) error {
	p := configPath(targetDir)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var stamp map[string]any
	if err := json.Unmarshal(b, &stamp); err != nil {
		return err
	}
	managed, err := computeManaged(targetDir, tpl, agentKeys, mode)
	if err != nil {
		return err
	}
	stamp["managed"] = managed
	for k, v := range extra {
		stamp[k] = v
	}
	return writeJSON(p, stamp)
}

// jsonEscape renders a string as a JSON string body (no surrounding quotes) so it can be
// substituted into the config template's "{{CHECK}}" slot without breaking the file. A check
// command legitimately contains quotes and backslashes (`sh -c "make check"`), and the template is
// text-substituted rather than marshalled, so escaping here is what keeps the stamp valid JSON.
func jsonEscape(s string) string {
	b, err := json.Marshal(s)
	if err != nil || len(b) < 2 {
		return ""
	}
	return string(b[1 : len(b)-1])
}

func writeJSON(p string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o644)
}

func isGitRepo(dir string) bool {
	cur := dir
	for {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return false
		}
		cur = parent
	}
}

// IsInstalled reports whether a specflow config already exists in targetDir.
func IsInstalled(targetDir string) bool {
	_, err := os.Stat(configPath(targetDir))
	return err == nil
}

// IsGitRepo reports whether targetDir is inside a git working tree.
func IsGitRepo(targetDir string) bool { return isGitRepo(targetDir) }

// initAction is what init will do with one file, decided by its template source vs. the on-disk
// state: create it fresh, inject specflow's region into an existing file, leave an
// already-wired file, or skip a pre-existing user-owned working file.
type initAction int

const (
	actionCreate       initAction = iota // file absent → write the template verbatim
	actionInject                         // managed file exists without a region → insert specflow's region
	actionAlreadyWired                   // managed file already carries a specflow region → leave for upgrade
	actionSkipExisting                   // non-managed file exists → user-owned, never touch
)

type fileAction struct {
	rel, src string
	action   initAction
}

// classifyInit decides, for every file init would place, what to do with it given the current disk
// state. Deterministic from (targetDir, tpl, agentKeys, mode) — PlanInit shows it, ApplyInit acts on it.
func classifyInit(targetDir string, tpl fs.FS, agentKeys []string, mode string) ([]fileAction, error) {
	entries, err := managedEntries(tpl, agentKeys, mode)
	if err != nil {
		return nil, err
	}
	managed := map[string]bool{}
	for _, e := range entries {
		managed[e.rel] = true
	}
	files, err := initFiles(tpl, agentKeys, mode)
	if err != nil {
		return nil, err
	}
	out := make([]fileAction, 0, len(files))
	for _, f := range files {
		a := fileAction{rel: f.rel, src: f.src}
		switch {
		case !pathExists(destPath(targetDir, f.rel)):
			a.action = actionCreate
		case managed[f.rel]:
			b, _ := os.ReadFile(destPath(targetDir, f.rel))
			content := string(b)
			switch {
			case hasRegionMarkers(content):
				// already carries a specflow region → leave it; upgrade refreshes it.
				a.action = actionAlreadyWired
			case f.rel != "AGENTS.md" && referencesAgents(content):
				// a per-agent file that already points at AGENTS.md (a link, mention, or @import) —
				// it's wired to the single source already; adding a second pointer would be noise.
				// (AGENTS.md itself is excluded: it must *carry* the protocol, not reference it.)
				a.action = actionAlreadyWired
			default:
				a.action = actionInject
			}
		default:
			a.action = actionSkipExisting
		}
		out = append(out, a)
	}
	return out, nil
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// InitPlan is what init will do, computed before any write so the CLI can show it and get consent.
// All fields are repo-relative paths.
type InitPlan struct {
	Create       []string // files to create fresh (specflow-owned)
	Inject       []string // existing files specflow will add its region to (content preserved)
	AlreadyWired []string // existing files that already carry a specflow region — left as-is
	SkipExisting []string // existing non-managed files — left untouched
}

// PlanInit classifies the work without touching disk.
func PlanInit(targetDir string, tpl fs.FS, agentKeys []string, mode string) (InitPlan, error) {
	acts, err := classifyInit(targetDir, tpl, agentKeys, mode)
	if err != nil {
		return InitPlan{}, err
	}
	var p InitPlan
	for _, a := range acts {
		switch a.action {
		case actionCreate:
			p.Create = append(p.Create, a.rel)
		case actionInject:
			p.Inject = append(p.Inject, a.rel)
		case actionAlreadyWired:
			p.AlreadyWired = append(p.AlreadyWired, a.rel)
		case actionSkipExisting:
			p.SkipExisting = append(p.SkipExisting, a.rel)
		}
	}
	return p, nil
}

// InitResult is what init actually did — its own tracked list of created/modified files (not derived
// from git, since the tree may carry unrelated changes), for the review handoff.
type InitResult struct {
	Created      []string
	Injected     []string
	AlreadyWired []string
	SkipExisting []string
	Declined     []string // inject targets the user declined (allowInject=false)
}

// ApplyInit performs the init writes non-destructively and fills the stamp. allowInject gates the
// injection of specflow's region into files that already exist (batched consent); when false those
// files are left untouched and reported as Declined — everything else is still written. check is
// recorded verbatim as config.check and may be empty. init never commits.
func ApplyInit(targetDir string, tpl fs.FS, version string, agentKeys []string, mode string, allowInject bool, check string) (InitResult, error) {
	var res InitResult
	acts, err := classifyInit(targetDir, tpl, agentKeys, mode)
	if err != nil {
		return res, err
	}
	for _, a := range acts {
		dest := destPath(targetDir, a.rel)
		switch a.action {
		case actionCreate:
			b, err := fs.ReadFile(tpl, a.src)
			if err != nil {
				return res, err
			}
			if err := writeFile(dest, renderFile(b, mode)); err != nil {
				return res, err
			}
			res.Created = append(res.Created, a.rel)
		case actionInject:
			if !allowInject {
				res.Declined = append(res.Declined, a.rel)
				continue
			}
			b, err := fs.ReadFile(tpl, a.src)
			if err != nil {
				return res, err
			}
			existing, err := os.ReadFile(dest)
			if err != nil {
				return res, err
			}
			injected, ok := injectRegion(string(existing), string(renderFile(b, mode)))
			if !ok {
				res.AlreadyWired = append(res.AlreadyWired, a.rel)
				continue
			}
			if err := os.WriteFile(dest, []byte(injected), 0o644); err != nil {
				return res, err
			}
			res.Injected = append(res.Injected, a.rel)
		case actionAlreadyWired:
			res.AlreadyWired = append(res.AlreadyWired, a.rel)
		case actionSkipExisting:
			res.SkipExisting = append(res.SkipExisting, a.rel)
		}
	}
	if err := fillStamp(targetDir, version, agentKeys, mode, check); err != nil {
		return res, err
	}
	if err := recordManaged(targetDir, tpl, agentKeys, mode, nil); err != nil {
		return res, err
	}
	return res, nil
}

// UpgradeResult is the outcome of an Upgrade, for the caller to report.
type UpgradeResult struct {
	NotInstalled  bool
	From          string
	To            string
	Refreshed     []string
	Added         []string
	Migrated      []string
	Drifted       []string
	SchemaChanged bool
}

// upgradeAction is the decision for one managed file during upgrade. It's computed without writing,
// so the apply path (Upgrade) and the --dry-run planner (PlanUpgrade) classify identically.
type upgradeAction int

const (
	upSkip    upgradeAction = iota // brownfield file, no baseline — never adopted; not reported
	upNoop                         // clean region already identical to the template — nothing to do
	upRefresh                      // clean region → swap in the fresh content
	upAdd                          // managed file absent → create it whole
	upMigrate                      // pre-marker file (had a baseline) → back up + rewrite
	upDrift                        // hand-edited / no baseline → write .specflow-new, don't overwrite
)

type upgradeDecision struct {
	action  upgradeAction
	updated string // full new file content (upRefresh/upAdd/upMigrate) or sidecar body (upDrift)
	backup  string // original bytes to preserve as .specflow-bak (upMigrate only)
	newHash string // region baseline to record (all actions that keep the file managed; not upDrift)
}

// decideUpgrade classifies one managed file from its rendered template (srcContent + its extracted
// srcParts) against the current disk state, touching nothing. The write side effects each action
// implies are carried in the decision so the caller can either apply or merely report them.
func decideUpgrade(dest, rel, srcContent string, srcParts regionParts, baseline map[string]string) (upgradeDecision, error) {
	db, err := os.ReadFile(dest)
	if err != nil {
		if !os.IsNotExist(err) {
			return upgradeDecision{}, err
		}
		// Managed file introduced by a newer kit version: write it whole.
		return upgradeDecision{action: upAdd, updated: srcContent, newHash: hashRegion(srcParts.region)}, nil
	}
	destContent := string(db)
	destParts, ok := extractRegion(destContent)
	if !ok {
		// No region on disk. Only migrate a file specflow actually owned before (a recorded baseline
		// ⇒ markers were stripped since install). A managed file with no region AND no baseline is a
		// brownfield file init deliberately left untouched (e.g. a CLAUDE.md that already pointed at
		// AGENTS.md) — never adopt/overwrite it.
		if baseline[rel] == "" {
			return upgradeDecision{action: upSkip}, nil
		}
		return upgradeDecision{action: upMigrate, updated: srcContent, backup: destContent, newHash: hashRegion(srcParts.region)}, nil
	}
	// Never overwrite a region we can't prove is pristine: a hash mismatch (hand-edited since install)
	// or a missing baseline (lost/corrupt stamp, or newly managed) leaves the on-disk region untouched
	// and drops the fresh version to a .specflow-new sidecar instead of clobbering in-region edits.
	if base := baseline[rel]; base == "" || hashRegion(destParts.region) != base {
		return upgradeDecision{action: upDrift, updated: srcContent}, nil
	}
	// Clean: swap in the fresh region (and the template's current marker wording), preserving
	// everything outside the markers verbatim.
	updated := destParts.before + srcParts.startMarker + srcParts.region + srcParts.endMarker + destParts.after
	if updated == destContent {
		return upgradeDecision{action: upNoop, newHash: hashRegion(srcParts.region)}, nil
	}
	return upgradeDecision{action: upRefresh, updated: updated, newHash: hashRegion(srcParts.region)}, nil
}

// relDecision pairs a managed file with its upgrade decision.
type relDecision struct {
	rel string
	dec upgradeDecision
}

// upgradeDecisions classifies every managed file for the install described by stamp, without writing.
// Shared by Upgrade (apply) and PlanUpgrade (--dry-run).
func upgradeDecisions(targetDir string, tpl fs.FS, stamp map[string]any) ([]relDecision, error) {
	mode := modeOf(stamp)
	baseline := baselineMap(stamp)
	entries, err := managedEntries(tpl, installedAgents(stamp), mode)
	if err != nil {
		return nil, err
	}
	var out []relDecision
	for _, e := range entries {
		srcb, err := fs.ReadFile(tpl, e.src)
		if err != nil {
			return nil, err
		}
		// Re-render the template region for the install's recorded mode, so a spec-only repo refreshes
		// to the spec-only composition (and its baseline hash, taken over the rendered region, matches).
		srcContent := renderBody(string(srcb), mode)
		srcParts, ok := extractRegion(srcContent)
		if !ok {
			continue // template lacks markers — nothing to manage (shouldn't happen)
		}
		d, err := decideUpgrade(destPath(targetDir, e.rel), e.rel, srcContent, srcParts, baseline)
		if err != nil {
			return nil, err
		}
		out = append(out, relDecision{rel: e.rel, dec: d})
	}
	return out, nil
}

// missingAdapterFiles returns the installed agents' non-managed adapter files (e.g. the Claude
// step-6 handoff hook) that `init` would place but that are absent here — the create-once files a
// newer kit shipped after this repo was installed. Managed-region files are handled by the region
// upgrade (a missing one is recreated via upAdd); base files are never recreated (a user may have
// deleted them deliberately), so this is scoped to the agents/ tree only. Create-once: present files
// are left untouched.
func missingAdapterFiles(targetDir string, tpl fs.FS, stamp map[string]any) ([]placedFile, error) {
	agents := installedAgents(stamp)
	mode := modeOf(stamp)
	files, err := initFiles(tpl, agents, mode)
	if err != nil {
		return nil, err
	}
	entries, err := managedEntries(tpl, agents, mode)
	if err != nil {
		return nil, err
	}
	managed := map[string]bool{}
	for _, e := range entries {
		managed[e.rel] = true
	}
	var out []placedFile
	for _, f := range files {
		if !strings.HasPrefix(f.src, "agents/") {
			continue // base files aren't recreated on upgrade
		}
		if managed[f.rel] {
			continue // region files are handled by the managed-region upgrade
		}
		if _, err := os.Stat(destPath(targetDir, f.rel)); err == nil {
			continue // create-once: already present
		}
		out = append(out, f)
	}
	return out, nil
}

// staleAdapterFiles returns the non-managed adapter files that are present but name machinery the
// install's mode omits — a skill stub left behind by a kit that shipped un-gated prose.
//
// These files are create-once and carry no baseline, so upgrade normally can't tell a pristine copy
// from a user-edited one and leaves them alone. A mode leak resolves that: no user would write
// "propagation to BUILD_QUEUE.md" into a repo that has no queue, so the leak itself proves the
// content is stale specflow text and is safe to replace. Without this, an existing spec-only install
// keeps a wrong `spec-edit` skill forever — and its YAML description loads into every session.
func staleAdapterFiles(targetDir string, tpl fs.FS, stamp map[string]any) ([]placedFile, error) {
	agents := installedAgents(stamp)
	mode := modeOf(stamp)
	files, err := initFiles(tpl, agents, mode)
	if err != nil {
		return nil, err
	}
	entries, err := managedEntries(tpl, agents, mode)
	if err != nil {
		return nil, err
	}
	managed := map[string]bool{}
	for _, e := range entries {
		managed[e.rel] = true
	}
	var out []placedFile
	for _, f := range files {
		if !strings.HasPrefix(f.src, "agents/") || managed[f.rel] {
			continue
		}
		b, err := os.ReadFile(destPath(targetDir, f.rel))
		if err != nil {
			continue // absent — missingAdapterFiles handles placement
		}
		if len(ModeLeaks(mode, string(b))) > 0 {
			out = append(out, f)
		}
	}
	return out, nil
}

// Upgrade refreshes specflow's managed region in each managed file to the installed kit version,
// non-destructively: a clean region has only its between-markers content replaced; a drifted region
// is left untouched with the fresh version dropped to a .specflow-new sidecar; a pre-marker file is
// migrated (backed up to .specflow-bak, then rewritten). Text outside the markers is never touched.
// It also places any non-managed adapter file a newer kit added for the installed agents (create-once).
func Upgrade(targetDir string, tpl fs.FS, version string) (UpgradeResult, error) {
	res := UpgradeResult{To: version}
	sp := configPath(targetDir)
	sb, err := os.ReadFile(sp)
	if err != nil {
		res.NotInstalled = true
		return res, nil
	}
	var stamp map[string]any
	if err := json.Unmarshal(sb, &stamp); err != nil {
		return res, fmt.Errorf("%s is corrupted (invalid JSON) — fix or restore it before upgrading: %w", filepath.Base(sp), err)
	}
	res.From, _ = stamp["kitVersion"].(string)

	baseline := baselineMap(stamp)
	next := map[string]string{}
	for k, v := range baseline {
		next[k] = v
	}

	decisions, err := upgradeDecisions(targetDir, tpl, stamp)
	if err != nil {
		return res, err
	}
	for _, rd := range decisions {
		rel, d := rd.rel, rd.dec
		dest := destPath(targetDir, rel)
		switch d.action {
		case upSkip:
			// brownfield file specflow never owned — leave it, don't record a baseline.
		case upNoop:
			next[rel] = d.newHash
		case upAdd:
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return res, err
			}
			if err := os.WriteFile(dest, []byte(d.updated), 0o644); err != nil {
				return res, err
			}
			next[rel] = d.newHash
			res.Added = append(res.Added, rel)
		case upMigrate:
			if err := os.WriteFile(dest+".specflow-bak", []byte(d.backup), 0o644); err != nil {
				return res, err
			}
			if err := os.WriteFile(dest, []byte(d.updated), 0o644); err != nil {
				return res, err
			}
			next[rel] = d.newHash
			res.Migrated = append(res.Migrated, rel)
		case upDrift:
			if err := os.WriteFile(dest+".specflow-new", []byte(d.updated), 0o644); err != nil {
				return res, err
			}
			res.Drifted = append(res.Drifted, rel)
		case upRefresh:
			if err := os.WriteFile(dest, []byte(d.updated), 0o644); err != nil {
				return res, err
			}
			next[rel] = d.newHash
			res.Refreshed = append(res.Refreshed, rel)
		}
	}

	// Place non-managed adapter files a newer kit added (e.g. the Claude handoff hook) — create-once,
	// not baselined (init doesn't baseline these either).
	missing, err := missingAdapterFiles(targetDir, tpl, stamp)
	if err != nil {
		return res, err
	}
	for _, f := range missing {
		srcb, err := fs.ReadFile(tpl, f.src)
		if err != nil {
			return res, err
		}
		dest := destPath(targetDir, f.rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return res, err
		}
		if err := os.WriteFile(dest, renderFile(srcb, modeOf(stamp)), 0o644); err != nil {
			return res, err
		}
		res.Added = append(res.Added, f.rel)
	}

	// Replace non-managed adapter files whose content names machinery this mode omits.
	stale, err := staleAdapterFiles(targetDir, tpl, stamp)
	if err != nil {
		return res, err
	}
	for _, f := range stale {
		srcb, err := fs.ReadFile(tpl, f.src)
		if err != nil {
			return res, err
		}
		if err := os.WriteFile(destPath(targetDir, f.rel), renderFile(srcb, modeOf(stamp)), 0o644); err != nil {
			return res, err
		}
		res.Refreshed = append(res.Refreshed, f.rel)
	}

	stamp["kitVersion"] = version
	stamp["upgradedAt"] = today()
	stamp["managed"] = next
	if err := writeJSON(sp, stamp); err != nil {
		return res, err
	}
	if sv, ok := stamp["schemaVersion"].(float64); !ok || sv != 1 {
		res.SchemaChanged = true
	}
	return res, nil
}

// UpgradePlan is what an upgrade would do, computed without touching disk — for `upgrade --dry-run`.
type UpgradePlan struct {
	NotInstalled bool
	From, To     string
	Refresh      []string // clean regions that would be refreshed
	Add          []string // managed files that would be created
	Migrate      []string // pre-marker files that would be backed up + rewritten
	Drift        []string // drifted regions left untouched (fresh version → .specflow-new)
}

// PlanUpgrade classifies what Upgrade would do, writing nothing. upNoop/upSkip files are omitted —
// the plan lists only the changes a real upgrade would make.
func PlanUpgrade(targetDir string, tpl fs.FS, version string) (UpgradePlan, error) {
	plan := UpgradePlan{To: version}
	sb, err := os.ReadFile(configPath(targetDir))
	if err != nil {
		plan.NotInstalled = true
		return plan, nil
	}
	var stamp map[string]any
	if err := json.Unmarshal(sb, &stamp); err != nil {
		return plan, fmt.Errorf("%s is corrupted (invalid JSON) — fix or restore it before upgrading: %w", filepath.Base(configPath(targetDir)), err)
	}
	plan.From, _ = stamp["kitVersion"].(string)
	decisions, err := upgradeDecisions(targetDir, tpl, stamp)
	if err != nil {
		return plan, err
	}
	for _, rd := range decisions {
		switch rd.dec.action {
		case upRefresh:
			plan.Refresh = append(plan.Refresh, rd.rel)
		case upAdd:
			plan.Add = append(plan.Add, rd.rel)
		case upMigrate:
			plan.Migrate = append(plan.Migrate, rd.rel)
		case upDrift:
			plan.Drift = append(plan.Drift, rd.rel)
		}
	}
	missing, err := missingAdapterFiles(targetDir, tpl, stamp)
	if err != nil {
		return plan, err
	}
	for _, f := range missing {
		plan.Add = append(plan.Add, f.rel)
	}
	return plan, nil
}

// AddAgentResult reports what `add-agent` did for one agent, for the CLI to summarize.
type AddAgentResult struct {
	NotInstalled   bool     // no specflow install in targetDir
	AlreadyPresent bool     // agent already recorded in config.agents — no-op
	Key            string   // the agent key acted on
	Created        []string // adapter files written fresh
	Injected       []string // existing instruction file specflow injected its region into
	AlreadyWired   []string // instruction file already carried a region / pointed at AGENTS.md
	SkipExisting   []string // specflow-owned adapter files already present — left untouched
	Agents         []string // resulting installed-agent list
}

// AddAgent copies one agent's adapter into an already-initialized repo (non-destructive) and records
// it in the stamp's config.agents. It mirrors init's per-file behavior for that agent: a missing file
// is created; the agent's instruction file, if it already exists, has specflow's region injected (or
// is left alone when already wired); any other adapter file that already exists is left untouched
// (skip-existing). The install mode is read from the stamp, so a spec-only repo doesn't gain the
// claim/finish skills. Nothing is committed — the caller reviews and commits. A no-op AlreadyPresent
// result means the agent was already installed. The key is assumed valid (caller validates against
// the known-agent list); an unknown key with no adapter templates returns an error.
func AddAgent(targetDir string, tpl fs.FS, version, key string) (AddAgentResult, error) {
	res := AddAgentResult{Key: key}
	p := configPath(targetDir)
	sb, err := os.ReadFile(p)
	if err != nil {
		res.NotInstalled = true
		return res, nil
	}
	var stamp map[string]any
	if err := json.Unmarshal(sb, &stamp); err != nil {
		return res, fmt.Errorf("%s is corrupted (invalid JSON) — fix or restore it before adding an agent: %w", filepath.Base(p), err)
	}
	mode := modeOf(stamp)
	existing := installedAgents(stamp)
	for _, k := range existing {
		if k == key {
			res.AlreadyPresent = true
			res.Agents = existing
			return res, nil
		}
	}

	root := "agents/" + key
	if _, err := fs.Stat(tpl, root); err != nil {
		return res, fmt.Errorf("no adapter templates for agent %q", key)
	}
	instrRel := agentInstructionFile[key]
	err = fs.WalkDir(tpl, root, func(pth string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(pth, root+"/")
		if mode == "spec-only" && specOnlyOmits(rel) {
			return nil // spec-only repo: no claim/finish skills
		}
		dest := destPath(targetDir, rel)
		b, err := fs.ReadFile(tpl, pth)
		if err != nil {
			return err
		}
		content := renderFile(b, mode)
		if !pathExists(dest) {
			if err := writeFile(dest, content); err != nil {
				return err
			}
			res.Created = append(res.Created, rel)
			return nil
		}
		// File exists. The managed instruction file gets specflow's region injected (non-destructive,
		// content preserved); already-wired files and every other adapter file are left untouched.
		if rel == instrRel {
			db, err := os.ReadFile(dest)
			if err != nil {
				return err
			}
			cur := string(db)
			if hasRegionMarkers(cur) || referencesAgents(cur) {
				res.AlreadyWired = append(res.AlreadyWired, rel)
				return nil
			}
			injected, ok := injectRegion(cur, string(content))
			if !ok {
				res.AlreadyWired = append(res.AlreadyWired, rel)
				return nil
			}
			if err := os.WriteFile(dest, []byte(injected), 0o644); err != nil {
				return err
			}
			res.Injected = append(res.Injected, rel)
			return nil
		}
		res.SkipExisting = append(res.SkipExisting, rel)
		return nil
	})
	if err != nil {
		return res, err
	}

	// Record the agent in config.agents and refresh the managed-region baselines for the full set,
	// so upgrade tracks the new instruction file's region. Files are already on disk, so
	// computeManaged fingerprints the regions just written.
	full := append(append([]string{}, existing...), key)
	if cfg, ok := stamp["config"].(map[string]any); ok {
		cfg["agents"] = strings.Join(full, ",")
	} else {
		stamp["config"] = map[string]any{"agents": strings.Join(full, ",")}
	}
	managed, err := computeManaged(targetDir, tpl, full, mode)
	if err != nil {
		return res, err
	}
	stamp["managed"] = managed
	if err := writeJSON(p, stamp); err != nil {
		return res, err
	}
	res.Agents = full
	return res, nil
}

// Section/heading matchers for reading the queue + claims ledger in Status. The In-progress section
// runs from its heading to the next top-level `## ` heading (always `## Completed` in practice); a
// batch is any `## Batch …` heading in BUILD_QUEUE.md (done batches are removed from that file).
var (
	inProgressRe   = regexp.MustCompile(`(?ms)^##\s+In progress\s*$(.*?)^##\s`)
	claimHeadingRe = regexp.MustCompile(`(?m)^###\s+(.+?)\s*$`)
	ownerLineRe    = regexp.MustCompile(`(?m)^\s*-\s*Owner:\s*(.+?)\s*$`)
	batchHeadingRe = regexp.MustCompile(`(?m)^##\s+Batch\b`)
)

// ClaimLine is one active claim parsed from CLAIMS.md's In-progress section.
type ClaimLine struct {
	Batch string // the ### heading text (e.g. "Batch 2 — `specflow status`")
	Owner string // the Owner: value, or "" when unset
}

// parseInProgress extracts the active claims from the In-progress section body, pairing each ###
// heading with the first Owner: line beneath it.
func parseInProgress(section string) []ClaimLine {
	locs := claimHeadingRe.FindAllStringSubmatchIndex(section, -1)
	out := make([]ClaimLine, 0, len(locs))
	for i, loc := range locs {
		end := len(section)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		owner := ""
		if m := ownerLineRe.FindStringSubmatch(section[loc[1]:end]); m != nil {
			owner = strings.TrimSpace(m[1])
		}
		out = append(out, ClaimLine{Batch: strings.TrimSpace(section[loc[2]:loc[3]]), Owner: owner})
	}
	return out
}

// StatusReport is a read-only snapshot of a specflow install, for `specflow status`.
type StatusReport struct {
	Installed     bool
	Mode          string      // full | spec-only
	StampVersion  string      // kitVersion recorded in config.json
	BinaryVersion string      // the running binary's version
	VersionMatch  bool        // stamp == binary
	Agents        []string    // wired agents
	Commit, Push  string      // config levers
	Check         string      // config.check — the repo's single check command, "" when unset
	HasQueue      bool        // BUILD_QUEUE.md present (full mode)
	UndoneBatches int         // count of un-done batches; -1 when there's no queue (spec-only)
	InProgress    []ClaimLine // active claims from CLAIMS.md
	Drifted       []string    // managed files whose region no longer matches its baseline
}

// Status assembles a read-only snapshot of the install: versions, mode, agents, commit/push levers,
// active claims, the un-done batch count, and any managed-region drift. It writes nothing and never
// fails on a missing queue/claims file (spec-only installs have neither). A corrupt config.json is
// the one hard error.
func Status(targetDir string, tpl fs.FS, version string) (StatusReport, error) {
	rep := StatusReport{BinaryVersion: version, UndoneBatches: -1}
	sb, err := os.ReadFile(configPath(targetDir))
	if err != nil {
		return rep, nil // Installed=false — caller prints the not-installed notice
	}
	var stamp map[string]any
	if err := json.Unmarshal(sb, &stamp); err != nil {
		return rep, fmt.Errorf("%s is corrupted (invalid JSON) — fix or restore it: %w", filepath.Base(configPath(targetDir)), err)
	}
	rep.Installed = true
	rep.StampVersion, _ = stamp["kitVersion"].(string)
	rep.VersionMatch = rep.StampVersion == version
	rep.Mode = modeOf(stamp)
	rep.Agents = installedAgents(stamp)
	if cfg, ok := stamp["config"].(map[string]any); ok {
		rep.Commit, _ = cfg["commit"].(string)
		rep.Push, _ = cfg["push"].(string)
		rep.Check, _ = cfg["check"].(string) // absent in installs predating the field

	}

	// Drift: a managed region whose current hash differs from the recorded baseline (same test the
	// upgrade/verify paths use). Missing files / missing baselines aren't drift — they're a verify
	// concern, kept out of this orientation summary.
	baseline := baselineMap(stamp)
	entries, err := managedEntries(tpl, rep.Agents, rep.Mode)
	if err != nil {
		return rep, err
	}
	for _, e := range entries {
		b, err := os.ReadFile(destPath(targetDir, e.rel))
		if err != nil {
			continue
		}
		parts, ok := extractRegion(string(b))
		if !ok {
			continue
		}
		if base := baseline[e.rel]; base != "" && hashRegion(parts.region) != base {
			rep.Drifted = append(rep.Drifted, e.rel)
		}
	}

	// Queue + claims live at the repo root in full mode; spec-only installs have neither.
	if qb, err := os.ReadFile(filepath.Join(targetDir, "BUILD_QUEUE.md")); err == nil {
		rep.HasQueue = true
		rep.UndoneBatches = len(batchHeadingRe.FindAllString(string(qb), -1))
	}
	if cb, err := os.ReadFile(filepath.Join(targetDir, "CLAIMS.md")); err == nil {
		if m := inProgressRe.FindStringSubmatch(string(cb)); m != nil {
			rep.InProgress = parseInProgress(m[1])
		}
	}
	return rep, nil
}

// VerifyReport is the outcome of an install-integrity check. Problems are Tier-1 failures (specflow
// can't work properly); Warnings are Tier-3 / drift issues (degraded but functional); OK lists the
// pieces that checked out.
type VerifyReport struct {
	Installed bool
	Mode      string // full | spec-only — surfaced so the report says which mode it validated
	OK        []string
	Warnings  []string
	Problems  []string
}

// Verify checks installation integrity against the working tree (so it passes on a fresh,
// uncommitted init): a valid stamp, the Tier-1 managed files present with intact regions (drift
// reported), and the installed agents' Tier-3 instruction files present and wired to AGENTS.md.
func Verify(targetDir string, tpl fs.FS, version string) (VerifyReport, error) {
	var rep VerifyReport
	sb, err := os.ReadFile(configPath(targetDir))
	if err != nil {
		rep.Problems = append(rep.Problems, "specflow/config.json not found — run `specflow init`")
		return rep, nil
	}
	var stamp map[string]any
	if err := json.Unmarshal(sb, &stamp); err != nil {
		rep.Problems = append(rep.Problems, "specflow/config.json is corrupt (invalid JSON) — fix or restore it")
		return rep, nil
	}
	rep.Installed = true
	rep.OK = append(rep.OK, "specflow/config.json valid")

	baseline := baselineMap(stamp)
	mode := modeOf(stamp)
	rep.Mode = mode
	entries, err := managedEntries(tpl, installedAgents(stamp), mode)
	if err != nil {
		return rep, err
	}
	for _, e := range entries {
		tier3 := isPerAgentFile(e.rel)
		b, err := os.ReadFile(destPath(targetDir, e.rel))
		if err != nil {
			if tier3 {
				rep.Warnings = append(rep.Warnings, e.rel+" missing — that agent isn't auto-wired (point its file at AGENTS.md)")
			} else {
				rep.Problems = append(rep.Problems, e.rel+" missing — specflow can't work properly")
			}
			continue
		}
		content := string(b)
		parts, ok := extractRegion(content)
		if !ok {
			switch {
			case tier3 && referencesAgents(content):
				rep.OK = append(rep.OK, e.rel+" (points at AGENTS.md)")
			case tier3:
				rep.Warnings = append(rep.Warnings, e.rel+" has no specflow region and doesn't reference AGENTS.md")
			default:
				rep.Problems = append(rep.Problems, e.rel+" has no specflow region (markers missing) — run `specflow upgrade`")
			}
			continue
		}
		// Mode consistency, checked over the managed region only — text outside the markers is the
		// user's and specflow doesn't police it.
		if leaks := ModeLeaks(mode, parts.region); len(leaks) > 0 {
			rep.Problems = append(rep.Problems, e.rel+" names batch/claim machinery this spec-only install doesn't have ("+
				strings.Join(leaks, ", ")+") — run `specflow upgrade`")
			continue
		}
		if base := baseline[e.rel]; base != "" && hashRegion(parts.region) != base {
			rep.Warnings = append(rep.Warnings, e.rel+" region edited since install (drift) — `upgrade` won't refresh it")
		} else {
			rep.OK = append(rep.OK, e.rel)
		}
	}

	// Non-managed adapter files (skill stubs, hooks) have no region to hash, so the loop above never
	// sees them — but a stale one is exactly as misleading to an agent as a stale region.
	stale, err := staleAdapterFiles(targetDir, tpl, stamp)
	if err != nil {
		return rep, err
	}
	for _, f := range stale {
		rep.Problems = append(rep.Problems, f.rel+" names batch/claim machinery this spec-only install doesn't have — run `specflow upgrade`")
	}
	return rep, nil
}
