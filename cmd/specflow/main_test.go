package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/MatanKoby/specflow/internal/kit"
)

// sha256Hex mirrors the whole-file baseline the kit records for an adapter, so a test can forge a
// stamp that claims a given content is what specflow installed.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// The tests build the real binary once, then drive it against temp repos and assert observable
// behavior — the Go port of the original Node smoke suite.

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "specflow-bin-")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "specflow")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("build failed: " + err.Error())
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type result struct {
	stdout, stderr string
	code           int
}

func run(t *testing.T, cwd string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = cwd
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return result{out.String(), errb.String(), code}
}

// runStdin is like run but feeds the process stdin (for interactive prompts).
func runStdin(t *testing.T, cwd, in string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader(in)
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return result{out.String(), errb.String(), code}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

// newRepo makes a temp dir that is a git work tree — `init` requires one.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return dir
}

func TestInitWritesFilesAndStamp(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "init", "--agents=claude,cursor")
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}

	for _, f := range []string{
		"AGENTS.md", "BUILD_QUEUE.md", "specflow/history/BUILD_QUEUE_DONE.md", "CLAIMS.md", "specflow/history/CLAIMS_DONE.md",
		"spec/README.md", "specflow/config.json",
		"specflow/procedures/claim-batch.md", "specflow/procedures/finish-batch.md", "specflow/procedures/spec-edit.md",
		"specflow/procedures/prune-ledgers.md",
		"CLAUDE.md", ".claude/skills/claim-batch/SKILL.md", ".claude/skills/prune-ledgers/SKILL.md",
		".cursor/rules/specflow.mdc",
	} {
		if !exists(filepath.Join(tmp, f)) {
			t.Errorf("missing %s", f)
		}
	}

	if exists(filepath.Join(tmp, ".github/copilot-instructions.md")) {
		t.Error("unselected copilot adapter leaked in")
	}

	raw := read(t, filepath.Join(tmp, "specflow/config.json"))
	if strings.Contains(raw, "{{") {
		t.Error("unfilled placeholder remains in config")
	}
	var stamp map[string]any
	if err := json.Unmarshal([]byte(raw), &stamp); err != nil {
		t.Fatalf("config not valid JSON: %v", err)
	}
	if v, _ := stamp["kitVersion"].(string); !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(v) {
		t.Errorf("kitVersion not a version: %q", v)
	}
	cfg, ok := stamp["config"].(map[string]any)
	if !ok {
		t.Fatal("config block missing")
	}
	if v, _ := cfg["agents"].(string); v != "claude,cursor" {
		t.Errorf("config.agents = %q, want claude,cursor", v)
	}
	if v, _ := cfg["mode"].(string); v != "full" {
		t.Errorf("config.mode = %q, want full", v)
	}
	managed, ok := stamp["managed"].(map[string]any)
	if !ok {
		t.Fatal("stamp has no managed map")
	}
	if h, _ := managed["AGENTS.md"].(string); h == "" {
		t.Error("no managed baseline hash for AGENTS.md")
	}

	for _, f := range []string{"AGENTS.md", "specflow/procedures/claim-batch.md"} {
		c := read(t, filepath.Join(tmp, f))
		if !regexp.MustCompile(`(?s)<!--\s*specflow:start\b`).MatchString(c) ||
			!regexp.MustCompile(`(?s)<!--\s*specflow:end\b`).MatchString(c) {
			t.Errorf("%s lacks region markers", f)
		}
	}
}

func TestInitBrownfieldInjectsAndPreserves(t *testing.T) {
	tmp := newRepo(t)
	// Pre-existing brownfield files: a managed AGENTS.md + CLAUDE.md with user content, and a
	// user-owned spec file specflow must not touch.
	mustWrite(t, filepath.Join(tmp, "AGENTS.md"), "# Our existing agent notes\nDeploy on Fridays.\n")
	mustWrite(t, filepath.Join(tmp, "CLAUDE.md"), "# Existing CLAUDE\nuse pnpm.\n")
	mustWrite(t, filepath.Join(tmp, "spec/README.md"), "OUR OWN SPEC — do not touch\n")

	r := run(t, tmp, "init", "--agents=claude") // non-interactive → proceeds without prompts
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}

	ag := read(t, filepath.Join(tmp, "AGENTS.md"))
	if !startMarker.MatchString(ag) {
		t.Error("AGENTS.md got no injected specflow region")
	}
	if !strings.Contains(ag, "Deploy on Fridays.") {
		t.Error("existing AGENTS.md content was lost on injection")
	}
	if strings.Index(ag, "specflow:start") > strings.Index(ag, "Deploy on Fridays.") {
		t.Error("specflow region not injected above the existing content")
	}

	cl := read(t, filepath.Join(tmp, "CLAUDE.md"))
	if !startMarker.MatchString(cl) || !strings.Contains(cl, "use pnpm.") {
		t.Error("CLAUDE.md injection lost content or skipped the region")
	}

	if read(t, filepath.Join(tmp, "spec/README.md")) != "OUR OWN SPEC — do not touch\n" {
		t.Error("existing user-owned spec/README.md was modified")
	}

	if !strings.Contains(r.stdout, "git diff") || !strings.Contains(r.stdout, "meta: install specflow") {
		t.Error("init did not print the review/commit handoff")
	}
	// init must never commit.
	if c := strings.TrimSpace(gitOut(t, tmp, "rev-list", "--all", "--count")); c != "0" {
		t.Errorf("init created commits (count %q) — it must never commit", c)
	}
	// The injected region is recorded so upgrade can refresh it.
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp)
	managed, _ := stamp["managed"].(map[string]any)
	if h, _ := managed["AGENTS.md"].(string); h == "" {
		t.Error("injected AGENTS.md not recorded in managed baseline")
	}
}

func TestInitInteractiveDeclineInjection(t *testing.T) {
	tmp := newRepo(t)
	mustWrite(t, filepath.Join(tmp, "AGENTS.md"), "# mine\nkeep me\n")
	// stdin: Enter (default Claude) for the agent pick, Enter to skip the check command, then "n"
	// to decline injection.
	r := runStdin(t, tmp, "\n\nn\n", "init")
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	ag := read(t, filepath.Join(tmp, "AGENTS.md"))
	if startMarker.MatchString(ag) {
		t.Error("declined injection still modified AGENTS.md")
	}
	if !strings.Contains(ag, "keep me") {
		t.Error("declined path damaged the existing file")
	}
	// Declining AGENTS.md is a Tier-1 problem — the notice must say specflow can't work properly.
	if !strings.Contains(r.stdout, "can't work properly") {
		t.Error("no Tier-1 warning printed for a declined AGENTS.md region")
	}
	// Everything else still installed.
	if !exists(filepath.Join(tmp, "specflow/config.json")) || !exists(filepath.Join(tmp, "BUILD_QUEUE.md")) {
		t.Error("declining injection should still install the rest")
	}
}

func TestInitIdempotentWhenFileReferencesAgents(t *testing.T) {
	tmp := newRepo(t)
	// A brownfield CLAUDE.md that already points at AGENTS.md — init must not add a second pointer.
	mustWrite(t, filepath.Join(tmp, "CLAUDE.md"), "# My Claude notes\nSee AGENTS.md for the protocol.\nuse pnpm.\n")

	r := run(t, tmp, "init", "--agents=claude")
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	cl := read(t, filepath.Join(tmp, "CLAUDE.md"))
	if startMarker.MatchString(cl) {
		t.Error("init injected a region into a CLAUDE.md that already references AGENTS.md")
	}
	if !strings.Contains(cl, "use pnpm.") {
		t.Error("existing CLAUDE.md content changed")
	}
	if !strings.Contains(r.stdout, "Already wired") {
		t.Error("no 'already wired' notice for the referenced file")
	}

	// A later upgrade must leave that un-injected brownfield file alone (no migrate/clobber).
	if ru := run(t, tmp, "upgrade"); ru.code != 0 {
		t.Fatalf("upgrade exit %d: %s", ru.code, ru.stderr)
	}
	if startMarker.MatchString(read(t, filepath.Join(tmp, "CLAUDE.md"))) {
		t.Error("upgrade adopted/migrated an un-injected brownfield CLAUDE.md")
	}
	if exists(filepath.Join(tmp, "CLAUDE.md.specflow-bak")) {
		t.Error("upgrade backed up + migrated a file specflow never owned")
	}
}

func TestInitRefusesOutsideGit(t *testing.T) {
	tmp := t.TempDir() // deliberately NOT a git repo
	r := run(t, tmp, "init", "--agents=claude")
	if !regexp.MustCompile(`(?i)git`).MatchString(r.stdout) {
		t.Error("no git-required message")
	}
	if exists(filepath.Join(tmp, "AGENTS.md")) {
		t.Error("init wrote files outside a git repo — it should write nothing")
	}
}

func TestReinitGuarded(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	r := run(t, tmp, "init", "--agents=claude")
	if r.code != 0 {
		t.Fatalf("re-init exit %d", r.code)
	}
	if !regexp.MustCompile(`(?i)already`).MatchString(r.stdout) {
		t.Error("no 'already installed' notice on re-init")
	}
}

func TestUpgradePreservesStateAndStamps(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	claimsBefore := read(t, filepath.Join(tmp, "CLAIMS.md"))

	r := run(t, tmp, "upgrade")
	if r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	if read(t, filepath.Join(tmp, "CLAIMS.md")) != claimsBefore {
		t.Error("upgrade modified CLAIMS.md (a state file)")
	}
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp)
	if _, ok := stamp["upgradedAt"]; !ok {
		t.Error("upgrade did not record upgradedAt")
	}
}

var startMarker = regexp.MustCompile(`(?s)<!--\s*specflow:start\b.*?-->`)

func TestUpgradePreservesTextOutsideMarkers(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")

	f, _ := os.OpenFile(ag, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("\n## Our team notes\nDeploy on Fridays only.\n")
	f.Close()

	r := run(t, tmp, "upgrade")
	if r.code != 0 {
		t.Fatalf("upgrade exit %d", r.code)
	}
	if !strings.Contains(read(t, ag), "Deploy on Fridays only.") {
		t.Error("user text outside the markers was lost")
	}
}

func TestUpgradeDoesNotClobberDriftedRegion(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")

	// Edit INSIDE the managed region: inject a sentinel right after the start marker.
	c := read(t, ag)
	edited := startMarker.ReplaceAllStringFunc(c, func(m string) string { return m + "\nDRIFT-SENTINEL (my edit)" })
	if edited == c {
		t.Fatal("could not locate start marker to simulate drift")
	}
	os.WriteFile(ag, []byte(edited), 0o644)

	r := run(t, tmp, "upgrade")
	if !strings.Contains(read(t, ag), "DRIFT-SENTINEL (my edit)") {
		t.Error("hand-edited managed region was clobbered")
	}
	if !regexp.MustCompile(`(?i)edited|drift`).MatchString(r.stdout) {
		t.Error("no drift warning printed")
	}
	if !exists(ag + ".specflow-new") {
		t.Error("no .specflow-new sidecar written for drift")
	}
}

// A lost/corrupt baseline (no recorded hash for a file) must be treated as drift, never overwritten.
func TestUpgradeNoBaselineTreatedAsDrift(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")
	stampPath := filepath.Join(tmp, "specflow/config.json")

	// Simulate a lost baseline: drop the managed map from the stamp.
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, stampPath)), &stamp)
	delete(stamp, "managed")
	b, _ := json.MarshalIndent(stamp, "", "  ")
	os.WriteFile(stampPath, b, 0o644)

	// Edit inside the region; with no baseline, upgrade must NOT overwrite it.
	c := read(t, ag)
	edited := startMarker.ReplaceAllStringFunc(c, func(m string) string { return m + "\nNO-BASELINE-SENTINEL" })
	os.WriteFile(ag, []byte(edited), 0o644)

	run(t, tmp, "upgrade")
	if !strings.Contains(read(t, ag), "NO-BASELINE-SENTINEL") {
		t.Error("upgrade overwrote a region with no baseline — should treat as drift")
	}
	if !exists(ag + ".specflow-new") {
		t.Error("no .specflow-new sidecar for the no-baseline case")
	}
}

func TestUpgradeMigratesPreMarkerFile(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")

	stripped := regexp.MustCompile(`(?s)<!--\s*specflow:(start|end)\b.*?-->`).ReplaceAllString(read(t, ag), "")
	os.WriteFile(ag, []byte(stripped), 0o644)

	r := run(t, tmp, "upgrade")
	if r.code != 0 {
		t.Fatalf("upgrade exit %d", r.code)
	}
	if !exists(ag + ".specflow-bak") {
		t.Error("no .specflow-bak backup on migration")
	}
	if !startMarker.MatchString(read(t, ag)) {
		t.Error("markers not added on migration")
	}
}

func TestUpgradeCanonicalizesBareMarkers(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")

	downgraded := startMarker.ReplaceAllString(read(t, ag), "<!-- specflow:start -->") + "\n## team note\nkeep me\n"
	os.WriteFile(ag, []byte(downgraded), 0o644)

	r := run(t, tmp, "upgrade")
	if r.code != 0 {
		t.Fatalf("upgrade exit %d", r.code)
	}
	out := read(t, ag)
	if !strings.Contains(out, "do not edit inside these markers") {
		t.Error("marker note not re-applied on canonicalize")
	}
	if !strings.Contains(out, "keep me") {
		t.Error("outside text lost during canonicalize")
	}
	if exists(ag + ".specflow-bak") {
		t.Error("unexpected backup on a clean canonicalize")
	}
}

