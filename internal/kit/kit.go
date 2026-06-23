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
// a repo with those agents.
func managedEntries(tpl fs.FS, agentKeys []string) ([]managedEntry, error) {
	var out []managedEntry
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
					out = append(out, managedEntry{rel: strings.TrimPrefix(p, "base/"), src: p})
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			out = append(out, managedEntry{rel: rel, src: src})
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
		out = append(out, managedEntry{rel: rel, src: src})
	}
	return out, nil
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
// adapters, as (dest, template-source) pairs.
func initFiles(tpl fs.FS, agentKeys []string) ([]placedFile, error) {
	var out []placedFile
	add := func(root string) error {
		return fs.WalkDir(tpl, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				out = append(out, placedFile{rel: strings.TrimPrefix(p, root+"/"), src: p})
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

// fillStamp substitutes the version/date/agents placeholders in the freshly copied config template.
func fillStamp(targetDir, version string, agentKeys []string) error {
	p := configPath(targetDir)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil // no stamp present (e.g. skipped) — nothing to fill
	}
	s := string(b)
	s = strings.Replace(s, "{{VERSION}}", version, 1)
	s = strings.Replace(s, "{{INIT_DATE}}", today(), 1)
	s = strings.Replace(s, "{{AGENTS}}", strings.Join(agentKeys, ","), 1)
	return os.WriteFile(p, []byte(s), 0o644)
}

// computeManaged returns the baseline hash of each managed file's region as currently on disk (the
// base set plus the installed agents' instruction files). Stored in the stamp so a later upgrade can
// tell a pristine region (safe to refresh) from a hand-edited one (drift).
func computeManaged(targetDir string, tpl fs.FS, agentKeys []string) (map[string]string, error) {
	entries, err := managedEntries(tpl, agentKeys)
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
func recordManaged(targetDir string, tpl fs.FS, agentKeys []string, extra map[string]any) error {
	p := configPath(targetDir)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var stamp map[string]any
	if err := json.Unmarshal(b, &stamp); err != nil {
		return err
	}
	managed, err := computeManaged(targetDir, tpl, agentKeys)
	if err != nil {
		return err
	}
	stamp["managed"] = managed
	for k, v := range extra {
		stamp[k] = v
	}
	return writeJSON(p, stamp)
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
// state. Deterministic from (targetDir, tpl, agentKeys) — PlanInit shows it, ApplyInit acts on it.
func classifyInit(targetDir string, tpl fs.FS, agentKeys []string) ([]fileAction, error) {
	entries, err := managedEntries(tpl, agentKeys)
	if err != nil {
		return nil, err
	}
	managed := map[string]bool{}
	for _, e := range entries {
		managed[e.rel] = true
	}
	files, err := initFiles(tpl, agentKeys)
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
			if _, ok := extractRegion(string(b)); ok {
				a.action = actionAlreadyWired
			} else {
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
func PlanInit(targetDir string, tpl fs.FS, agentKeys []string) (InitPlan, error) {
	acts, err := classifyInit(targetDir, tpl, agentKeys)
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
// files are left untouched and reported as Declined — everything else is still written. init never
// commits.
func ApplyInit(targetDir string, tpl fs.FS, version string, agentKeys []string, allowInject bool) (InitResult, error) {
	var res InitResult
	acts, err := classifyInit(targetDir, tpl, agentKeys)
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
			if err := writeFile(dest, b); err != nil {
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
			injected, ok := injectRegion(string(existing), string(b))
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
	if err := fillStamp(targetDir, version, agentKeys); err != nil {
		return res, err
	}
	if err := recordManaged(targetDir, tpl, agentKeys, nil); err != nil {
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

// Upgrade refreshes specflow's managed region in each managed file to the installed kit version,
// non-destructively: a clean region has only its between-markers content replaced; a drifted region
// is left untouched with the fresh version dropped to a .specflow-new sidecar; a pre-marker file is
// migrated (backed up to .specflow-bak, then rewritten). Text outside the markers is never touched.
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

	baseline := map[string]string{}
	if raw, ok := stamp["managed"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				baseline[k] = s
			}
		}
	}
	next := map[string]string{}
	for k, v := range baseline {
		next[k] = v
	}

	entries, err := managedEntries(tpl, installedAgents(stamp))
	if err != nil {
		return res, err
	}
	for _, e := range entries {
		rel := e.rel
		srcb, err := fs.ReadFile(tpl, e.src)
		if err != nil {
			return res, err
		}
		srcContent := string(srcb)
		srcParts, ok := extractRegion(srcContent)
		if !ok {
			continue // template lacks markers — nothing to manage (shouldn't happen)
		}
		dest := destPath(targetDir, rel)

		db, readErr := os.ReadFile(dest)
		if readErr != nil {
			if !os.IsNotExist(readErr) {
				return res, readErr
			}
			// Managed file introduced by a newer kit version: write it whole.
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return res, err
			}
			if err := os.WriteFile(dest, []byte(srcContent), 0o644); err != nil {
				return res, err
			}
			next[rel] = hashRegion(srcParts.region)
			res.Added = append(res.Added, rel)
			continue
		}
		destContent := string(db)
		destParts, ok := extractRegion(destContent)
		if !ok {
			// Pre-marker install: back up verbatim, then write the marked template.
			if err := os.WriteFile(dest+".specflow-bak", db, 0o644); err != nil {
				return res, err
			}
			if err := os.WriteFile(dest, []byte(srcContent), 0o644); err != nil {
				return res, err
			}
			next[rel] = hashRegion(srcParts.region)
			res.Migrated = append(res.Migrated, rel)
			continue
		}

		// Never overwrite a region we can't prove is pristine. Two cases leave the on-disk region
		// untouched, drop the fresh version to a sidecar, and keep flagging:
		//   - hash mismatch → the region was hand-edited since install (drift);
		//   - missing baseline → no recorded fingerprint to compare against (a lost/corrupt stamp,
		//     or a file newly brought under management). Defaulting to "overwrite" here would
		//     silently clobber a user's in-region edits, so we default to "don't touch".
		if base := baseline[rel]; base == "" || hashRegion(destParts.region) != base {
			if err := os.WriteFile(dest+".specflow-new", []byte(srcContent), 0o644); err != nil {
				return res, err
			}
			res.Drifted = append(res.Drifted, rel)
			continue
		}

		// Clean: swap in the fresh region (and the template's current marker wording), preserving
		// everything outside the markers verbatim.
		updated := destParts.before + srcParts.startMarker + srcParts.region + srcParts.endMarker + destParts.after
		if updated != destContent {
			if err := os.WriteFile(dest, []byte(updated), 0o644); err != nil {
				return res, err
			}
			res.Refreshed = append(res.Refreshed, rel)
		}
		next[rel] = hashRegion(srcParts.region)
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
