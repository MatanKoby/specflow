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

func TestInitWritesFilesAndStamp(t *testing.T) {
	tmp := t.TempDir()
	r := run(t, tmp, "init", "--agents=claude,cursor")
	if r.code != 0 {
		t.Fatalf("init exit %d: %s", r.code, r.stderr)
	}

	for _, f := range []string{
		"AGENTS.md", "BUILD_QUEUE.md", "BUILD_QUEUE_DONE.md", "CLAIMS.md", "CLAIMS_DONE.md",
		"spec/README.md", "specflow/.spec-batch.json",
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

	if !regexp.MustCompile(`(?i)not a git repository`).MatchString(r.stdout) {
		t.Error("no git warning on init outside a repo")
	}

	raw := read(t, filepath.Join(tmp, "specflow/.spec-batch.json"))
	if strings.Contains(raw, "{{") {
		t.Error("unfilled placeholder remains in stamp")
	}
	var stamp map[string]any
	if err := json.Unmarshal([]byte(raw), &stamp); err != nil {
		t.Fatalf("stamp not valid JSON: %v", err)
	}
	if v, _ := stamp["kitVersion"].(string); !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(v) {
		t.Errorf("kitVersion not a version: %q", v)
	}
	if v, _ := stamp["agents"].(string); v != "claude,cursor" {
		t.Errorf("agents = %q, want claude,cursor", v)
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

func TestReinitGuarded(t *testing.T) {
	tmp := t.TempDir()
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
	tmp := t.TempDir()
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
	json.Unmarshal([]byte(read(t, filepath.Join(tmp, "specflow/.spec-batch.json"))), &stamp)
	if _, ok := stamp["upgradedAt"]; !ok {
		t.Error("upgrade did not record upgradedAt")
	}
}

var startMarker = regexp.MustCompile(`(?s)<!--\s*specflow:start\b.*?-->`)

func TestUpgradePreservesTextOutsideMarkers(t *testing.T) {
	tmp := t.TempDir()
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
	tmp := t.TempDir()
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

func TestUpgradeMigratesPreMarkerFile(t *testing.T) {
	tmp := t.TempDir()
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
	tmp := t.TempDir()
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