func TestPerAgentFilesAreManaged(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude,cursor")

	// Installed agents' instruction files carry markers and a recorded baseline.
	for _, f := range []string{"CLAUDE.md", ".cursor/rules/specflow.mdc"} {
		if !startMarker.MatchString(read(t, filepath.Join(tmp, f))) {
			t.Errorf("%s lacks region markers", f)
		}
	}
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp)
	managed, _ := stamp["managed"].(map[string]any)
	for _, rel := range []string{"CLAUDE.md", ".cursor/rules/specflow.mdc"} {
		if h, _ := managed[rel].(string); h == "" {
			t.Errorf("no managed baseline for %s", rel)
		}
	}
	// An uninstalled agent's file is not in the managed set.
	if _, ok := managed[".github/copilot-instructions.md"]; ok {
		t.Error("uninstalled copilot file recorded as managed")
	}
}

func TestUpgradeRefreshesPerAgentRegion(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	cl := filepath.Join(tmp, "CLAUDE.md")

	// Downgrade the marker wording + add outside text; the region content (and so its baseline hash)
	// is unchanged, so a clean upgrade must re-canonicalize the CLAUDE.md region — proving per-agent
	// files go through the managed refresh path — while keeping the outside text.
	downgraded := startMarker.ReplaceAllString(read(t, cl), "<!-- specflow:start -->") + "\n## my notes\nkeep me\n"
	os.WriteFile(cl, []byte(downgraded), 0o644)

	r := run(t, tmp, "upgrade")
	if r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	out := read(t, cl)
	if !strings.Contains(out, "do not edit inside these markers") {
		t.Error("CLAUDE.md marker note not re-applied — per-agent region not refreshed")
	}
	if !strings.Contains(out, "keep me") {
		t.Error("outside text in CLAUDE.md lost on upgrade")
	}
}

func TestUpgradeDriftProtectsPerAgentRegion(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	cl := filepath.Join(tmp, "CLAUDE.md")

	edited := startMarker.ReplaceAllStringFunc(read(t, cl), func(m string) string { return m + "\nDRIFT-IN-CLAUDE" })
	os.WriteFile(cl, []byte(edited), 0o644)

	run(t, tmp, "upgrade")
	if !strings.Contains(read(t, cl), "DRIFT-IN-CLAUDE") {
		t.Error("hand-edited CLAUDE.md region was clobbered")
	}
	if !exists(cl + ".specflow-new") {
		t.Error("no .specflow-new sidecar for drifted CLAUDE.md")
	}
}

func TestSubcommandHelp(t *testing.T) {
	tmp := newRepo(t)
	ri := run(t, tmp, "init", "--help")
	if ri.code != 0 {
		t.Fatalf("init --help exit %d", ri.code)
	}
	if !strings.Contains(ri.stdout, "specflow init") || !regexp.MustCompile(`(?i)--agents`).MatchString(ri.stdout) {
		t.Error("init --help did not describe the init command")
	}
	// Help must not install anything.
	if exists(filepath.Join(tmp, "AGENTS.md")) || exists(filepath.Join(tmp, "specflow/config.json")) {
		t.Error("init --help wrote files — help must not install")
	}

	ru := run(t, tmp, "upgrade", "-h")
	if ru.code != 0 {
		t.Fatalf("upgrade -h exit %d", ru.code)
	}
	if !strings.Contains(ru.stdout, "specflow upgrade") || !regexp.MustCompile(`(?i)non-destructive`).MatchString(ru.stdout) {
		t.Error("upgrade -h did not describe the upgrade command")
	}
}

func TestVerifyPassesOnFreshInit(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	r := run(t, tmp, "verify") // working tree, uncommitted — must pass
	if r.code != 0 {
		t.Fatalf("verify on a fresh install exit %d: %s", r.code, r.stdout)
	}
	if !regexp.MustCompile(`(?i)all good|installed`).MatchString(r.stdout) {
		t.Errorf("verify did not report a clean install: %s", r.stdout)
	}
}

func TestVerifyFailsOnMissingTier1(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	os.Remove(filepath.Join(tmp, "specflow/procedures/claim-batch.md"))
	r := run(t, tmp, "verify")
	if r.code == 0 {
		t.Error("verify should exit non-zero when a Tier-1 file is missing")
	}
	if !strings.Contains(r.stdout, "claim-batch.md") {
		t.Errorf("verify did not name the missing file: %s", r.stdout)
	}
}

func TestVerifyWarnsOnDrift(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")
	edited := startMarker.ReplaceAllStringFunc(read(t, ag), func(m string) string { return m + "\nDRIFT" })
	os.WriteFile(ag, []byte(edited), 0o644)
	r := run(t, tmp, "verify")
	if r.code != 0 {
		t.Errorf("drift is a warning, not a hard failure; got exit %d", r.code)
	}
	if !regexp.MustCompile(`(?i)drift|edited`).MatchString(r.stdout) {
		t.Errorf("verify did not warn about drift: %s", r.stdout)
	}
}

func TestVerifyBatchStub(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	r := run(t, tmp, "verify", "--batch")
	if r.code != 0 {
		t.Fatalf("verify --batch exit %d", r.code)
	}
	if !regexp.MustCompile(`(?i)later release|Batch E`).MatchString(r.stdout) {
		t.Errorf("verify --batch should be stubbed: %s", r.stdout)
	}
}

func TestVerifyNotInstalled(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "verify")
	if r.code == 0 {
		t.Error("verify in an uninitialized repo should exit non-zero")
	}
	if !regexp.MustCompile(`(?i)not installed`).MatchString(r.stdout) {
		t.Errorf("verify did not say 'not installed': %s", r.stdout)
	}
}

// specOnlyOmitted / specOnlyKept are the files the two install modes differ on.
var (
	specOnlyOmitted = []string{
		"BUILD_QUEUE.md", "CLAIMS.md",
		"specflow/procedures/claim-batch.md", "specflow/procedures/finish-batch.md",
		"specflow/procedures/prune-ledgers.md",
		".claude/skills/claim-batch/SKILL.md", ".claude/skills/finish-batch/SKILL.md",
		".claude/skills/prune-ledgers/SKILL.md",
		".claude/hooks/specflow-handoff-reminder.sh",
		"specflow/history/BUILD_QUEUE_DONE.md", "specflow/history/CLAIMS_DONE.md",
	}
	specOnlyKept = []string{
		"AGENTS.md", "spec/README.md", "specflow/config.json",
		"specflow/procedures/spec-edit.md", ".claude/skills/spec-edit/SKILL.md", "CLAUDE.md",
	}
	// composeTag finds any leftover section-composition scaffolding in a rendered file.
	composeTag = regexp.MustCompile(`specflow:(full-only|spec-only)`)
	// batchProse catches queue/claim language that names no identifier, so kit.QueueTokens can't see
	// it — e.g. spec/README.md's "For an agent claiming a batch: read the queue entry first". Kept to
	// the test side: verify scans managed regions where a literal-token match is the precise rule,
	// while this guards the wording specflow authors into files it then hands to the user.
	batchProse = regexp.MustCompile(`(?i)claiming a batch|the queue entry|claimed in git|split into batches|new batch|wrapping up a batch`)
)

func TestInitSpecOnlyOmitsBatchMachinery(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "init", "--agents=claude", "--spec-only")
	if r.code != 0 {
		t.Fatalf("init --spec-only exit %d: %s", r.code, r.stderr)
	}
	for _, f := range specOnlyOmitted {
		if exists(filepath.Join(tmp, f)) {
			t.Errorf("spec-only install wrote %s — should omit the batch/claim machinery", f)
		}
	}
	for _, f := range specOnlyKept {
		if !exists(filepath.Join(tmp, f)) {
			t.Errorf("spec-only install missing %s", f)
		}
	}
	// The step-6 hook backstops a boundary spec-only doesn't have, so its notice must stay silent.
	if strings.Contains(r.stdout, "activate the step-6 handoff hook") {
		t.Error("spec-only init printed the step-6 hook notice")
	}
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp)
	cfg, _ := stamp["config"].(map[string]any)
	if m, _ := cfg["mode"].(string); m != "spec-only" {
		t.Errorf("config.mode = %q, want spec-only", m)
	}
	// The managed baseline must cover the rendered AGENTS.md / spec-edit.md but not the omitted procedures.
	managed, _ := stamp["managed"].(map[string]any)
	if h, _ := managed["AGENTS.md"].(string); h == "" {
		t.Error("no managed baseline for AGENTS.md in spec-only")
	}
	if _, ok := managed["specflow/procedures/claim-batch.md"]; ok {
		t.Error("omitted claim-batch.md recorded as managed in spec-only")
	}
}

func TestSpecOnlyManagedFilesCarryNoBatchReferences(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude", "--spec-only"); r.code != 0 {
		t.Fatalf("init --spec-only exit %d: %s", r.code, r.stderr)
	}
	ag := read(t, filepath.Join(tmp, "AGENTS.md"))
	se := read(t, filepath.Join(tmp, "specflow/procedures/spec-edit.md"))
	for _, f := range []struct{ name, body string }{{"AGENTS.md", ag}, {"spec-edit.md", se}} {
		for _, banned := range []string{"BUILD_QUEUE", "CLAIMS.md", "claim-batch", "finish-batch", "## The work queue"} {
			if strings.Contains(f.body, banned) {
				t.Errorf("spec-only %s references batch/claim machinery (%q)", f.name, banned)
			}
		}
		if composeTag.MatchString(f.body) {
			t.Errorf("spec-only %s carries leftover section-composition tags", f.name)
		}
	}
	// The spec discipline must survive: AGENTS.md still wires the spec-edit procedure.
	if !strings.Contains(ag, "spec-edit") || !strings.Contains(ag, "## File ownership") {
		t.Error("spec-only AGENTS.md lost the spec discipline content")
	}
}

// A spec-only install must not name the machinery it deliberately omits, in *any* generated file —
// not just the two that happen to be section-composed. The pre-existing checks assert which files
// spec-only omits, never what the kept files say, which is how full-mode prose shipped into every
// adapter unnoticed. This walks the whole install instead of naming files, so a newly added
// template is covered the day it lands rather than the day someone remembers to list it.
func TestSpecOnlyInstallNamesNoOmittedMachinery(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--all", "--spec-only"); r.code != 0 {
		t.Fatalf("init --all --spec-only exit %d: %s", r.code, r.stderr)
	}
	err := filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(tmp, path)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(b)
		for _, tok := range kit.QueueTokens {
			if strings.Contains(body, tok) {
				t.Errorf("spec-only install: %s names %q, which this mode does not install", rel, tok)
			}
		}
		if m := batchProse.FindString(body); m != "" {
			t.Errorf("spec-only install: %s uses batch/claim language (%q)", rel, m)
		}
		if composeTag.MatchString(body) {
			t.Errorf("spec-only install: %s carries leftover section-composition tags", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// verify must report a mode leak as a Tier-1 problem. Without this, the check is invisible: a
// leaking install reports "All good" because every region still matches its own baseline hash.
func TestVerifyCatchesModeLeak(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude", "--spec-only"); r.code != 0 {
		t.Fatalf("init --spec-only exit %d: %s", r.code, r.stderr)
	}
	if r := run(t, tmp, "verify"); r.code != 0 {
		t.Fatalf("verify on a clean spec-only install should pass, got %d: %s", r.code, r.stdout)
	}
	// Re-stamp a full-mode CLAUDE.md region as if it were the spec-only rendering: the baseline is
	// recomputed over it, so hash-checking alone still says clean.
	claudePath := filepath.Join(tmp, "CLAUDE.md")
	body := read(t, claudePath)
	leaked := strings.Replace(body, "# CLAUDE.md", "# CLAUDE.md\n\nBefore starting any new batch → `claim-batch`.", 1)
	if leaked == body {
		t.Fatal("could not inject a leak into CLAUDE.md")
	}
	os.WriteFile(claudePath, []byte(leaked), 0o644)
	r := run(t, tmp, "verify")
	if r.code == 0 {
		t.Errorf("verify passed on a spec-only install naming claim-batch: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "claim-batch") {
		t.Errorf("verify did not name the leaked token: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "mode: spec-only") {
		t.Errorf("verify did not surface the install mode: %s", r.stdout)
	}
}

// An existing spec-only install must be fixable by `upgrade`, not only by re-init. Managed regions
// refresh on their own, but the skill stubs are non-managed create-once files that upgrade would
// otherwise leave stale forever — and a stale spec-edit stub is the worst case, since its YAML
// description loads into every session's skill listing. Simulates a leaking legacy install by
// writing full-mode content over a spec-only one.
func TestUpgradeRepairsLegacySpecOnlyLeak(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--all", "--spec-only"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	stub := filepath.Join(tmp, ".claude/skills/spec-edit/SKILL.md")
	legacy := "---\nname: spec-edit\ndescription: Covers size-watch, and propagation to BUILD_QUEUE.md.\n---\n\nUpdate `spec/` and `BUILD_QUEUE.md` — never put claim state in the queue.\n"
	if err := os.WriteFile(stub, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := run(t, tmp, "verify"); r.code == 0 {
		t.Errorf("verify passed with a leaking non-managed skill stub: %s", r.stdout)
	}
	if r := run(t, tmp, "upgrade"); r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	got := read(t, stub)
	for _, tok := range kit.QueueTokens {
		if strings.Contains(got, tok) {
			t.Errorf("upgrade left %q in the spec-edit stub:\n%s", tok, got)
		}
	}
	if r := run(t, tmp, "verify"); r.code != 0 {
		t.Errorf("verify still failing after upgrade: %s", r.stdout)
	}
}

// The repair above keys off a mode leak, so it must never fire in full mode — a full install's
// stubs legitimately name the queue, and rewriting a user's customized stub would be destructive.
func TestUpgradeLeavesFullModeStubsAlone(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	stub := filepath.Join(tmp, ".claude/skills/spec-edit/SKILL.md")
	custom := read(t, stub) + "\nMy own note about BUILD_QUEUE.md.\n"
	if err := os.WriteFile(stub, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := run(t, tmp, "upgrade"); r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	if got := read(t, stub); got != custom {
		t.Errorf("upgrade overwrote a customized full-mode stub:\n%s", got)
	}
}

func TestFullModeManagedFilesAreComplete(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	ag := read(t, filepath.Join(tmp, "AGENTS.md"))
	// Full mode keeps the batch/claim sections...
	for _, want := range []string{"## The work queue", "BUILD_QUEUE.md", "CLAIMS.md", "batch-N", "claim-batch.md"} {
		if !strings.Contains(ag, want) {
			t.Errorf("full-mode AGENTS.md missing %q", want)
		}
	}
	// ...with the scaffolding tags fully rendered away.
	if composeTag.MatchString(ag) {
		t.Error("full-mode AGENTS.md carries leftover section-composition tags")
	}
	if composeTag.MatchString(read(t, filepath.Join(tmp, "specflow/procedures/spec-edit.md"))) {
		t.Error("full-mode spec-edit.md carries leftover section-composition tags")
	}
}

func TestSpecOnlyUpgradeStaysClean(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--spec-only")
	r := run(t, tmp, "upgrade")
	if r.code != 0 {
		t.Fatalf("spec-only upgrade exit %d: %s", r.code, r.stderr)
	}
	if regexp.MustCompile(`(?i)drift|edited`).MatchString(r.stdout) {
		t.Errorf("spec-only upgrade reported drift on a pristine install: %s", r.stdout)
	}
	// Upgrade must not introduce the omitted machinery, and AGENTS.md stays batch-free.
	if exists(filepath.Join(tmp, "specflow/procedures/claim-batch.md")) {
		t.Error("upgrade added claim-batch.md to a spec-only install")
	}
	if strings.Contains(read(t, filepath.Join(tmp, "AGENTS.md")), "## The work queue") {
		t.Error("upgrade re-rendered spec-only AGENTS.md with batch sections")
	}
	for _, f := range []string{"AGENTS.md", "specflow/procedures/spec-edit.md"} {
		if exists(filepath.Join(tmp, f+".specflow-new")) {
			t.Errorf("spec-only upgrade wrote a drift sidecar for %s", f)
		}
	}
}

func TestSpecOnlyVerifyPasses(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--spec-only")
	r := run(t, tmp, "verify")
	if r.code != 0 {
		t.Fatalf("verify on a spec-only install exit %d: %s", r.code, r.stdout)
	}
	if strings.Contains(r.stdout, "claim-batch") || strings.Contains(r.stdout, "finish-batch") {
		t.Errorf("verify flagged the (intentionally absent) batch procedures in spec-only: %s", r.stdout)
	}
}

// stampAgents / stampManaged pull the config.agents CSV and the managed-baseline map from a repo's
// stamp — small readers so the add-agent tests stay readable.
func stampAgents(t *testing.T, dir string) string {
	t.Helper()
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(dir, "specflow/config.json"))), &stamp)
	cfg, _ := stamp["config"].(map[string]any)
	s, _ := cfg["agents"].(string)
	return s
}

func stampManaged(t *testing.T, dir string) map[string]any {
	t.Helper()
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(dir, "specflow/config.json"))), &stamp)
	m, _ := stamp["managed"].(map[string]any)
	return m
}

