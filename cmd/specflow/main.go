// Command specflow drops a spec-driven, batch, claim-before-work protocol into any repo.
// A single statically-compiled binary: no runtime required on the user's machine. The templates
// tree is embedded by the root package; this CLI scaffolds and non-destructively refreshes it.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	specflow "github.com/MatanKoby/specflow"
	"github.com/MatanKoby/specflow/internal/kit"
)

// version is overridden at build time via -ldflags "-X main.version=x.y.z".
var version = "0.1.0"

// useColor gates ANSI output — set once in main(): on only for an interactive terminal with
// NO_COLOR unset, so piped/redirected output (and CI / agent capture) stays clean plain text.
var useColor bool

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func bold(s string) string   { return paint("1", s) }
func green(s string) string  { return paint("32", s) }
func yellow(s string) string { return paint("33", s) }
func dim(s string) string    { return paint("2", s) }
func cyan(s string) string   { return paint("36", s) }

type agentChoice struct{ key, label, detail string }

var agentChoices = []agentChoice{
	{"claude", "Claude Code", "CLAUDE.md + auto-triggering skills"},
	{"cursor", "Cursor", ".cursor/rules/specflow.mdc"},
	{"copilot", "GitHub Copilot", ".github/copilot-instructions.md"},
	{"bob", "IBM Bob", ".bob/rules/ (also reads AGENTS.md)"},
	{"antigravity", "Google Antigravity", ".agents/rules/ (also reads AGENTS.md)"},
}

func allAgentKeys() []string {
	out := make([]string, len(agentChoices))
	for i, c := range agentChoices {
		out[i] = c.key
	}
	return out
}

