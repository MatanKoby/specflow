package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

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
		"CLAUDE.md", ".claude/skills/claim-batch/SKILL.md", ".cursor/rules/specflow.mdc",
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
	// stdin: Enter (default Claude) for the agent pick, then "n" to decline injection.
	r := runStdin(t, tmp, "\nn\n", "init")
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
		".claude/skills/claim-batch/SKILL.md", ".claude/skills/finish-batch/SKILL.md",
		"specflow/history/BUILD_QUEUE_DONE.md", "specflow/history/CLAIMS_DONE.md",
	}
	specOnlyKept = []string{
		"AGENTS.md", "spec/README.md", "specflow/config.json",
		"specflow/procedures/spec-edit.md", ".claude/skills/spec-edit/SKILL.md", "CLAUDE.md",
	}
	// composeTag finds any leftover section-composition scaffolding in a rendered file.
	composeTag = regexp.MustCompile(`specflow:(full-only|spec-only)`)
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