func TestAddAgentAddsAdapterStampAndManaged(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	r := run(t, tmp, "add-agent", "cursor")
	if r.code != 0 {
		t.Fatalf("add-agent exit %d: %s", r.code, r.stderr)
	}
	if !exists(filepath.Join(tmp, ".cursor/rules/specflow.mdc")) {
		t.Error("add-agent did not write the cursor adapter")
	}
	if got := stampAgents(t, tmp); got != "claude,cursor" {
		t.Errorf("config.agents = %q, want claude,cursor", got)
	}
	if h, _ := stampManaged(t, tmp)[".cursor/rules/specflow.mdc"].(string); h == "" {
		t.Error("new agent's instruction file not recorded in managed baseline")
	}
	if !strings.Contains(r.stdout, "git diff") {
		t.Error("add-agent did not print the review/commit handoff")
	}
	// add-agent never commits.
	if c := strings.TrimSpace(gitOut(t, tmp, "rev-list", "--all", "--count")); c != "0" {
		t.Errorf("add-agent created commits (count %q) — it must never commit", c)
	}
}

func TestAddAgentMultipleAndAlreadyPresent(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	if r := run(t, tmp, "add-agent", "cursor", "bob"); r.code != 0 {
		t.Fatalf("add-agent exit %d: %s", r.code, r.stderr)
	}
	if got := stampAgents(t, tmp); got != "claude,cursor,bob" {
		t.Errorf("config.agents = %q, want claude,cursor,bob", got)
	}
	// Re-adding an installed agent is a no-op that leaves the list unchanged.
	r := run(t, tmp, "add-agent", "cursor")
	if r.code != 0 {
		t.Fatalf("re-add exit %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "already installed") {
		t.Errorf("re-adding cursor should report already installed: %s", r.stdout)
	}
	if got := stampAgents(t, tmp); got != "claude,cursor,bob" {
		t.Errorf("config.agents changed on no-op re-add = %q", got)
	}
}

func TestAddAgentBrownfieldInjectsInstructionFile(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=cursor"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	// A pre-existing CLAUDE.md with user content — add-agent must inject the region, not clobber it.
	mustWrite(t, filepath.Join(tmp, "CLAUDE.md"), "# My notes\nuse pnpm.\n")
	r := run(t, tmp, "add-agent", "claude")
	if r.code != 0 {
		t.Fatalf("add-agent exit %d: %s", r.code, r.stderr)
	}
	cl := read(t, filepath.Join(tmp, "CLAUDE.md"))
	if !startMarker.MatchString(cl) {
		t.Error("add-agent did not inject a region into the existing CLAUDE.md")
	}
	if !strings.Contains(cl, "use pnpm.") {
		t.Error("add-agent lost existing CLAUDE.md content on injection")
	}
	if strings.Index(cl, "specflow:start") > strings.Index(cl, "use pnpm.") {
		t.Error("region not injected above the existing content")
	}
	if h, _ := stampManaged(t, tmp)["CLAUDE.md"].(string); h == "" {
		t.Error("injected CLAUDE.md not recorded in managed baseline")
	}
	if !strings.Contains(r.stdout, "injected") {
		t.Errorf("add-agent did not report the injection: %s", r.stdout)
	}
}

func TestAddAgentLeavesAlreadyWiredFile(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=cursor"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	// A CLAUDE.md that already points at AGENTS.md (no region) — add-agent must leave it as-is.
	mustWrite(t, filepath.Join(tmp, "CLAUDE.md"), "# Notes\nSee AGENTS.md for the protocol.\n")
	r := run(t, tmp, "add-agent", "claude")
	if r.code != 0 {
		t.Fatalf("add-agent exit %d: %s", r.code, r.stderr)
	}
	if startMarker.MatchString(read(t, filepath.Join(tmp, "CLAUDE.md"))) {
		t.Error("add-agent injected a region into a CLAUDE.md that already references AGENTS.md")
	}
	if !strings.Contains(r.stdout, "already wired") {
		t.Errorf("add-agent did not report the file as already wired: %s", r.stdout)
	}
	// The agent's owned skills are still installed even though its instruction file was left alone.
	if !exists(filepath.Join(tmp, ".claude/skills/spec-edit/SKILL.md")) {
		t.Error("add-agent skipped the agent's skill files")
	}
}

func TestAddAgentSpecOnlyOmitsBatchSkills(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=cursor", "--spec-only"); r.code != 0 {
		t.Fatalf("init --spec-only exit %d: %s", r.code, r.stderr)
	}
	if r := run(t, tmp, "add-agent", "claude"); r.code != 0 {
		t.Fatalf("add-agent exit %d: %s", r.code, r.stderr)
	}
	// A spec-only install must not gain the claim/finish skills, only spec-edit.
	if !exists(filepath.Join(tmp, ".claude/skills/spec-edit/SKILL.md")) {
		t.Error("add-agent (spec-only) missing the spec-edit skill")
	}
	for _, f := range []string{".claude/skills/claim-batch/SKILL.md", ".claude/skills/finish-batch/SKILL.md"} {
		if exists(filepath.Join(tmp, f)) {
			t.Errorf("add-agent (spec-only) wrote %s — batch skills must be omitted", f)
		}
	}
}

func TestAddAgentUnknownAndNotInstalled(t *testing.T) {
	// Unknown agent in an installed repo → clean error, non-zero exit, nothing written.
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	r := run(t, tmp, "add-agent", "bogus")
	if r.code == 0 {
		t.Error("add-agent with an unknown agent should exit non-zero")
	}
	if !strings.Contains(r.stdout, "Unknown agent") {
		t.Errorf("no unknown-agent message: %s", r.stdout)
	}
	if stampAgents(t, tmp) != "claude" {
		t.Error("unknown agent altered config.agents")
	}
	// add-agent before init → guarded.
	fresh := newRepo(t)
	r = run(t, fresh, "add-agent", "claude")
	if r.code == 0 {
		t.Error("add-agent without an install should exit non-zero")
	}
	if !strings.Contains(r.stdout, "No specflow install") {
		t.Errorf("no not-installed message: %s", r.stdout)
	}
}

func TestAddAgentVerifyPassesAfterAdd(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	if r := run(t, tmp, "add-agent", "cursor"); r.code != 0 {
		t.Fatalf("add-agent exit %d: %s", r.code, r.stderr)
	}
	r := run(t, tmp, "verify")
	if r.code != 0 {
		t.Fatalf("verify exit %d after add-agent: %s", r.code, r.stdout)
	}
	if !strings.Contains(r.stdout, ".cursor/rules/specflow.mdc") {
		t.Errorf("verify did not account for the added agent's file: %s", r.stdout)
	}
}

func TestAddAgentHelp(t *testing.T) {
	tmp := t.TempDir()
	r := run(t, tmp, "add-agent", "--help")
	if r.code != 0 {
		t.Fatalf("add-agent --help exit %d", r.code)
	}
	if !strings.Contains(r.stdout, "specflow add-agent") || !strings.Contains(r.stdout, "Never commits") {
		t.Errorf("add-agent --help missing usage: %s", r.stdout)
	}
}

// setStampVersion rewrites the stamp's kitVersion — used to force the version-mismatch path.
func setStampVersion(t *testing.T, dir, v string) {
	t.Helper()
	p := filepath.Join(dir, "specflow/config.json")
	var stamp map[string]any
	if err := json.Unmarshal([]byte(read(t, p)), &stamp); err != nil {
		t.Fatal(err)
	}
	stamp["kitVersion"] = v
	b, _ := json.MarshalIndent(stamp, "", "  ")
	mustWrite(t, p, string(b)+"\n")
}

// TestInitCheckFlagRecorded covers --check=: the value lands in config.check verbatim and status
// surfaces it, so an agent has one command to run instead of rediscovering the repo's check parts.
func TestInitCheckFlagRecorded(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude", "--check=npm run verify"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp)
	cfg, _ := stamp["config"].(map[string]any)
	if got, _ := cfg["check"].(string); got != "npm run verify" {
		t.Errorf("config.check = %q, want %q", got, "npm run verify")
	}
	if r := run(t, tmp, "status"); !strings.Contains(r.stdout, "npm run verify") {
		t.Errorf("status did not surface the check command:\n%s", r.stdout)
	}
}

// TestInitCheckOptional: skipping the prompt is a supported answer, not a half-configured install.
// The key is present and empty, and status says so rather than pretending a check exists.
func TestInitCheckOptional(t *testing.T) {
	tmp := newRepo(t)
	// Non-interactive with no --check at all.
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	var stamp map[string]any
	if err := json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}
	cfg, _ := stamp["config"].(map[string]any)
	got, ok := cfg["check"].(string)
	if !ok || got != "" {
		t.Errorf("config.check = %#v, want an empty string", cfg["check"])
	}
	if r := run(t, tmp, "status"); !strings.Contains(r.stdout, "not set") {
		t.Errorf("status should report an unset check:\n%s", r.stdout)
	}
}

// TestInitCheckEscapesJSON guards the stamp against a check command containing quotes or
// backslashes — the template is text-substituted, so an unescaped value would produce a
// config.json that every later command fails to parse.
func TestInitCheckEscapesJSON(t *testing.T) {
	tmp := newRepo(t)
	cmdStr := `sh -c "make check && echo \"ok\""`
	if r := run(t, tmp, "init", "--agents=claude", "--check="+cmdStr); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	var stamp map[string]any
	if err := json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp); err != nil {
		t.Fatalf("config.json is not valid JSON after a quoted check command: %v", err)
	}
	cfg, _ := stamp["config"].(map[string]any)
	if got, _ := cfg["check"].(string); got != cmdStr {
		t.Errorf("config.check = %q, want %q", got, cmdStr)
	}
	// status must still work on that install.
	if r := run(t, tmp, "status"); r.code != 0 {
		t.Fatalf("status exit %d on an install with a quoted check: %s", r.code, r.stderr)
	}
}

// TestInitCheckPromptedInteractively drives the prompt itself: agent pick, then the check answer.
func TestInitCheckPromptedInteractively(t *testing.T) {
	tmp := newRepo(t)
	r := runStdin(t, tmp, "\nmake check\n", "init")
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "check command") {
		t.Errorf("init never asked for the check command:\n%s", r.stdout)
	}
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp)
	cfg, _ := stamp["config"].(map[string]any)
	if got, _ := cfg["check"].(string); got != "make check" {
		t.Errorf("config.check = %q, want %q", got, "make check")
	}
}

// TestStatusCheckMissingKey covers an install predating config.check: the key is absent, not empty.
// Status must degrade to "not set" rather than erroring, since upgrade doesn't rewrite the config.
func TestStatusCheckMissingKey(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	p := filepath.Join(tmp, "specflow/config.json")
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, p)), &stamp)
	cfg, _ := stamp["config"].(map[string]any)
	delete(cfg, "check")
	b, _ := json.MarshalIndent(stamp, "", "  ")
	mustWrite(t, p, string(b))
	r := run(t, tmp, "status")
	if r.code != 0 {
		t.Fatalf("status exit %d on a pre-check install: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "not set") {
		t.Errorf("status should report an unset check for a legacy install:\n%s", r.stdout)
	}
}

