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
		"AGENTS.md", "BUILD_QUEUE.md", "BUILD_QUEUE_DONE.md", "CLAIMS.md", "CLAIMS_DONE.md",
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
