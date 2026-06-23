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

func helpRequested(args []string) bool {
	return hasFlag(args, "-h") || hasFlag(args, "--help")
}

// stdin is a single shared reader so multiple prompts in one run (agent pick, then injection
// consent) don't lose buffered input to a fresh per-call reader's read-ahead.
var stdin = bufio.NewReader(os.Stdin)

func readLine() string {
	line, _ := stdin.ReadString('\n')
	return line
}

// confirm asks a yes/no question, defaulting to yes (Enter = yes) — init's whole purpose is to
// install, so the safe default is to proceed; the review-the-diff handoff is the real safety net.
func confirm(prompt string) bool {
	fmt.Print(prompt + " " + cyan("[Y/n]") + ": ")
	switch strings.ToLower(strings.TrimSpace(readLine())) {
	case "", "y", "yes":
		return true
	default:
		return false
	}
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

func initUsage() {
	fmt.Printf(`
%s — scaffold specflow into the current repo

%s
  specflow init [--agents=claude,cursor,...] [--all]

%s
  --agents=<list>   comma-separated agents to wire, non-interactive (%s)
  --all             wire every supported agent, non-interactive
  -h, --help        show this help

Run with no flags to pick agents interactively, then confirm before specflow
injects its region into any file that already exists. Brownfield-safe: your
content is preserved (specflow's block sits between markers) and user-owned files
are never overwritten. %s and %s — review with %s, then commit (e.g. %s).

%s %s
`,
		bold("specflow init"),
		bold("Usage:"),
		bold("Flags:"),
		strings.Join(allAgentKeys(), ", "),
		bold("Requires a git repo"), bold("never commits"),
		cyan("git diff"), cyan("meta: install specflow"),
		bold("Agents:"), strings.Join(allAgentKeys(), ", "))
}

func upgradeUsage() {
	fmt.Printf(`
%s — refresh specflow's managed files to the installed version

%s
  specflow upgrade

Refreshes only specflow's marker-delimited regions in AGENTS.md, the procedures,
and each installed agent's instruction file. %s: text outside the markers is
preserved; a region you edited (drift) is left untouched and the fresh version is
written alongside as %s. Your queue, claims, and spec are never
touched, and upgrade never commits.

  -h, --help   show this help
`,
		bold("specflow upgrade"),
		bold("Usage:"),
		bold("Non-destructive"),
		cyan("<file>.specflow-new"))
}

func cmdInit(args []string) error {
	if helpRequested(args) {
		initUsage()
		return nil
	}
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
	// Interactive only when the user gave no agent preset — then we prompt for agents and for
	// injection consent. With --agents= / --all (agents, CI) we proceed and notify, since init
	// never commits.
	interactive := preset == ""

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

	plan, err := kit.PlanInit(target, specflow.Templates(), agentKeys)
	if err != nil {
		return err
	}

	// Phase 1 — files specflow will inject its region into (they already exist).
	allowInject := true
	if len(plan.Inject) > 0 {
		fmt.Println(bold("\nPhase 1 — existing files specflow will add its region to") +
			dim("  (your content is preserved; specflow's block is inserted between markers):"))
		for _, f := range plan.Inject {
			fmt.Println(dim("    · ") + cyan(f))
		}
		if interactive {
			allowInject = confirm("\nInject specflow's marker region into the file(s) above?")
		}
	}
	if len(plan.AlreadyWired) > 0 {
		fmt.Println(dim("\n  Already wired (specflow region or an AGENTS.md pointer present) — left as-is: " + strings.Join(plan.AlreadyWired, ", ")))
	}

	// Phase 2 — specflow-owned files to create.
	if len(plan.Create) > 0 {
		fmt.Println(bold(fmt.Sprintf("\nPhase 2 — %d specflow-owned file(s) to create", len(plan.Create))) +
			dim("  (BUILD_QUEUE.md, CLAIMS.md, spec/, specflow/, agent adapters)."))
	}

	res, err := kit.ApplyInit(target, specflow.Templates(), version, agentKeys, allowInject)
	if err != nil {
		return err
	}

	// Hand off for review — init never commits.
	fmt.Println(green("\n✓ specflow installed.") + dim("  Nothing was committed."))
	if len(res.Injected) > 0 {
		fmt.Println("  Modified " + dim("(region injected, your content kept)") + ": " + strings.Join(res.Injected, ", "))
	}
	if len(res.Created) > 0 {
		fmt.Printf("  Created:  %d file(s).\n", len(res.Created))
	}
	if len(res.SkipExisting) > 0 {
		fmt.Println(dim("  Left untouched (already present): " + strings.Join(res.SkipExisting, ", ")))
	}
	// Tier-aware notices on declined injections: AGENTS.md (Tier 1) is load-bearing; per-agent
	// instruction files (Tier 3) just auto-wire one agent and degrade gracefully.
	if len(res.Declined) > 0 {
		var tier1, tier3 []string
		for _, f := range res.Declined {
			if f == "AGENTS.md" {
				tier1 = append(tier1, f)
			} else {
				tier3 = append(tier3, f)
			}
		}
		if len(tier1) > 0 {
			fmt.Println(yellow("\n  ⚠ Declined the AGENTS.md region — specflow can't work properly without it."))
			fmt.Println(dim("    Re-run ") + cyan("specflow init") + dim(" to add it, then ") + cyan("specflow verify") + dim(" to confirm."))
		}
		if len(tier3) > 0 {
			fmt.Println(yellow("  Declined injection (left untouched): " + strings.Join(tier3, ", ")))
			fmt.Println(dim("    → those agents aren't auto-wired; each works once its file points at ") + cyan("AGENTS.md") + dim(" (re-run init to add it)."))
		}
	}

	fmt.Println(bold("\nReview, then commit:"))
	fmt.Println("  1. Inspect the changes with " + cyan("git diff") + " / " + cyan("git status") + " — remove anything you don't want.")
	fmt.Println(dim("     (specflow may be limited if required pieces are removed — run ") + cyan("specflow verify") + dim(" to check.)"))
	fmt.Println("  2. Commit when satisfied — ideally its own commit, e.g. " + cyan("meta: install specflow") + ".")
	fmt.Println("  3. Fill in " + cyan("spec/README.md") + ", seed " + cyan("BUILD_QUEUE.md") + ", and point your agent at " + cyan("AGENTS.md") + ".\n")
	return nil
}

func cmdUpgrade(args []string) error {
	if helpRequested(args) {
		upgradeUsage()
		return nil
	}
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

%s
  specflow init --help · specflow upgrade --help

%s %s
`,
		bold("specflow"), dim(version),
		bold("Usage:"),
		dim("scaffold into the current repo"),
		dim("refresh the managed protocol files"),
		dim("print the installed version"),
		bold("Per-command help:"),
		bold("Agents:"), strings.Join(allAgentKeys(), ", "))
}

func dispatch(command string, args []string) error {
	switch command {
	case "init":
		return cmdInit(args)
	case "upgrade":
		return cmdUpgrade(args)
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