func TestStatusFreshInstall(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude,cursor"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	r := run(t, tmp, "status")
	if r.code != 0 {
		t.Fatalf("status exit %d: %s", r.code, r.stderr)
	}
	for _, want := range []string{"full", "claude, cursor", "commit=agent", "push=agent", "un-done batch", "none active"} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("status missing %q:\n%s", want, r.stdout)
		}
	}
	// A pristine install reports no drift.
	if !regexp.MustCompile(`(?m)drift\s+none`).MatchString(r.stdout) {
		t.Errorf("fresh install should report drift none:\n%s", r.stdout)
	}
}

func TestStatusReportsActiveClaims(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	mustWrite(t, filepath.Join(tmp, "CLAIMS.md"), `# Agent claims

## In progress

### Batch 9 — some work
- Owner: alice
- Started: 2026-01-01 00:00

### Batch 7 — handoff item
- Owner: none

## Completed
`)
	r := run(t, tmp, "status")
	if r.code != 0 {
		t.Fatalf("status exit %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "2 active") {
		t.Errorf("status did not count 2 active claims:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Batch 9 — some work") || !strings.Contains(r.stdout, "alice") {
		t.Errorf("status did not list the owned claim:\n%s", r.stdout)
	}
	// Owner: none renders as unassigned, not the literal "none".
	if !strings.Contains(r.stdout, "unassigned") {
		t.Errorf("status did not render an ownerless claim as unassigned:\n%s", r.stdout)
	}
}

func TestStatusFlagsDrift(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	ag := filepath.Join(tmp, "AGENTS.md")
	edited := startMarker.ReplaceAllStringFunc(read(t, ag), func(m string) string { return m + "\nDRIFT" })
	os.WriteFile(ag, []byte(edited), 0o644)
	r := run(t, tmp, "status")
	if r.code != 0 {
		t.Fatalf("status should not fail on drift; exit %d: %s", r.code, r.stderr)
	}
	if !regexp.MustCompile(`(?i)edited since install`).MatchString(r.stdout) || !strings.Contains(r.stdout, "AGENTS.md") {
		t.Errorf("status did not flag AGENTS.md drift:\n%s", r.stdout)
	}
}

func TestStatusVersionMismatch(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	setStampVersion(t, tmp, "0.0.9")
	r := run(t, tmp, "status")
	if r.code != 0 {
		t.Fatalf("status exit %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "0.0.9") || !strings.Contains(r.stdout, "upgrade") {
		t.Errorf("status did not surface the stamp/binary version gap + upgrade hint:\n%s", r.stdout)
	}
}

func TestStatusSpecOnlyQueueNA(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--spec-only", "--agents=cursor"); r.code != 0 {
		t.Fatalf("init --spec-only exit %d: %s", r.code, r.stderr)
	}
	r := run(t, tmp, "status")
	if r.code != 0 {
		t.Fatalf("status exit %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "spec-only") {
		t.Errorf("status did not report spec-only mode:\n%s", r.stdout)
	}
	if !regexp.MustCompile(`(?i)queue\s+n/a`).MatchString(r.stdout) {
		t.Errorf("spec-only status should mark the queue n/a:\n%s", r.stdout)
	}
}

func TestStatusNotInstalled(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "status")
	if r.code == 0 {
		t.Error("status in an uninitialized repo should exit non-zero")
	}
	if !regexp.MustCompile(`(?i)not installed`).MatchString(r.stdout) {
		t.Errorf("status did not say 'not installed':\n%s", r.stdout)
	}
}

func TestStatusHelp(t *testing.T) {
	tmp := t.TempDir()
	r := run(t, tmp, "status", "--help")
	if r.code != 0 {
		t.Fatalf("status --help exit %d", r.code)
	}
	if !strings.Contains(r.stdout, "specflow status") {
		t.Errorf("status --help missing usage:\n%s", r.stdout)
	}
}

func TestInitDryRunWritesNothing(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "init", "--dry-run")
	if r.code != 0 {
		t.Fatalf("init --dry-run exit %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "would create") || !strings.Contains(r.stdout, "AGENTS.md") {
		t.Errorf("init --dry-run did not preview the file plan:\n%s", r.stdout)
	}
	// Nothing on disk: not even the stamp.
	for _, f := range []string{"specflow/config.json", "AGENTS.md", "BUILD_QUEUE.md", "CLAUDE.md"} {
		if exists(filepath.Join(tmp, f)) {
			t.Errorf("init --dry-run wrote %s — it must write nothing", f)
		}
	}
	// git tree stays empty too.
	if c := strings.TrimSpace(gitOut(t, tmp, "status", "--porcelain")); c != "" {
		t.Errorf("init --dry-run dirtied the tree:\n%s", c)
	}
}

func TestInitDryRunPreviewsBrownfieldInjection(t *testing.T) {
	tmp := newRepo(t)
	mustWrite(t, filepath.Join(tmp, "AGENTS.md"), "# ours\nkeep me\n")
	r := run(t, tmp, "init", "--dry-run", "--agents=claude")
	if r.code != 0 {
		t.Fatalf("init --dry-run exit %d: %s", r.code, r.stderr)
	}
	if !regexp.MustCompile(`(?s)would inject.*AGENTS\.md`).MatchString(r.stdout) {
		t.Errorf("dry-run did not flag the existing AGENTS.md as an injection target:\n%s", r.stdout)
	}
	// The existing file is untouched (no region injected).
	if got := read(t, filepath.Join(tmp, "AGENTS.md")); got != "# ours\nkeep me\n" {
		t.Errorf("init --dry-run modified an existing file:\n%s", got)
	}
}

func TestInitDryRunSpecOnlyPreview(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "init", "--dry-run", "--spec-only", "--agents=claude")
	if r.code != 0 {
		t.Fatalf("init --dry-run --spec-only exit %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "spec-only") {
		t.Errorf("dry-run did not report spec-only mode:\n%s", r.stdout)
	}
	// The preview omits the batch/claim machinery, just like a real spec-only install.
	if strings.Contains(r.stdout, "BUILD_QUEUE.md") || strings.Contains(r.stdout, "claim-batch") {
		t.Errorf("spec-only dry-run previewed batch/claim files:\n%s", r.stdout)
	}
}

func TestUpgradeDryRunPreviewsWithoutWriting(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	// Drift AGENTS.md (hand-edit inside the region) and strip CLAUDE.md's markers (pre-marker → migrate).
	ag := filepath.Join(tmp, "AGENTS.md")
	os.WriteFile(ag, []byte(startMarker.ReplaceAllStringFunc(read(t, ag), func(m string) string { return m + "\nDRIFT" })), 0o644)
	cl := filepath.Join(tmp, "CLAUDE.md")
	stripped := regexp.MustCompile(`(?s)<!--\s*specflow:(start|end).*?-->`).ReplaceAllString(read(t, cl), "")
	os.WriteFile(cl, []byte(stripped), 0o644)

	before := read(t, filepath.Join(tmp, "specflow/config.json"))
	r := run(t, tmp, "upgrade", "--dry-run")
	if r.code != 0 {
		t.Fatalf("upgrade --dry-run exit %d: %s", r.code, r.stderr)
	}
	if !regexp.MustCompile(`(?s)would NOT overwrite.*AGENTS\.md`).MatchString(r.stdout) {
		t.Errorf("dry-run did not preview the AGENTS.md drift:\n%s", r.stdout)
	}
	if !regexp.MustCompile(`(?s)would migrate.*CLAUDE\.md`).MatchString(r.stdout) {
		t.Errorf("dry-run did not preview the CLAUDE.md migration:\n%s", r.stdout)
	}
	// It must write nothing: no sidecars, and the stamp is byte-identical.
	for _, suffix := range []string{".specflow-new", ".specflow-bak"} {
		if exists(ag+suffix) || exists(cl+suffix) {
			t.Errorf("upgrade --dry-run wrote a %s sidecar", suffix)
		}
	}
	if read(t, filepath.Join(tmp, "specflow/config.json")) != before {
		t.Error("upgrade --dry-run modified the stamp")
	}
}

func TestUpgradeDryRunAlreadyCurrent(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	r := run(t, tmp, "upgrade", "--dry-run")
	if r.code != 0 {
		t.Fatalf("upgrade --dry-run exit %d: %s", r.code, r.stderr)
	}
	if !regexp.MustCompile(`(?i)already current|nothing to refresh`).MatchString(r.stdout) {
		t.Errorf("a fresh install should preview as already current:\n%s", r.stdout)
	}
}

func TestUpgradeDryRunNotInstalled(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "upgrade", "--dry-run")
	if r.code != 0 {
		t.Fatalf("upgrade --dry-run on an empty repo should exit 0 with a notice; got %d", r.code)
	}
	if !regexp.MustCompile(`(?i)no specflow install|not installed`).MatchString(r.stdout) {
		t.Errorf("upgrade --dry-run did not report the missing install:\n%s", r.stdout)
	}
}

func TestVersionAndUnknownCommand(t *testing.T) {
	tmp := t.TempDir()
	for _, flag := range []string{"--version", "-v"} {
		r := run(t, tmp, flag)
		if r.code != 0 {
			t.Errorf("%s exit %d", flag, r.code)
		}
		if !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(strings.TrimSpace(r.stdout)) {
			t.Errorf("%s did not print a version: %q", flag, r.stdout)
		}
	}
	if r := run(t, tmp, "frobnicate"); r.code == 0 {
		t.Error("unknown command should exit non-zero")
	}
}

// TestAgentsMdContentSections locks the shape of the generated AGENTS.md — the document every agent
// reads before working. A refactor that quietly drops a section or the commit-grammar table would
// otherwise pass the file-existence smoke checks.
func TestAgentsMdContentSections(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	ag := read(t, filepath.Join(tmp, "AGENTS.md"))

	// The key sections a reader (and every agent) relies on.
	for _, sec := range []string{
		"## Commit & push authority",
		"## File ownership",
		"## The work queue",
		"## The claims file",
		"## The procedures",
		"## Commit message convention",
		"## Editing rules",
	} {
		if !strings.Contains(ag, sec) {
			t.Errorf("generated AGENTS.md missing section %q", sec)
		}
	}

	// The commit-grammar table: its header plus the grammar rows agents must follow.
	if !strings.Contains(ag, "| Prefix | When to use |") {
		t.Error("AGENTS.md missing the commit-grammar table header")
	}
	for _, cell := range []string{
		"batch-N: <imperative>",
		"meta: claim batch-N (<agent>)",
		"meta: complete batch-N",
		"spec: <change>",
	} {
		if !strings.Contains(ag, cell) {
			t.Errorf("commit-grammar table missing the %q row", cell)
		}
	}

	// The procedures section points at the real specflow/procedures/ paths, not bare filenames.
	for _, p := range []string{
		"specflow/procedures/spec-edit.md",
		"specflow/procedures/claim-batch.md",
		"specflow/procedures/finish-batch.md",
		"specflow/procedures/prune-ledgers.md",
	} {
		if !strings.Contains(ag, p) {
			t.Errorf("AGENTS.md does not reference the procedure path %q", p)
		}
	}
}

// adapterPrimaryFile is the primary (managed instruction) file each agent adapter drops. Restated
// independently of the CLI's own map so a wrong-path regression in either is caught.
var adapterPrimaryFile = map[string]string{
	"claude":      "CLAUDE.md",
	"cursor":      ".cursor/rules/specflow.mdc",
	"copilot":     ".github/copilot-instructions.md",
	"bob":         ".bob/rules/specflow.md",
	"antigravity": ".agents/rules/specflow.md",
}

// TestInitInteractivePicksAgentsByNumber drives the interactive agent picker over piped stdin. The
// menu is claude=1, cursor=2, copilot=3, bob=4, antigravity=5; a fresh repo has no existing files,
// so no injection prompt follows and a single line drives the whole run.
func TestInitInteractivePicksAgentsByNumber(t *testing.T) {
	tmp := newRepo(t)
	r := runStdin(t, tmp, "3,4,5\n", "init")
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	if got := stampAgents(t, tmp); got != "copilot,bob,antigravity" {
		t.Errorf("config.agents = %q, want copilot,bob,antigravity", got)
	}
	for _, k := range []string{"copilot", "bob", "antigravity"} {
		if !exists(filepath.Join(tmp, adapterPrimaryFile[k])) {
			t.Errorf("interactive pick did not write the %s adapter (%s)", k, adapterPrimaryFile[k])
		}
	}
	// The unpicked agents' files must not appear.
	if exists(filepath.Join(tmp, adapterPrimaryFile["cursor"])) {
		t.Error("unpicked cursor adapter leaked in")
	}
	if exists(filepath.Join(tmp, "CLAUDE.md")) {
		t.Error("claude was not picked but CLAUDE.md was written")
	}
}

// TestInitInteractiveAllShortcut covers the picker's "a" (all) branch over piped stdin.
func TestInitInteractiveAllShortcut(t *testing.T) {
	tmp := newRepo(t)
	r := runStdin(t, tmp, "a\n", "init")
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	if got := stampAgents(t, tmp); got != "claude,cursor,copilot,bob,antigravity" {
		t.Errorf("config.agents = %q, want all five agents", got)
	}
}

// TestInitAllFlagWiresEveryAdapter covers `--all` and, with it, the copilot / bob / antigravity
// adapters that the rest of the suite never exercised: each adapter's instruction file is written,
// carries the managed region, and has a baseline hash — and a follow-up upgrade stays a clean no-op.
func TestInitAllFlagWiresEveryAdapter(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "init", "--all")
	if r.code != 0 {
		t.Fatalf("init --all exit %d: %s", r.code, r.stderr)
	}
	if got := stampAgents(t, tmp); got != "claude,cursor,copilot,bob,antigravity" {
		t.Errorf("config.agents = %q, want all five agents", got)
	}
	managed := stampManaged(t, tmp)
	for _, k := range []string{"claude", "cursor", "copilot", "bob", "antigravity"} {
		rel := adapterPrimaryFile[k]
		if !exists(filepath.Join(tmp, rel)) {
			t.Errorf("%s adapter file %s not written", k, rel)
			continue
		}
		if !startMarker.MatchString(read(t, filepath.Join(tmp, rel))) {
			t.Errorf("%s adapter file %s lacks the specflow region markers", k, rel)
		}
		if h, _ := managed[rel].(string); h == "" {
			t.Errorf("%s adapter file %s has no managed baseline hash", k, rel)
		}
	}
	// A later upgrade over the full adapter set is a clean no-op — no drift sidecars.
	if ru := run(t, tmp, "upgrade"); ru.code != 0 {
		t.Fatalf("upgrade after --all exit %d: %s", ru.code, ru.stderr)
	}
	for _, k := range []string{"copilot", "bob", "antigravity"} {
		if exists(filepath.Join(tmp, adapterPrimaryFile[k]+".specflow-new")) {
			t.Errorf("clean upgrade wrote a drift sidecar for the %s adapter", k)
		}
	}
}