func knownAgent(k string) bool {
	for _, c := range agentChoices {
		if c.key == k {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func hasFlag(args []string, f string) bool {
	for _, a := range args {
		if a == f {
			return true
		}
	}
	return false
}

func readLine() string {
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return line
}

func pickAgents(preset string) []string {
	if preset != "" {
		var valid, bad []string
		for _, k := range splitCSV(preset) {
			if knownAgent(k) {
				valid = append(valid, k)
			} else {
				bad = append(bad, k)
			}
		}
		if len(bad) > 0 {
			fmt.Println(yellow("  ignoring unknown agent(s): " + strings.Join(bad, ", ")))
		}
		return valid
	}
	fmt.Println(bold("\nWhich agents will work in this repo?") + dim("  (AGENTS.md is always written as the universal base)"))
	for i, c := range agentChoices {
		fmt.Printf("  %s. %s  %s\n", cyan(fmt.Sprint(i+1)), c.label, dim(c.detail))
	}
	fmt.Print("\nEnter numbers (comma-separated), " + cyan("a") + " for all, or Enter for " + cyan("Claude Code") + ": ")
	answer := strings.TrimSpace(readLine())
	if answer == "" {
		return []string{"claude"}
	}
	if strings.ToLower(answer) == "a" {
		return allAgentKeys()
	}
	var out []string
	for _, tok := range splitCSV(answer) {
		var n int
		if _, err := fmt.Sscanf(tok, "%d", &n); err == nil && n >= 1 && n <= len(agentChoices) {
			out = append(out, agentChoices[n-1].key)
		}
	}
	return out
}

func cmdInit(args []string) error {
	target, err := os.Getwd()
	if err != nil {
		return err
	}
	preset := ""
	for _, a := range args {
		if strings.HasPrefix(a, "--agents=") {
			preset = strings.TrimPrefix(a, "--agents=")
		}
	}
	if preset == "" && hasFlag(args, "--all") {
		preset = strings.Join(allAgentKeys(), ",")
	}

	if kit.IsInstalled(target) {
		fmt.Println(yellow("\nThis repo already has specflow installed.") + " Run " + cyan("specflow upgrade") + " to update it.\n")
		return nil
	}
	if !kit.IsGitRepo(target) {
		fmt.Println(yellow("\nspecflow only works in git repositories.") + " Run " + cyan("git init") + " first — nothing was written.")
		return nil
	}

	agentKeys := pickAgents(preset)
	if len(agentKeys) == 0 {
		fmt.Println(yellow("\nNo agents selected. Writing the universal AGENTS.md base only.\n"))
	}
	fmt.Println(bold("\nspecflow "+version) + " → " + dim(target))

	skipped, err := kit.Scaffold(target, specflow.Templates(), version, agentKeys)
	if err != nil {
		return err
	}

	fmt.Println(green("\n✓ specflow installed."))
	fmt.Println("  Base protocol:   AGENTS.md, BUILD_QUEUE.md, CLAIMS.md, spec/, specflow/")
	if len(agentKeys) > 0 {
		fmt.Println("  Agent adapters:  " + strings.Join(agentKeys, ", "))
	}
	if len(skipped) > 0 {
		fmt.Println(yellow(fmt.Sprintf("\n  Left %d existing file(s) untouched (review/merge manually):", len(skipped))))
		for _, f := range skipped {
			fmt.Println(dim("    · " + f))
		}
	}
	fmt.Println(bold("\nNext steps:"))
	fmt.Println("  1. Fill in " + cyan("spec/README.md") + " with what this project is.")
	fmt.Println("  2. Replace the example batch in " + cyan("BUILD_QUEUE.md") + " with real work.")
	fmt.Println("  3. Point your agent at " + cyan("AGENTS.md") + " and let it claim a batch.\n")
	return nil
}

func cmdUpgrade() error {
	target, err := os.Getwd()
	if err != nil {
		return err
	}
	res, err := kit.Upgrade(target, specflow.Templates(), version)
	if err != nil {
		return err
	}
	if res.NotInstalled {
		fmt.Println(yellow("\nNo specflow install found here.") + " Run " + cyan("specflow init") + " first.\n")
		return nil
	}
	fmt.Println(green(fmt.Sprintf("\n✓ Upgraded specflow %s → %s", res.From, res.To)))
	if len(res.Refreshed) > 0 {
		fmt.Println(dim("  refreshed: " + strings.Join(res.Refreshed, ", ")))
	}
	if len(res.Added) > 0 {
		fmt.Println(dim("  added:     " + strings.Join(res.Added, ", ")))
	}
	if len(res.Migrated) > 0 {
		fmt.Println(yellow("  migrated to managed-region format (previous saved as *.specflow-bak): " + strings.Join(res.Migrated, ", ")))
	}
	if len(res.Drifted) > 0 {
		fmt.Println(yellow(fmt.Sprintf("\n  ⚠ %d managed region(s) edited since install — left untouched:", len(res.Drifted))))
		for _, f := range res.Drifted {
			fmt.Println(dim("    · " + f + "  → new version written to " + f + ".specflow-new (reconcile, then re-run upgrade)"))
		}
	}
	if len(res.Refreshed)+len(res.Added)+len(res.Migrated)+len(res.Drifted) == 0 {
		fmt.Println(dim("  Already current — nothing to refresh."))
	}
	fmt.Println(dim("\n  Your queue, claims, and spec were left untouched."))
	if res.SchemaChanged {
		fmt.Println(yellow("  Note: schema changed — review state-file format."))
	}
	fmt.Println("")
	return nil
}

func usage() {
	fmt.Printf(`
%s %s — spec-driven batch/claim protocol for AI coding agents

%s
  specflow init [--agents=claude,cursor] [--all]   %s
  specflow upgrade                                 %s
  specflow --version                               %s
  specflow --help

%s %s
`,
		bold("specflow"), dim(version),
		bold("Usage:"),
		dim("scaffold into the current repo"),
		dim("refresh the managed protocol files"),
		dim("print the installed version"),
		bold("Agents:"), strings.Join(allAgentKeys(), ", "))
}

func dispatch(command string, args []string) error {
	switch command {
	case "init":
		return cmdInit(args)
	case "upgrade":
		return cmdUpgrade()
	case "--version", "-v", "version":
		fmt.Println(version)
	case "", "--help", "-h", "help":
		usage()
	default:
		fmt.Println(yellow("Unknown command: " + command))
		usage()
		os.Exit(1)
	}
	return nil
}

func main() {
	useColor = colorEnabled()
	args := os.Args[1:]
	command := ""
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	if err := dispatch(command, args); err != nil {
		fmt.Fprintln(os.Stderr, yellow("\nspecflow error: ")+err.Error())
		os.Exit(1)
	}
}