// assertResearchFlowShipped checks that a fresh install carries the research-flow convention in all
// three places Batch RF ships it: the AGENTS.md spec-discipline region, the spec-edit procedure, and
// the spec/README.md file map.
func assertResearchFlowShipped(t *testing.T, dir string) {
	t.Helper()
	ag := read(t, filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(ag, "spec/research") || !strings.Contains(ag, "Research notes") {
		t.Errorf("AGENTS.md does not name the research pre-step / spec/research home:\n%s", ag)
	}
	se := read(t, filepath.Join(dir, "specflow/procedures/spec-edit.md"))
	for _, want := range []string{"## Research notes", "spec/research", "Gate-free", "graduate"} {
		if !strings.Contains(se, want) {
			t.Errorf("spec-edit.md research section missing %q", want)
		}
	}
	rd := read(t, filepath.Join(dir, "spec/README.md"))
	if !strings.Contains(rd, "spec/research") || !strings.Contains(rd, "Research notes") {
		t.Errorf("spec/README.md does not document the research/ convention:\n%s", rd)
	}
}

// TestResearchFlowConventionShipped: a full install carries the research-flow convention.
func TestResearchFlowConventionShipped(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	assertResearchFlowShipped(t, tmp)
}

// TestResearchFlowInSpecOnly: research discipline is spec discipline, so a spec-only install inherits
// the convention too — and the research sections stay self-contained (the existing spec-only
// banned-word test guards that they drag in no queue/claim machinery).
func TestResearchFlowInSpecOnly(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude", "--spec-only"); r.code != 0 {
		t.Fatalf("init --spec-only exit %d: %s", r.code, r.stderr)
	}
	assertResearchFlowShipped(t, tmp)
}

// TestClaudeStep6HookInstalledWithNotice: a full claude install drops the handoff-reminder hook,
// prints the opt-in activation notice with the paste block, and carries the adapter relay line.
func TestClaudeStep6HookInstalledWithNotice(t *testing.T) {
	tmp := newRepo(t)
	r := run(t, tmp, "init", "--agents=claude")
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	if !exists(filepath.Join(tmp, ".claude/hooks/specflow-handoff-reminder.sh")) {
		t.Fatal("claude full install missing the step-6 handoff hook script")
	}
	if !strings.Contains(r.stdout, "activate the step-6 handoff hook") {
		t.Error("init did not print the hook activation notice")
	}
	if !strings.Contains(r.stdout, "${CLAUDE_PROJECT_DIR}/.claude/hooks/specflow-handoff-reminder.sh") {
		t.Error("notice did not include the settings.json paste-block command")
	}
	if c := read(t, filepath.Join(tmp, "CLAUDE.md")); !strings.Contains(c, "relay the Claude-Code step-6 handoff hook") {
		t.Error("CLAUDE.md missing the full-only relay instruction")
	}
}

// TestAddAgentClaudeHookNotice: adding claude to a full repo installs the hook and prints the notice
// once; re-adding an already-present claude is a no-op and must not re-print it.
func TestAddAgentClaudeHookNotice(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=copilot"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	r := run(t, tmp, "add-agent", "claude")
	if r.code != 0 {
		t.Fatalf("add-agent exit %d: %s", r.code, r.stderr)
	}
	if !exists(filepath.Join(tmp, ".claude/hooks/specflow-handoff-reminder.sh")) {
		t.Error("add-agent claude did not install the hook script")
	}
	if !strings.Contains(r.stdout, "activate the step-6 handoff hook") {
		t.Error("add-agent claude did not print the hook notice")
	}
	if r2 := run(t, tmp, "add-agent", "claude"); strings.Contains(r2.stdout, "activate the step-6 handoff hook") {
		t.Error("re-adding an already-present claude re-printed the hook notice")
	}
}

// runHook drives the installed hook script with a synthetic PostToolUse payload and returns its
// stdout and exit code.
func runHook(t *testing.T, script, cwd, command string) (string, int) {
	t.Helper()
	payload := fmt.Sprintf(`{"cwd":%q,"tool_input":{"command":%q}}`, cwd, command)
	cmd := exec.Command("bash", script)
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader(payload)
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return out.String(), ee.ExitCode()
		}
		t.Fatalf("run hook: %v (stderr=%s)", err, errb.String())
	}
	return out.String(), 0
}

// TestClaudeStep6HookScriptBehavior exercises the shipped hook: it blocks the loop only when a
// `meta: complete batch-*` commit just landed, and stays silent otherwise. Skipped without jq
// (the hook fails open when jq is absent, so there's nothing deterministic to assert).
func TestClaudeStep6HookScriptBehavior(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed — the hook fails open without it")
	}
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	script := filepath.Join(tmp, ".claude/hooks/specflow-handoff-reminder.sh")

	// A repo whose HEAD subject we control, pointed at by the payload's cwd.
	gd := newRepo(t)
	gitOut(t, gd, "config", "user.email", "t@t.t")
	gitOut(t, gd, "config", "user.name", "t")
	gitOut(t, gd, "commit", "--allow-empty", "-m", "meta: complete batch-CH")

	// Matching: a git commit landed on a `meta: complete batch-*` HEAD → block + reminder.
	out, code := runHook(t, script, gd, "git commit -q -m x")
	if code != 0 {
		t.Fatalf("hook exit %d (want 0), out=%s", code, out)
	}
	var dec struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(out), &dec); err != nil {
		t.Fatalf("hook output is not valid JSON: %v\n%s", err, out)
	}
	if dec.Decision != "block" {
		t.Errorf("decision = %q, want block", dec.Decision)
	}
	if !strings.Contains(dec.Reason, "finish-batch step 6") {
		t.Errorf("block reason missing the step-6 reminder: %q", dec.Reason)
	}

	// Non-commit command → silent, exit 0.
	if o, c := runHook(t, script, gd, "ls -la"); c != 0 || strings.TrimSpace(o) != "" {
		t.Errorf("non-commit command should be silent; exit=%d out=%q", c, o)
	}

	// git commit but HEAD is a claim, not a completion → silent.
	gitOut(t, gd, "commit", "--allow-empty", "-m", "meta: claim batch-X (claude)")
	if o, c := runHook(t, script, gd, "git commit -m y"); c != 0 || strings.TrimSpace(o) != "" {
		t.Errorf("non-completion HEAD should be silent; exit=%d out=%q", c, o)
	}
}

// TestUpgradeAddsMissingAdapterHook: upgrade converges an install that predates a non-managed adapter
// file (the hook) by adding it create-once, but never resurrects a user-deleted base file.
func TestUpgradeAddsMissingAdapterHook(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	hook := filepath.Join(tmp, ".claude/hooks/specflow-handoff-reminder.sh")
	queue := filepath.Join(tmp, "BUILD_QUEUE.md")
	// Simulate an install that predates the hook, and a deliberately-removed base file.
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(queue); err != nil {
		t.Fatal(err)
	}

	r := run(t, tmp, "upgrade")
	if r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	if !exists(hook) {
		t.Error("upgrade did not add the missing adapter hook")
	}
	if !strings.Contains(r.stdout, ".claude/hooks/specflow-handoff-reminder.sh") {
		t.Error("upgrade did not report the added hook")
	}
	if exists(queue) {
		t.Error("upgrade resurrected a user-deleted base file (BUILD_QUEUE.md)")
	}

	// Idempotent: a second upgrade adds nothing.
	if r2 := run(t, tmp, "upgrade"); strings.Contains(r2.stdout, "added:") {
		t.Errorf("second upgrade re-added files: %s", r2.stdout)
	}
}

// The 600-line cap is a stop-and-ask, not a nudge, so the shipped procedure has to carry the four
// parts an agent needs to execute it without the spec in context: the headline listing, the
// single-concern claim, the verbatim read-cost warning, and the waiver format. The warning is
// asserted word-for-word because the user specified those exact words — a paraphrase drops the
// "you're the boss" that makes the ask a real question rather than a lecture.
func assertSizeCapShipped(t *testing.T, dir string) {
	t.Helper()
	raw := read(t, filepath.Join(dir, "specflow/procedures/spec-edit.md"))
	// The prose is hard-wrapped, so collapse whitespace before matching sentence-length strings.
	body := strings.Join(strings.Fields(raw), " ")
	want := []string{
		"600 lines is a hard cap, not a nudge",
		"section headlines",
		"single concern",
		"The bigger a spec file is, the more I read when I need even just a small chunk from it, " +
			"so it's best the file is small in advance. But, you're the boss.",
		"Never ask the user to pick a number",
		"specflow:size-ok",
		"next check at 800",
		"plus 200",
		"`archive.md` and anything under `research/`",
	}
	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("spec-edit.md is missing the size-cap rule %q", w)
		}
	}
	// The old nudge wording and its token gloss were both wrong: it never fired, and 600 lines is
	// roughly 10-11k tokens at this corpus's line length, not 20k.
	for _, stale := range []string{"20k tokens", "consider whether the next bite of content"} {
		if strings.Contains(body, stale) {
			t.Errorf("spec-edit.md still carries the superseded size-watch wording %q", stale)
		}
	}
}

func TestSizeCapShippedFullMode(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	assertSizeCapShipped(t, tmp)
	// The skill description loads into every session, so it is where a stale summary does damage.
	skill := read(t, filepath.Join(tmp, ".claude/skills/spec-edit/SKILL.md"))
	if !strings.Contains(skill, "600-line size cap") || !strings.Contains(skill, "never decide it yourself") {
		t.Error("spec-edit SKILL.md does not summarize the size cap as a stop")
	}
}

// The cap is spec discipline, not batch machinery, so spec-only inherits it whole.
func TestSizeCapShippedSpecOnly(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude", "--spec-only"); r.code != 0 {
		t.Fatalf("init --spec-only exit %d: %s", r.code, r.stderr)
	}
	assertSizeCapShipped(t, tmp)
}

// The waiver marker shares specflow's `<!-- specflow:… -->` comment shape with the region markers
// (specflow:start/end) and the composition tags (specflow:full-only/spec-only), all of which are
// matched by token. This asserts specflow:size-ok is inert to all of them, on the two surfaces that
// can actually parse it. (A waiver on a real spec file can't collide with anything: `spec/**` is
// user-owned, so no specflow command ever parses it — the exposure is entirely in managed files.)
//
//	inside a region — the shipped procedure carries the literal as its example, so a false start/end
//	                  match would truncate or duplicate the region on every upgrade.
//	outside a region — a waiver placed above a managed file's region must survive upgrade untouched,
//	                  the same guarantee any other user text outside the markers gets.
func TestSizeOkMarkerDoesNotCollideWithSpecflowMarkers(t *testing.T) {
	for _, mode := range []string{"full", "spec-only"} {
		t.Run(mode, func(t *testing.T) {
			tmp := newRepo(t)
			args := []string{"init", "--agents=claude"}
			if mode == "spec-only" {
				args = append(args, "--spec-only")
			}
			if r := run(t, tmp, args...); r.code != 0 {
				t.Fatalf("init exit %d: %s", r.code, r.stderr)
			}

			proc := filepath.Join(tmp, "specflow/procedures/spec-edit.md")
			before := read(t, proc)
			if !strings.Contains(before, "specflow:size-ok") {
				t.Fatal("spec-edit.md does not carry the size-ok example — nothing to collide")
			}
			if composeTag.MatchString(before) {
				t.Error("rendered spec-edit.md carries leftover composition tags")
			}
			// Exactly one region: a size-ok line mistaken for a start/end marker would change these.
			if got := strings.Count(before, "specflow:start"); got != 1 {
				t.Errorf("spec-edit.md has %d specflow:start markers, want 1", got)
			}
			if got := strings.Count(before, "specflow:end"); got != 1 {
				t.Errorf("spec-edit.md has %d specflow:end markers, want 1", got)
			}

			// Put a waiver above a managed file's region. A token that collided with specflow:start
			// would make extractRegion open the region here, so upgrade would eat the line.
			ag := filepath.Join(tmp, "AGENTS.md")
			waiver := "<!-- specflow:size-ok - user approved this file over 600 lines on " +
				"2026-01-31 14:05 UTC; next check at 800. -->\n"
			if err := os.WriteFile(ag, []byte(waiver+read(t, ag)), 0o644); err != nil {
				t.Fatal(err)
			}

			up := run(t, tmp, "upgrade")
			if up.code != 0 {
				t.Fatalf("upgrade exit %d: %s", up.code, up.stderr)
			}
			// A colliding token would open the region at the waiver, which changes the region hash
			// and sends AGENTS.md down the drift path — text preserved, but silently un-upgradable
			// from here on. That degradation is the failure this asserts against, not just text loss.
			if strings.Contains(up.stdout, "region(s) edited since install") {
				t.Errorf("upgrade flagged drift after a size-ok waiver:\n%s", up.stdout)
			}
			if exists(ag + ".specflow-new") {
				t.Error("upgrade wrote a drift sidecar for AGENTS.md after a size-ok waiver")
			}
			if got := read(t, proc); got != before {
				t.Error("upgrade rewrote spec-edit.md — the size-ok example perturbed region parsing")
			}
			// The region must still open *after* the waiver, not at it. (A count of start markers
			// would be the wrong assertion here: AGENTS.md names the token in its own prose.)
			got := read(t, ag)
			if !strings.HasPrefix(got, waiver+"<!-- specflow:start") {
				t.Errorf("upgrade did not preserve the size-ok waiver above the region, got:\n%.200s", got)
			}
			if !strings.Contains(got, "## File ownership") {
				t.Error("upgrade lost the AGENTS.md region content")
			}
			if r := run(t, tmp, "verify"); r.code != 0 {
				t.Fatalf("verify exit %d after a size-ok waiver: %s", r.code, r.stdout+r.stderr)
			}
		})
	}
}

// TestPruneLedgersProcedureShipped: the retention rule is the whole point of the procedure, so the
// number and the "count, not a budget" framing are asserted literally. A silent drift here (5 to some
// other number, or a rewrite into a byte budget) would leave two agents pruning the same ledger to
// different results, which is exactly what the count exists to prevent.
func TestPruneLedgersProcedureShipped(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}

	proc := read(t, filepath.Join(tmp, "specflow/procedures/prune-ledgers.md"))
	for _, want := range []string{
		"keep the 5 most recent completed entries",
		"specflow/history/CLAIMS_DONE.md",
		"count, not a line or byte budget",
		"catch-up pass",
		"Never touch `## In progress`",
		"meta: prune ledgers",
	} {
		if !strings.Contains(proc, want) {
			t.Errorf("prune-ledgers.md missing %q", want)
		}
	}
	// Pruning must stay mechanical. A stop-and-ask here would put a prompt in front of every
	// batch finish, and the archive is lossless, so there is nothing for the user to decide.
	if !strings.Contains(proc, "never gated behind a stop-and-ask") {
		t.Error("prune-ledgers.md no longer states that pruning is not gated behind a stop-and-ask")
	}

	// finish-batch must delegate, not restate: one copy of the rules, reachable by every agent.
	fin := read(t, filepath.Join(tmp, "specflow/procedures/finish-batch.md"))
	if !strings.Contains(fin, "specflow/procedures/prune-ledgers.md") {
		t.Error("finish-batch.md does not delegate to prune-ledgers.md")
	}

	// The Claude skill is a thin trigger. If the rules migrate into it, pruning silently becomes
	// Claude-only and every other agent stops pruning.
	skill := read(t, filepath.Join(tmp, ".claude/skills/prune-ledgers/SKILL.md"))
	if !strings.Contains(skill, "specflow/procedures/prune-ledgers.md") {
		t.Error("prune-ledgers SKILL.md does not point at the procedure")
	}
	if len(strings.Split(skill, "\n")) > 30 {
		t.Errorf("prune-ledgers SKILL.md is %d lines; it is a trigger, not a copy of the procedure",
			len(strings.Split(skill, "\n")))
	}
}

// TestPruneLedgersOmittedFromSpecOnly: prune-ledgers operates on BUILD_QUEUE.md and CLAIMS.md, which
// a spec-only install does not have, so shipping it would point the agent at machinery that isn't
// there: the exact class of defect Batch SL fixed.
func TestPruneLedgersOmittedFromSpecOnly(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude", "--spec-only"); r.code != 0 {
		t.Fatalf("init --spec-only exit %d: %s", r.code, r.stderr)
	}
	for _, p := range []string{
		"specflow/procedures/prune-ledgers.md",
		".claude/skills/prune-ledgers/SKILL.md",
	} {
		if exists(filepath.Join(tmp, p)) {
			t.Errorf("spec-only install shipped %s", p)
		}
	}
}

// ---------------------------------------------------------------------------
// Queue verbs: next / claim / finish
// ---------------------------------------------------------------------------

// TestParseQueueDeclaredShape drives the parser over the declared batch shape and the ways a
// hand-edited queue breaks it. The rule under test: a batch missing a declared field comes back
// with Problem set, never silently claimable.
func TestParseQueueDeclaredShape(t *testing.T) {
	cases := []struct {
		name    string
		queue   string
		wantID  string
		wantTag string
		deps    []string
		files   []string
		problem string
	}{
		{
			name: "plain claimable batch",
			queue: "## Batch 7 — Do the thing\n\n**Depends on:** none.\n\n" +
				"### Files this batch creates/edits\n- `src/a.go` — the a.\n",
			wantID: "7", files: []string{"src/a.go"},
		},
		{
			name: "backticked tag and multi-dependency",
			queue: "## Batch W `[NOT READY]` — Workflow model\n\n**Depends on:** Batch A, Batch B (both edit the same file).\n\n" +
				"### Files this batch creates/edits\n- `x.go`\n",
			wantID: "W", wantTag: "NOT READY", deps: []string{"A", "B"}, files: []string{"x.go"},
		},
		{
			name: "bare tag, brace-expanded file list",
			queue: "## Batch NB [MANUAL] — Provision\n\n" +
				"### Files this batch creates/edits\n- `proc/{one,two}.md`\n",
			wantID: "NB", wantTag: "MANUAL", files: []string{"proc/one.md", "proc/two.md"},
		},
		{
			// Trailing punctuation is prose; a leading dot is path. Trimming both ends made every
			// dotfile tree — `.claude/`, `.github/`, `.cursor/`, where specflow's own adapters live —
			// compare as a different file, silently defeating the overlap check.
			name: "dotfile paths keep their leading dot",
			queue: "## Batch AF — Adapters\n\n" +
				"### Files this batch creates/edits\n- `.claude/skills/{a,b}/SKILL.md` · `.github/x.yml`.\n",
			wantID: "AF",
			files:  []string{".claude/skills/a/SKILL.md", ".claude/skills/b/SKILL.md", ".github/x.yml"},
		},
		{
			name:    "missing file list is unparseable",
			queue:   "## Batch 9 — No files declared\n\nSome prose, no declared list.\n",
			wantID:  "9",
			problem: "no `### Files",
		},
		{
			name:    "empty file list is unparseable",
			queue:   "## Batch 9 — Empty list\n\n### Files this batch creates/edits\n\n### Verification\n- run it\n",
			wantID:  "9",
			problem: "lists no files",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := kit.ParseQueue(c.queue)
			if len(got) != 1 {
				t.Fatalf("parsed %d batches, want 1", len(got))
			}
			b := got[0]
			if b.ID != c.wantID || b.Tag != c.wantTag {
				t.Errorf("id/tag = %q/%q, want %q/%q", b.ID, b.Tag, c.wantID, c.wantTag)
			}
			if strings.Join(b.DependsOn, ",") != strings.Join(c.deps, ",") {
				t.Errorf("deps = %v, want %v", b.DependsOn, c.deps)
			}
			if c.problem == "" && strings.Join(b.Files, ",") != strings.Join(c.files, ",") {
				t.Errorf("files = %v, want %v", b.Files, c.files)
			}
			if c.problem == "" && b.Problem != "" {
				t.Errorf("unexpected problem: %s", b.Problem)
			}
			if c.problem != "" && !strings.Contains(b.Problem, c.problem) {
				t.Errorf("problem = %q, want it to mention %q", b.Problem, c.problem)
			}
		})
	}
}

// TestParseQueueDuplicateIDs: two sections answering to one id can't be claimed unambiguously, so
// both are flagged rather than one silently winning.
func TestParseQueueDuplicateIDs(t *testing.T) {
	q := "## Batch 3 — First\n\n### Files this batch creates/edits\n- `a.go`\n\n" +
		"## Batch 3 — Second\n\n### Files this batch creates/edits\n- `b.go`\n"
	got := kit.ParseQueue(q)
	if len(got) != 2 {
		t.Fatalf("parsed %d batches, want 2", len(got))
	}
	for _, b := range got {
		if !strings.Contains(b.Problem, "duplicate") {
			t.Errorf("Batch %s problem = %q, want a duplicate-id report", b.ID, b.Problem)
		}
	}
}

// seedQueue replaces the installed queue's batches with the given sections, keeping the header.
func seedQueue(t *testing.T, dir, sections string) {
	t.Helper()
	p := filepath.Join(dir, "BUILD_QUEUE.md")
	body := read(t, p)
	// Cut at a real heading: the header prose quotes "## Batch <id>" inline as an example.
	if loc := regexp.MustCompile(`(?m)^## Batch`).FindStringIndex(body); loc != nil {
		body = body[:loc[0]]
	}
	mustWrite(t, p, body+sections)
}

const twoBatchQueue = "## Batch A — First thing\n\n" +
	"### Files this batch creates/edits\n- `src/a.go`\n\n---\n\n" +
	"## Batch B — Second thing\n\n**Depends on:** Batch A.\n\n" +
	"### Files this batch creates/edits\n- `src/a.go`\n- `src/b.go`\n"

// TestNextReportsEligibilityAndBlockReasons covers the whole Eligibility section of claim-batch.md
// in one call: dependency order first, then the overlap rule once a batch is in progress.
func TestNextReportsEligibilityAndBlockReasons(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)

	r := run(t, tmp, "next")
	if r.code != 0 {
		t.Fatalf("next exit %d: %s%s", r.code, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "Batch A") || !strings.Contains(r.stdout, "First thing") {
		t.Error("next did not offer the claimable Batch A")
	}
	if !strings.Contains(r.stdout, "depends on Batch A") {
		t.Errorf("next did not block Batch B on its dependency:\n%s", r.stdout)
	}

	// Batch C has no dependency, so once A is in progress the overlap rule is what blocks it.
	seedQueue(t, tmp, twoBatchQueue+"\n---\n\n## Batch C — Independent but overlapping\n\n"+
		"### Files this batch creates/edits\n- `src/a.go`\n")
	run(t, tmp, "claim", "A")
	r2 := run(t, tmp, "next")
	if !strings.Contains(r2.stdout, "already in progress (claude)") {
		t.Errorf("claimed batch not reported as in progress:\n%s", r2.stdout)
	}
	if !strings.Contains(r2.stdout, "overlap") || !strings.Contains(r2.stdout, "src/a.go") {
		t.Errorf("next did not report the file overlap with the in-progress batch:\n%s", r2.stdout)
	}
}

// TestNextJSONShape: agents read the machine form, so it must carry the fields they branch on.
func TestNextJSONShape(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)

	r := run(t, tmp, "next", "--json")
	var rep struct {
		Claimable []struct {
			ID    string   `json:"id"`
			Title string   `json:"title"`
			Files []string `json:"files"`
		} `json:"claimable"`
		Blocked []struct {
			ID     string `json:"id"`
			Reason string `json:"reason"`
		} `json:"blocked"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &rep); err != nil {
		t.Fatalf("next --json is not valid JSON: %v\n%s", err, r.stdout)
	}
	if len(rep.Claimable) != 1 || rep.Claimable[0].ID != "A" || len(rep.Claimable[0].Files) != 1 {
		t.Errorf("claimable = %+v, want just Batch A with its declared file", rep.Claimable)
	}
	if len(rep.Blocked) != 1 || rep.Blocked[0].ID != "B" || rep.Blocked[0].Reason == "" {
		t.Errorf("blocked = %+v, want Batch B with a reason", rep.Blocked)
	}
}

// TestClaimWritesEntryAndRefusesIneligible: the verb writes the documented entry shape, and holds
// the same eligibility line `next` does, so using the CLI can't smuggle in an illegal claim.
func TestClaimWritesEntryAndRefusesIneligible(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)

	if r := run(t, tmp, "claim", "A"); r.code != 0 {
		t.Fatalf("claim A exit %d: %s%s", r.code, r.stdout, r.stderr)
	}
	claims := read(t, filepath.Join(tmp, "CLAIMS.md"))
	inProgress := claims[strings.Index(claims, "## In progress"):strings.Index(claims, "## Completed")]
	if !strings.Contains(inProgress, "### Batch A — First thing") {
		t.Errorf("claim did not write the entry heading into In progress:\n%s", inProgress)
	}
	if !strings.Contains(inProgress, "- Owner: claude") {
		t.Error("claim did not record the Owner from config.agents")
	}
	if !regexp.MustCompile(`- Started: \d{4}-\d{2}-\d{2} \d{2}:\d{2}`).MatchString(inProgress) {
		t.Errorf("claim did not write a UTC Started stamp:\n%s", inProgress)
	}

	// Re-claiming, and claiming a dependency-blocked batch, both fail loudly.
	if r := run(t, tmp, "claim", "A"); r.code == 0 {
		t.Error("claiming an already-claimed batch succeeded")
	}
	if r := run(t, tmp, "claim", "B"); r.code == 0 {
		t.Error("claiming a dependency-blocked batch succeeded")
	}
	if r := run(t, tmp, "claim", "ZZ"); r.code == 0 {
		t.Error("claiming a batch that isn't in the queue succeeded")
	}
}

// TestFinishRoundTripAndPrune walks next → claim → finish on a temp repo and asserts each file
// matches what the procedures describe by hand, including the prune boundary.
func TestFinishRoundTripAndPrune(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)

	// Fill Completed to the retention bound so finishing one more pushes the oldest out.
	claimsPath := filepath.Join(tmp, "CLAIMS.md")
	body := read(t, claimsPath)
	var filler strings.Builder
	for i := 1; i <= kit.CompletedRetention; i++ {
		fmt.Fprintf(&filler, "\n### Batch F%d — Filler %d\n- Owner: claude\n- Started: 2026-01-0%d 09:00\n- Finished: 2026-01-0%d 10:00\n- Commit: aaa000%d\n", i, i, i, i, i)
	}
	mustWrite(t, claimsPath, body+filler.String())

	// A pre-existing archive entry: both archives are newest-first, so the pruned entry must land
	// above it rather than at the end of the file.
	archivePath := filepath.Join(tmp, "specflow/history/CLAIMS_DONE.md")
	mustWrite(t, archivePath, read(t, archivePath)+"\n### Batch OLD — Archived earlier\n- Owner: claude\n- Commit: 000aaa1\n")

	run(t, tmp, "claim", "A")
	sum := filepath.Join(tmp, "sum.md")
	done := filepath.Join(tmp, "done.md")
	mustWrite(t, sum, "**What shipped**\n- The a, shipped.\n")
	mustWrite(t, done, "Shipped the a in `src/a.go`. Key commit `abc1234`.\n")

	r := run(t, tmp, "finish", "A", "--commit", "abc1234", "--summary-file", sum, "--done-file", done)
	if r.code != 0 {
		t.Fatalf("finish exit %d: %s%s", r.code, r.stdout, r.stderr)
	}

	claims := read(t, claimsPath)
	inProgress := claims[strings.Index(claims, "## In progress"):strings.Index(claims, "## Completed")]
	if strings.Contains(inProgress, "Batch A") {
		t.Error("finished batch is still in In progress")
	}
	completed := claims[strings.Index(claims, "## Completed"):]
	for _, want := range []string{"### Batch A — First thing", "- Commit: abc1234", "**What shipped**", "- The a, shipped."} {
		if !strings.Contains(completed, want) {
			t.Errorf("Completed entry missing %q:\n%s", want, completed)
		}
	}
	if !regexp.MustCompile(`- Finished: \d{4}-\d{2}-\d{2} \d{2}:\d{2}`).MatchString(completed) {
		t.Error("finish did not write a Finished stamp")
	}
	// Newest first: the batch just finished heads the section.
	if i, j := strings.Index(completed, "Batch A"), strings.Index(completed, "Batch F1"); i < 0 || j < 0 || i > j {
		t.Error("finished entry was not placed at the top of Completed")
	}
	if n := strings.Count(completed, "### Batch "); n != kit.CompletedRetention {
		t.Errorf("Completed holds %d entries, want the retention bound of %d", n, kit.CompletedRetention)
	}
	// The oldest filler moved verbatim to the archive.
	archive := read(t, archivePath)
	if i, j := strings.Index(archive, "Batch F"+fmt.Sprint(kit.CompletedRetention)), strings.Index(archive, "Batch OLD"); i < 0 || j < 0 || i > j {
		t.Errorf("archived entry was not placed above the older one (newest-first):\n%s", archive)
	}
	if !strings.Contains(archive, "### Batch F"+fmt.Sprint(kit.CompletedRetention)) || !strings.Contains(archive, fmt.Sprintf("- Commit: aaa000%d", kit.CompletedRetention)) {
		t.Errorf("oldest entry was not archived verbatim:\n%s", archive)
	}
	if strings.Contains(completed, "Batch F"+fmt.Sprint(kit.CompletedRetention)) {
		t.Error("archived entry is still in CLAIMS.md")
	}

	// The batch left the queue and its paragraph reached the queue archive.
	queue := read(t, filepath.Join(tmp, "BUILD_QUEUE.md"))
	if strings.Contains(queue, "## Batch A") {
		t.Error("finished batch still listed in BUILD_QUEUE.md")
	}
	if !strings.Contains(queue, "## Batch B") {
		t.Error("finish removed a batch it was not asked about")
	}
	qDone := read(t, filepath.Join(tmp, "specflow/history/BUILD_QUEUE_DONE.md"))
	if !strings.Contains(qDone, "## Batch A — First thing") || !strings.Contains(qDone, "Key commit `abc1234`") {
		t.Errorf("queue archive missing the batch paragraph:\n%s", qDone)
	}
	// The paragraph must land as a real entry, not inside the template's worked-example comment.
	if i, j := strings.Index(qDone, "-->"), strings.Index(qDone, "## Batch A"); i < 0 || j < i {
		t.Error("queue archive paragraph landed inside the template's HTML comment")
	}

	// B is claimable now that A is done and its files are free.
	if out := run(t, tmp, "next").stdout; !strings.Contains(out, "✓") || !strings.Contains(out, "Batch B") {
		t.Errorf("Batch B not offered after its dependency completed:\n%s", out)
	}
}

// TestFinishRefusesUnreadableLedger: the parser must never be the reason a hand edit is lost, so an
// unparseable CLAIMS.md stops the command with the file untouched.
func TestFinishRefusesUnreadableLedger(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)
	run(t, tmp, "claim", "A")

	claimsPath := filepath.Join(tmp, "CLAIMS.md")
	broken := strings.Replace(read(t, claimsPath), "## Completed", "## Done (renamed by hand)", 1)
	mustWrite(t, claimsPath, broken)

	r := run(t, tmp, "finish", "A", "--commit", "abc1234")
	if r.code == 0 {
		t.Fatal("finish succeeded against an unparseable CLAIMS.md")
	}
	if !strings.Contains(r.stderr, "In progress") && !strings.Contains(r.stderr, "Completed") {
		t.Errorf("error did not name the missing section: %s", r.stderr)
	}
	if read(t, claimsPath) != broken {
		t.Error("finish rewrote a CLAIMS.md it could not parse")
	}
	if !strings.Contains(read(t, filepath.Join(tmp, "BUILD_QUEUE.md")), "## Batch A") {
		t.Error("finish removed the queue section despite failing")
	}
}

// TestFinishWithoutProse still moves the batch, and says which prose the agent owes by hand.
func TestFinishWithoutProse(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)
	run(t, tmp, "claim", "A")

	r := run(t, tmp, "finish", "A", "--commit", "abc1234")
	if r.code != 0 {
		t.Fatalf("finish exit %d: %s%s", r.code, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "--stub-file") || !strings.Contains(r.stdout, "--done-file") {
		t.Errorf("finish did not name the prose it left to the agent:\n%s", r.stdout)
	}
	if !strings.Contains(read(t, filepath.Join(tmp, "CLAIMS.md")), "- Commit: abc1234") {
		t.Error("finish did not record the commit SHA")
	}
}

// TestVerbsDoNotCommit: committing stays with the commit/push levers and the procedures.
func TestVerbsDoNotCommit(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)
	gitOut(t, tmp, "add", "-A")
	gitOut(t, tmp, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "seed")

	run(t, tmp, "claim", "A")
	run(t, tmp, "finish", "A", "--commit", "abc1234")
	if out := gitOut(t, tmp, "log", "--oneline"); strings.Count(out, "\n") != 1 {
		t.Errorf("a verb created a commit — git log:\n%s", out)
	}
	if out := gitOut(t, tmp, "status", "--porcelain"); !strings.Contains(out, "CLAIMS.md") {
		t.Errorf("verb changes were not left uncommitted in the work tree:\n%s", out)
	}
}

// TestVerbHelpDescribesTheDivisionOfLabor — the help is where an agent learns the CLI owns
// placement while the agent owns prose, and that no verb commits.
func TestVerbHelpDescribesTheDivisionOfLabor(t *testing.T) {
	tmp := newRepo(t)
	for _, c := range []struct{ cmd, want string }{
		{"next", "claim-batch"},
		{"claim", "config.agents"},
		{"finish", "does not commit"},
	} {
		r := run(t, tmp, c.cmd, "--help")
		if r.code != 0 {
			t.Fatalf("%s --help exit %d", c.cmd, r.code)
		}
		if !strings.Contains(r.stdout, "specflow "+c.cmd) || !strings.Contains(r.stdout, c.want) {
			t.Errorf("%s --help did not mention %q:\n%s", c.cmd, c.want, r.stdout)
		}
	}
}

// ---------------------------------------------------------------------------
// Batch AF — the adapters (skill stubs, handoff hook) are managed as whole files
// ---------------------------------------------------------------------------

// adapterRels are the wholly-generated files a full claude install places that carry no marker
// region. Before whole-file management they were create-once: a fix to one never reached a repo
// that already had it.
var adapterRels = []string{
	".claude/skills/claim-batch/SKILL.md",
	".claude/skills/spec-edit/SKILL.md",
	".claude/skills/finish-batch/SKILL.md",
	".claude/skills/prune-ledgers/SKILL.md",
	".claude/hooks/specflow-handoff-reminder.sh",
}

// editStampManaged rewrites the stamp's managed map through fn — the way a test forges an install
// made by an older kit.
func editStampManaged(t *testing.T, dir string, fn func(m map[string]any)) {
	t.Helper()
	p := filepath.Join(dir, "specflow/config.json")
	var stamp map[string]any
	if err := json.Unmarshal([]byte(read(t, p)), &stamp); err != nil {
		t.Fatal(err)
	}
	m, _ := stamp["managed"].(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	fn(m)
	stamp["managed"] = m
	b, _ := json.MarshalIndent(stamp, "", "  ")
	mustWrite(t, p, string(b)+"\n")
}

// dropAdapterBaselines forges a pre-v0.1.6 stamp: region baselines only, no adapter entries. This is
// the state every existing install is in, and the one the adoption path has to handle.
func dropAdapterBaselines(t *testing.T, dir string) {
	t.Helper()
	editStampManaged(t, dir, func(m map[string]any) {
		for _, rel := range adapterRels {
			delete(m, rel)
		}
	})
}

func TestInitBaselinesAdapterFiles(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	m := stampManaged(t, tmp)
	for _, rel := range adapterRels {
		if _, ok := m[rel]; !ok {
			t.Errorf("stamp records no baseline for %s — upgrade can't tell pristine from edited", rel)
		}
	}
}

// The defect this batch exists for: specflow ships a corrected stub, and an install that already
// has the file must actually receive it.
func TestUpgradeRefreshesStaleAdapter(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	stub := filepath.Join(tmp, ".claude/skills/finish-batch/SKILL.md")
	fresh := read(t, stub)
	old := "---\nname: finish-batch\ndescription: old.\n---\n\nIn short: hand-edit the ledgers.\n"
	mustWrite(t, stub, old)
	// Baseline says the old content is what specflow put there — i.e. the user never touched it.
	editStampManaged(t, tmp, func(m map[string]any) {
		m[".claude/skills/finish-batch/SKILL.md"] = sha256Hex(old)
	})

	if r := run(t, tmp, "status"); !strings.Contains(r.stdout, "stale") || !strings.Contains(r.stdout, "finish-batch") {
		t.Errorf("status did not report the stale stub:\n%s", r.stdout)
	}
	if r := run(t, tmp, "upgrade"); r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	if got := read(t, stub); got != fresh {
		t.Errorf("upgrade left the stale stub in place:\n%s", got)
	}
	if exists(stub + ".specflow-bak") {
		t.Error("a provably pristine stub was backed up — nothing was at risk")
	}
	if r := run(t, tmp, "status"); !strings.Contains(r.stdout, "stale") || !strings.Contains(r.stdout, "none") {
		t.Errorf("status still reports staleness after upgrade:\n%s", r.stdout)
	}
}

// An install made before whole-file management has no adapter baselines at all. A pristine copy is
// adopted silently; anything else is replaced but backed up first, because specflow cannot prove it
// wrote the content and must never destroy what it didn't.
func TestUpgradeAdoptsAdaptersWithNoBaseline(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	pristine := filepath.Join(tmp, ".claude/skills/claim-batch/SKILL.md")
	pristineWant := read(t, pristine)
	legacy := filepath.Join(tmp, ".claude/skills/finish-batch/SKILL.md")
	legacyWant := read(t, legacy)
	legacyOld := "---\nname: finish-batch\ndescription: shipped by an older kit.\n---\n\nIn short: hand-edit.\n"
	mustWrite(t, legacy, legacyOld)
	dropAdapterBaselines(t, tmp)

	if r := run(t, tmp, "verify"); !strings.Contains(r.stdout, "predates whole-file management") {
		t.Errorf("verify said nothing about the un-baselined adapters:\n%s", r.stdout)
	}
	if r := run(t, tmp, "upgrade"); r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	if got := read(t, legacy); got != legacyWant {
		t.Errorf("the older kit's stub survived the upgrade:\n%s", got)
	}
	if got := read(t, legacy+".specflow-bak"); got != legacyOld {
		t.Errorf("the replaced copy was not preserved:\n%s", got)
	}
	if got := read(t, pristine); got != pristineWant {
		t.Errorf("adoption rewrote an already-current stub:\n%s", got)
	}
	if exists(pristine + ".specflow-bak") {
		t.Error("an already-current stub was backed up — there was nothing to preserve")
	}
	// Adoption is one-time: every adapter now carries a baseline.
	m := stampManaged(t, tmp)
	for _, rel := range adapterRels {
		if _, ok := m[rel]; !ok {
			t.Errorf("%s still has no baseline after the adoption upgrade", rel)
		}
	}
	if r := run(t, tmp, "verify"); r.code != 0 || strings.Contains(r.stdout, "predates whole-file management") {
		t.Errorf("verify still unhappy after adoption:\n%s", r.stdout)
	}
}

// The other half of the contract: once a file has a baseline, an edit to it is the user's and is
// never overwritten — the same protection a drifted region gets.
func TestUpgradeProtectsEditedAdapter(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	hook := filepath.Join(tmp, ".claude/hooks/specflow-handoff-reminder.sh")
	mine := read(t, hook) + "\n# my own tweak\n"
	mustWrite(t, hook, mine)

	if r := run(t, tmp, "status"); !strings.Contains(r.stdout, "specflow-handoff-reminder.sh") {
		t.Errorf("status did not report the edited hook as drift:\n%s", r.stdout)
	}
	if r := run(t, tmp, "verify"); !strings.Contains(r.stdout, "specflow-handoff-reminder.sh") {
		t.Errorf("verify did not report the edited hook:\n%s", r.stdout)
	}
	if r := run(t, tmp, "upgrade"); r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	if got := read(t, hook); got != mine {
		t.Errorf("upgrade clobbered an edited hook:\n%s", got)
	}
	if !exists(hook + ".specflow-new") {
		t.Error("no .specflow-new sidecar written for the drifted hook")
	}
}

// Before this batch a deleted or truncated stub passed verify clean.
func TestVerifyCatchesDeletedAdapter(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	stub := filepath.Join(tmp, ".claude/skills/prune-ledgers/SKILL.md")
	if err := os.Remove(stub); err != nil {
		t.Fatal(err)
	}
	r := run(t, tmp, "verify")
	if !strings.Contains(r.stdout, "prune-ledgers") || !strings.Contains(r.stdout, "missing") {
		t.Errorf("verify passed clean with a deleted skill stub:\n%s", r.stdout)
	}
	if r := run(t, tmp, "upgrade"); r.code != 0 {
		t.Fatalf("upgrade exit %d: %s", r.code, r.stderr)
	}
	if !exists(stub) {
		t.Error("upgrade did not restore the deleted stub")
	}
}

// A skill is what an agent loads *before* the procedure, so the stub's summary is what gets acted
// on. Each one must name the verb that does the work, or the agent hand-edits markdown a CLI call
// would have done correctly.
func TestSkillStubsNameTheirVerb(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	for stub, verb := range map[string]string{
		"claim-batch":   "specflow claim",
		"finish-batch":  "specflow finish",
		"prune-ledgers": "specflow finish",
		"spec-edit":     "specflow next",
	} {
		got := read(t, filepath.Join(tmp, ".claude/skills/"+stub+"/SKILL.md"))
		if !strings.Contains(got, verb) {
			t.Errorf("%s stub never mentions %q:\n%s", stub, verb, got)
		}
	}
}

// The spec-only rendering of the one stub that ships in both modes must still name no queue verbs —
// they operate on machinery a spec-only install doesn't have.
func TestSpecOnlySpecEditStubNamesNoVerbs(t *testing.T) {
	tmp := newRepo(t)
	if r := run(t, tmp, "init", "--agents=claude", "--spec-only"); r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}
	got := read(t, filepath.Join(tmp, ".claude/skills/spec-edit/SKILL.md"))
	for _, tok := range append(kit.QueueTokens, "specflow next", "specflow claim", "specflow finish") {
		if strings.Contains(got, tok) {
			t.Errorf("spec-only spec-edit stub names %q:\n%s", tok, got)
		}
	}
}

// TestFinishRefusesOverLongStub: retention bounds how many entries CLAIMS.md holds; the stub cap
// bounds how big one of them gets. Over the cap nothing is written at all, so the agent can move
// the prose into the done-file and retry against an untouched repo.
func TestFinishRefusesOverLongStub(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)
	run(t, tmp, "claim", "A")

	claimsPath := filepath.Join(tmp, "CLAIMS.md")
	before := read(t, claimsPath)
	stub := filepath.Join(tmp, "stub.md")
	var b strings.Builder
	for i := 0; i <= kit.StubMaxLines; i++ {
		fmt.Fprintf(&b, "- shipped thing %d\n", i)
	}
	mustWrite(t, stub, b.String())

	r := run(t, tmp, "finish", "A", "--commit", "abc1234", "--stub-file", stub, "--done-file", stub)
	if r.code == 0 {
		t.Fatal("finish accepted a stub over the cap")
	}
	if !strings.Contains(r.stderr, "--done-file") {
		t.Errorf("the error did not name the fix: %s", r.stderr)
	}
	if read(t, claimsPath) != before {
		t.Error("finish rewrote CLAIMS.md despite refusing the stub")
	}
	if !strings.Contains(read(t, filepath.Join(tmp, "BUILD_QUEUE.md")), "## Batch A") {
		t.Error("finish removed the queue section despite refusing the stub")
	}
	if strings.Contains(read(t, filepath.Join(tmp, "specflow/history/BUILD_QUEUE_DONE.md")), "Batch A") {
		t.Error("finish filed the archive paragraph despite refusing the stub")
	}
}

// TestFinishStubCapCountsProseOnly: the cap is on content. Blank lines and the pointer at the
// archived narrative are structure the procedure itself requires, so they don't count against it.
func TestFinishStubCapCountsProseOnly(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)
	run(t, tmp, "claim", "A")

	stub := filepath.Join(tmp, "stub.md")
	var b strings.Builder
	b.WriteString("**What shipped**\n\n")
	for i := 1; i < kit.StubMaxLines; i++ {
		fmt.Fprintf(&b, "- shipped thing %d\n\n", i)
	}
	b.WriteString("- Full narrative: `specflow/history/BUILD_QUEUE_DONE.md` → Batch A\n")
	mustWrite(t, stub, b.String())

	r := run(t, tmp, "finish", "A", "--commit", "abc1234", "--stub-file", stub)
	if r.code != 0 {
		t.Fatalf("finish rejected a stub of %d prose lines: %s%s", kit.StubMaxLines, r.stdout, r.stderr)
	}
	if !strings.Contains(read(t, filepath.Join(tmp, "CLAIMS.md")), "Full narrative:") {
		t.Error("the pointer at the archived narrative did not reach the entry")
	}
}

// TestFinishAcceptsLegacySummaryFlag: --summary-file is the pre-0.1.8 name. An agent following an
// older procedure copy must still file its prose rather than silently dropping it.
func TestFinishAcceptsLegacySummaryFlag(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)
	run(t, tmp, "claim", "A")

	sum := filepath.Join(tmp, "sum.md")
	mustWrite(t, sum, "- The a, shipped.\n")
	if r := run(t, tmp, "finish", "A", "--commit", "abc1234", "--summary-file", sum); r.code != 0 {
		t.Fatalf("finish exit %d: %s%s", r.code, r.stdout, r.stderr)
	}
	if !strings.Contains(read(t, filepath.Join(tmp, "CLAIMS.md")), "- The a, shipped.") {
		t.Error("--summary-file prose did not reach the entry")
	}
}

// TestNextReportsLedgerWeight: a count of entries can stay correct while the file behind it grows
// unreadable, so next states the size every time and warns only past a bound — which the user's
// size-ok waiver can raise, the same marker the spec-file cap uses.
func TestNextReportsLedgerWeight(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude", "--check=")
	seedQueue(t, tmp, twoBatchQueue)

	if out := run(t, tmp, "next").stdout; !strings.Contains(out, "ledger weight") {
		t.Errorf("next did not report ledger weight:\n%s", out)
	}

	queuePath := filepath.Join(tmp, "BUILD_QUEUE.md")
	body := read(t, queuePath)
	at := strings.Index(body, "## Batch A")
	if at < 0 {
		t.Fatal("seeded queue has no batch heading")
	}
	bloat := strings.Repeat("> a durable fact nobody could place in spec/\n", kit.PreambleMaxLines)
	mustWrite(t, queuePath, body[:at]+bloat+"\n"+body[at:])

	out := run(t, tmp, "next").stdout
	if !strings.Contains(out, "preamble") || !strings.Contains(out, "section 3") {
		t.Errorf("next did not warn about the over-cap preamble:\n%s", out)
	}

	waiver := fmt.Sprintf("<!-- specflow:size-ok - user approved this preamble over %d lines on 2026-01-31 14:05 UTC; next check at 400. -->\n", kit.PreambleMaxLines)
	mustWrite(t, queuePath, waiver+read(t, queuePath))
	if out := run(t, tmp, "next").stdout; strings.Contains(out, "over its") {
		t.Errorf("the size-ok waiver did not raise the preamble limit:\n%s", out)
	}
}

// ---- Batch RC: the drift sidecar, adoption on reconcile, and waivers ----

// driftRegion injects a sentinel inside the managed region of a file, simulating a hand edit.
func driftRegion(t *testing.T, path, sentinel string) {
	t.Helper()
	c := read(t, path)
	edited := startMarker.ReplaceAllStringFunc(c, func(m string) string { return m + "\n" + sentinel })
	if edited == c {
		t.Fatalf("could not locate start marker in %s", path)
	}
	os.WriteFile(path, []byte(edited), 0o644)
}

// The sidecar for a marker-delimited file is the user's file with the fresh region spliced in, not
// the bare template: `mv` is the reconciliation the warning invites, and it must not cost the user
// everything they wrote outside the markers.
func TestUpgradeSidecarKeepsTextOutsideTheRegion(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")

	f, _ := os.OpenFile(ag, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("\n## Our team notes\nDeploy on Fridays only.\n")
	f.Close()
	driftRegion(t, ag, "DRIFT-SENTINEL")

	run(t, tmp, "upgrade")
	side := read(t, ag+".specflow-new")
	if !strings.Contains(side, "Deploy on Fridays only.") {
		t.Error("sidecar dropped the user's text outside the region — `mv` over it would destroy that text")
	}
	if strings.Contains(side, "DRIFT-SENTINEL") {
		t.Error("sidecar carried the drifted region forward; it should hold the fresh region")
	}
}

// Reconciling by taking the sidecar must end the drift. Before this, the baseline was carried
// forward unchanged, so a reconciled file mismatched forever: every upgrade re-drifted it and every
// verify warned, with no exit but discarding the edit.
func TestUpgradeAdoptsReconciledRegion(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")
	driftRegion(t, ag, "DRIFT-SENTINEL")
	run(t, tmp, "upgrade")

	// The user reconciles the only way the CLI suggests.
	if err := os.Rename(ag+".specflow-new", ag); err != nil {
		t.Fatalf("mv sidecar: %v", err)
	}
	r := run(t, tmp, "upgrade")
	if regexp.MustCompile(`(?i)edited since install`).MatchString(r.stdout) {
		t.Errorf("reconciled file still reported as drift: %s", r.stdout)
	}
	if exists(ag + ".specflow-new") {
		t.Error("a second sidecar was written for an already-reconciled file")
	}
	if v := run(t, tmp, "verify"); regexp.MustCompile(`(?i)drift`).MatchString(v.stdout) {
		t.Errorf("verify still warns after reconcile: %s", v.stdout)
	}
	if s := run(t, tmp, "status"); !regexp.MustCompile(`drift\s+none`).MatchString(s.stdout) {
		t.Errorf("status still reports drift after reconcile: %s", s.stdout)
	}
}

// Waiving keeps the edit, stops the sidecar, and stops the warning — without re-recording the
// baseline, which would hand the file back to the next refresh.
func TestWaiveKeepsEditAndSilencesDrift(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")
	driftRegion(t, ag, "DRIFT-SENTINEL")
	run(t, tmp, "upgrade")
	os.Remove(ag + ".specflow-new")

	w := run(t, tmp, "waive", "AGENTS.md")
	if w.code != 0 {
		t.Fatalf("waive exit %d: %s", w.code, w.stderr)
	}
	if !strings.Contains(read(t, ag), "DRIFT-SENTINEL") {
		t.Fatal("waive changed the file; it must only record in the stamp")
	}

	r := run(t, tmp, "upgrade")
	if regexp.MustCompile(`(?i)edited since install`).MatchString(r.stdout) {
		t.Errorf("waived file still reported as drift: %s", r.stdout)
	}
	if exists(ag + ".specflow-new") {
		t.Error("sidecar written for a waived file")
	}
	if !strings.Contains(read(t, ag), "DRIFT-SENTINEL") {
		t.Fatal("upgrade overwrote a waived edit")
	}
	if v := run(t, tmp, "verify"); v.code != 0 || regexp.MustCompile(`(?i)drift`).MatchString(v.stdout) {
		t.Errorf("verify still warns about a waived file: %s", v.stdout)
	}

	// The waiver is pinned to the bytes waived: edit again and it is drift again.
	driftRegion(t, ag, "SECOND-EDIT")
	if r2 := run(t, tmp, "upgrade"); !regexp.MustCompile(`(?i)edited since install`).MatchString(r2.stdout) {
		t.Errorf("a second edit stayed silent under the old waiver: %s", r2.stdout)
	}
}

// --clear puts the file back under the normal drift contract.
func TestWaiveClearRestoresDriftReporting(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	ag := filepath.Join(tmp, "AGENTS.md")
	driftRegion(t, ag, "DRIFT-SENTINEL")
	run(t, tmp, "waive", "AGENTS.md")
	run(t, tmp, "waive", "--clear", "AGENTS.md")
	if r := run(t, tmp, "upgrade"); !regexp.MustCompile(`(?i)edited since install`).MatchString(r.stdout) {
		t.Errorf("cleared waiver did not restore drift reporting: %s", r.stdout)
	}
}

// Waiving a clean file would silently opt it out of every future refresh for no reason.
func TestWaiveRefusesCleanAndUnmanagedFiles(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	mustWrite(t, filepath.Join(tmp, "notes.md"), "mine\n")

	r := run(t, tmp, "waive", "AGENTS.md", "notes.md")
	if !strings.Contains(r.stdout, "not drifted") {
		t.Errorf("waive accepted a clean file: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "not a specflow-managed file") {
		t.Errorf("waive accepted a file it doesn't manage: %s", r.stdout)
	}
	var stamp map[string]any
	json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/config.json"))), &stamp)
	if _, ok := stamp["waived"]; ok {
		t.Error("waive wrote a waiver for files it reported as skipped")
	}
}

// The adapters are whole-file managed, and a waiver has to work the same way there.
func TestWaiveAllCoversAdapters(t *testing.T) {
	tmp := newRepo(t)
	run(t, tmp, "init", "--agents=claude")
	stub := filepath.Join(tmp, ".claude/skills/claim-batch/SKILL.md")
	mustWrite(t, stub, read(t, stub)+"\nMy own trigger note.\n")

	if r := run(t, tmp, "waive", "--all"); !strings.Contains(r.stdout, ".claude/skills/claim-batch/SKILL.md") {
		t.Fatalf("--all did not waive the drifted adapter: %s", r.stdout)
	}
	r := run(t, tmp, "upgrade")
	if regexp.MustCompile(`(?i)edited since install`).MatchString(r.stdout) {
		t.Errorf("waived adapter still reported as drift: %s", r.stdout)
	}
	if !strings.Contains(read(t, stub), "My own trigger note.") {
		t.Error("upgrade overwrote a waived adapter")
	}
}
