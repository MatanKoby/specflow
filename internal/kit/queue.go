// Queue verbs: the file surgery the procedures otherwise ask an agent to perform by hand.
//
// `next` answers the whole eligibility section of claim-batch.md, `claim` writes the In-progress
// entry, and `finish` moves that entry to Completed, drops the batch from the queue, files both
// archive paragraphs, and prunes the ledger. The CLI owns placement, format, and timestamps; every
// word of prose still comes from the agent. No verb commits, and no verb rewrites a file it could
// not fully parse: a hand edit that confuses the parser stops the command instead of being lost.
package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CompletedRetention is the number of newest entries CLAIMS.md `## Completed` keeps; older ones
// move to specflow/history/CLAIMS_DONE.md. Same bound prune-ledgers.md states in prose.
const CompletedRetention = 5

// Repo-relative paths of the four files the verbs read and write.
const (
	queueRel      = "BUILD_QUEUE.md"
	claimsRel     = "CLAIMS.md"
	queueDoneRel  = "specflow/history/BUILD_QUEUE_DONE.md"
	claimsDoneRel = "specflow/history/CLAIMS_DONE.md"
)

// stampFormat is the UTC timestamp shape every ledger entry uses.
const stampFormat = "2006-01-02 15:04"

// Line matchers for the declared batch shape. Everything not matched here is free prose the parser
// steps over: only the heading, the Depends-on line, and the file list carry machine meaning.
var (
	qBatchHeadRe = regexp.MustCompile(`^##\s+Batch\s+([A-Za-z0-9._-]+)\s*(.*)$`)
	qTagRe       = regexp.MustCompile("^`?\\[([^\\]]+)\\]`?\\s*")
	qDependsRe   = regexp.MustCompile(`(?i)^\s*\**Depends on\**:?\**\s*(.*)$`)
	qFilesHeadRe = regexp.MustCompile(`(?i)^###\s+Files this batch\b`)
	qBatchRefRe  = regexp.MustCompile(`(?i)Batch\s+([A-Za-z0-9._-]+)`)
	qTickedRe    = regexp.MustCompile("`([^`]+)`")
	qAnyH2Re     = regexp.MustCompile(`^##\s`)
	qAnyHeadRe   = regexp.MustCompile(`^#{1,6}\s`)
	qBraceRe     = regexp.MustCompile(`^([^{]*)\{([^{}]*)\}(.*)$`)
)

// Tags that exclude a batch from claiming. Any *other* tag is exclusionary too (AGENTS.md: "any tag
// you don't recognize — treat as exclusionary and ask the user"), so this list only exists to name
// the familiar ones in the reason line.
var knownTags = map[string]string{
	"MANUAL":    "the user executes this batch",
	"NOT READY": "blocked on external work or undecided design",
	"DEFERRED":  "parked by the user",
}

// Batch is one `## Batch …` section of BUILD_QUEUE.md, parsed into the fields the eligibility rules
// need. Start/End are the section's half-open line range in the file, which is what lets finish
// delete it without touching a neighbor.
type Batch struct {
	ID        string   `json:"id"`
	Tag       string   `json:"tag,omitempty"`
	Title     string   `json:"title"`
	Heading   string   `json:"-"`
	DependsOn []string `json:"dependsOn,omitempty"`
	Files     []string `json:"files,omitempty"`
	Problem   string   `json:"problem,omitempty"`
	Start     int      `json:"-"`
	End       int      `json:"-"`
}

// ParseQueue splits BUILD_QUEUE.md into batch sections. It never fails: a section that is missing a
// declared field comes back with Problem set, which callers surface rather than treating the batch
// as claimable.
func ParseQueue(content string) []Batch {
	lines := splitLines(content)
	var out []Batch
	for i := 0; i < len(lines); i++ {
		m := qBatchHeadRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if qAnyH2Re.MatchString(lines[j]) {
				end = j
				break
			}
		}
		b := Batch{ID: m[1], Heading: strings.TrimRight(lines[i], " \t"), Start: i, End: end}
		rest := strings.TrimSpace(m[2])
		if t := qTagRe.FindStringSubmatch(rest); t != nil {
			b.Tag = strings.ToUpper(strings.TrimSpace(t[1]))
			rest = strings.TrimSpace(qTagRe.ReplaceAllString(rest, ""))
		}
		b.Title = strings.TrimSpace(strings.TrimLeft(rest, "—–-: "))
		b.DependsOn, b.Files, b.Problem = parseBatchBody(lines[i+1 : end])
		out = append(out, b)
		i = end - 1
	}
	markDuplicates(out)
	return out
}

// parseBatchBody pulls the Depends-on line and the declared file list out of one section body.
func parseBatchBody(body []string) (deps, files []string, problem string) {
	sawFilesHeading := false
	for i := 0; i < len(body); i++ {
		line := body[i]
		if d := qDependsRe.FindStringSubmatch(line); d != nil && strings.Contains(strings.ToLower(line), "depends on") {
			deps = parseDepends(d[1])
			continue
		}
		if qFilesHeadRe.MatchString(line) {
			sawFilesHeading = true
			for j := i + 1; j < len(body); j++ {
				if qAnyHeadRe.MatchString(body[j]) {
					i = j - 1
					break
				}
				files = append(files, parseFileLine(body[j])...)
			}
			continue
		}
	}
	switch {
	case !sawFilesHeading:
		problem = "no `### Files this batch creates/edits` section, so the overlap check can't be answered"
	case len(files) == 0:
		problem = "the `### Files this batch creates/edits` section lists no files"
	}
	return deps, dedupe(files), problem
}

// parseDepends reads `Batch X, Batch Y` out of a Depends-on line, ignoring any parenthetical
// rationale after the list and treating "none" as no dependencies.
func parseDepends(rest string) []string {
	if i := strings.Index(rest, "("); i >= 0 {
		rest = rest[:i]
	}
	rest = strings.TrimSpace(rest)
	if rest == "" || strings.HasPrefix(strings.ToLower(rest), "none") {
		return nil
	}
	var out []string
	for _, m := range qBatchRefRe.FindAllStringSubmatch(rest, -1) {
		if id := strings.Trim(m[1], ".,;:"); id != "" {
			out = append(out, id)
		}
	}
	return dedupe(out)
}

// parseFileLine extracts the paths from one line of a declared file list. Backticked paths are the
// convention the template demonstrates; a bare list item falls back to its text up to the first
// separator, so a queue written without backticks still yields something to compare.
func parseFileLine(line string) []string {
	var out []string
	if m := qTickedRe.FindAllStringSubmatch(line, -1); len(m) > 0 {
		for _, g := range m {
			out = append(out, expandBraces(strings.TrimSpace(g[1]))...)
		}
		return out
	}
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "- ") && !strings.HasPrefix(t, "* ") {
		return nil
	}
	t = strings.TrimSpace(t[2:])
	for _, sep := range []string{" — ", " – ", " - ", ": ", " · "} {
		if i := strings.Index(t, sep); i >= 0 {
			t = t[:i]
		}
	}
	t = strings.Trim(strings.TrimSpace(t), "`.,")
	if t == "" || strings.Contains(t, " ") {
		return nil // prose, not a path
	}
	return expandBraces(t)
}

// expandBraces turns `dir/{a,b}.md` into the two paths it stands for. One level only, which is the
// idiom queues actually use; anything else comes back untouched.
func expandBraces(p string) []string {
	m := qBraceRe.FindStringSubmatch(p)
	if m == nil {
		return []string{normalizePath(p)}
	}
	var out []string
	for _, part := range strings.Split(m[2], ",") {
		out = append(out, expandBraces(m[1]+strings.TrimSpace(part)+m[3])...)
	}
	return out
}

// normalizePath makes two spellings of the same file compare equal for the overlap check.
func normalizePath(p string) string {
	p = strings.TrimSpace(strings.Trim(strings.TrimSpace(p), "`.,"))
	p = strings.TrimPrefix(p, "./")
	return strings.TrimSuffix(p, "/")
}

// markDuplicates flags every batch sharing an id: with two sections answering to one name, neither
// can be claimed unambiguously.
func markDuplicates(batches []Batch) {
	seen := map[string]int{}
	for _, b := range batches {
		seen[strings.ToLower(b.ID)]++
	}
	for i := range batches {
		if seen[strings.ToLower(batches[i].ID)] > 1 && batches[i].Problem == "" {
			batches[i].Problem = "duplicate batch id: two sections declare this id"
		}
	}
}

// ClaimEntry is one `### Batch …` entry in CLAIMS.md, with its half-open line range so the verbs can
// move it verbatim rather than re-rendering it.
type ClaimEntry struct {
	ID      string
	Heading string
	Owner   string
	Start   int
	End     int
}

// Claims is a parsed CLAIMS.md: the two sections, their entries, and the lines they came from.
type Claims struct {
	lines      []string
	progStart  int // first line after the `## In progress` heading
	progEnd    int // one past the section's last line
	compStart  int
	compEnd    int
	InProgress []ClaimEntry
	Completed  []ClaimEntry
}

// ParseClaims reads CLAIMS.md. A missing section heading is a hard error: the verbs would otherwise
// have nowhere well-defined to write, and guessing is how a hand edit gets clobbered.
func ParseClaims(content string) (Claims, error) {
	c := Claims{lines: splitLines(content)}
	prog, pOK := sectionRange(c.lines, "In progress")
	comp, cOK := sectionRange(c.lines, "Completed")
	if !pOK || !cOK {
		return c, fmt.Errorf("%s is missing a `## In progress` or `## Completed` heading, so it can't be edited safely", claimsRel)
	}
	c.progStart, c.progEnd = prog[0], prog[1]
	c.compStart, c.compEnd = comp[0], comp[1]
	c.InProgress = parseEntries(c.lines, c.progStart, c.progEnd)
	c.Completed = parseEntries(c.lines, c.compStart, c.compEnd)
	return c, nil
}

// sectionRange locates a `## <name>` section body as a half-open line range.
func sectionRange(lines []string, name string) ([2]int, bool) {
	head := regexp.MustCompile(`(?i)^##\s+` + regexp.QuoteMeta(name) + `\s*$`)
	for i, l := range lines {
		if !head.MatchString(l) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if qAnyH2Re.MatchString(lines[j]) {
				end = j
				break
			}
		}
		return [2]int{i + 1, end}, true
	}
	return [2]int{}, false
}

// parseEntries pairs each `### Batch …` heading in a section with its Owner line and line range.
func parseEntries(lines []string, start, end int) []ClaimEntry {
	var out []ClaimEntry
	for i := start; i < end; i++ {
		m := claimEntryHeadRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		stop := end
		for j := i + 1; j < end; j++ {
			if claimEntryHeadRe.MatchString(lines[j]) || qAnyHeadRe.MatchString(lines[j]) && strings.HasPrefix(lines[j], "###") {
				stop = j
				break
			}
		}
		e := ClaimEntry{ID: m[1], Heading: strings.TrimRight(lines[i], " \t"), Start: i, End: stop}
		for j := i + 1; j < stop; j++ {
			if o := ownerLineRe.FindStringSubmatch(lines[j]); o != nil {
				e.Owner = strings.TrimSpace(o[1])
				break
			}
		}
		for e.End > e.Start+1 && strings.TrimSpace(lines[e.End-1]) == "" {
			e.End--
		}
		out = append(out, e)
		i = stop - 1
	}
	return out
}

var claimEntryHeadRe = regexp.MustCompile(`^###\s+Batch\s+([A-Za-z0-9._-]+)`)

// find returns the entry for id in a section, or nil.
func find(entries []ClaimEntry, id string) *ClaimEntry {
	for i := range entries {
		if strings.EqualFold(entries[i].ID, id) {
			return &entries[i]
		}
	}
	return nil
}

// insertPoint is where a new entry goes at the "top" of a section: after the heading and after any
// leading blank lines or HTML comments the template puts there.
func insertPoint(lines []string, start, end int) int {
	i := start
	for i < end {
		t := strings.TrimSpace(lines[i])
		switch {
		case t == "":
			i++
		case strings.HasPrefix(t, "<!--"):
			for i < end && !strings.Contains(lines[i], "-->") {
				i++
			}
			i++
		default:
			return i
		}
	}
	return i
}

// NextItem is one batch as `next` reports it: claimable, or blocked with the reason why.
type NextItem struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Tag    string   `json:"tag,omitempty"`
	Files  []string `json:"files,omitempty"`
	Reason string   `json:"reason,omitempty"`
}

// NextReport is the read-only eligibility answer: the whole Eligibility section of claim-batch.md,
// computed in one call instead of six to nine reads across two files.
type NextReport struct {
	Claimable  []NextItem `json:"claimable"`
	Blocked    []NextItem `json:"blocked"`
	Problems   []string   `json:"problems,omitempty"`
	InProgress []string   `json:"inProgress,omitempty"`
}

// Next reports which batches are claimable right now. It writes nothing.
func Next(targetDir string) (NextReport, error) {
	rep := NextReport{Claimable: []NextItem{}, Blocked: []NextItem{}}
	qb, err := os.ReadFile(destPath(targetDir, queueRel))
	if err != nil {
		return rep, fmt.Errorf("no %s here (spec-only installs have no queue): %w", queueRel, err)
	}
	batches := ParseQueue(string(qb))
	cb, err := os.ReadFile(destPath(targetDir, claimsRel))
	if err != nil {
		return rep, fmt.Errorf("no %s here: %w", claimsRel, err)
	}
	claims, err := ParseClaims(string(cb))
	if err != nil {
		return rep, err
	}
	done := completedIDs(targetDir, claims)

	byID := map[string]Batch{}
	for _, b := range batches {
		byID[strings.ToLower(b.ID)] = b
	}
	// Files locked by batches currently in progress, mapped back to the batch holding them.
	locked := map[string]string{}
	for _, e := range claims.InProgress {
		rep.InProgress = append(rep.InProgress, e.ID)
		for _, f := range byID[strings.ToLower(e.ID)].Files {
			locked[f] = e.ID
		}
	}

	for _, b := range batches {
		item := NextItem{ID: b.ID, Title: b.Title, Tag: b.Tag, Files: b.Files}
		item.Reason = blockReason(b, claims, done, locked)
		if item.Reason == "" {
			rep.Claimable = append(rep.Claimable, item)
		} else {
			rep.Blocked = append(rep.Blocked, item)
		}
		if b.Problem != "" {
			rep.Problems = append(rep.Problems, "Batch "+b.ID+": "+b.Problem)
		}
	}
	return rep, nil
}

// blockReason applies the eligibility rules in the order claim-batch.md states them, returning the
// first that fires. An empty string means claimable.
func blockReason(b Batch, claims Claims, done map[string]bool, locked map[string]string) string {
	if b.Tag != "" {
		if why, ok := knownTags[b.Tag]; ok {
			return "[" + b.Tag + "] " + why
		}
		return "[" + b.Tag + "] unrecognized tag, treat as exclusionary and ask the user"
	}
	if b.Problem != "" {
		return "unparseable: " + b.Problem
	}
	if e := find(claims.InProgress, b.ID); e != nil {
		if e.Owner == "" || e.Owner == "none" {
			return "handed off and unowned in " + claimsRel + " (see claim-batch.md → Mid-batch handoff)"
		}
		return "already in progress (" + e.Owner + ")"
	}
	if find(claims.Completed, b.ID) != nil || done[strings.ToLower(b.ID)] {
		return "already completed"
	}
	for _, d := range b.DependsOn {
		if !done[strings.ToLower(d)] && find(claims.Completed, d) == nil {
			return "depends on Batch " + d + ", which is not completed"
		}
	}
	for _, f := range b.Files {
		if other, ok := locked[f]; ok {
			return "files overlap with Batch " + other + " (in progress): " + f
		}
	}
	return ""
}

// completedIDs is every batch id known to be done: the Completed section plus the archive.
func completedIDs(targetDir string, claims Claims) map[string]bool {
	out := map[string]bool{}
	for _, e := range claims.Completed {
		out[strings.ToLower(e.ID)] = true
	}
	if b, err := os.ReadFile(destPath(targetDir, claimsDoneRel)); err == nil {
		for _, l := range splitLines(string(b)) {
			if m := claimEntryHeadRe.FindStringSubmatch(l); m != nil {
				out[strings.ToLower(m[1])] = true
			}
		}
	}
	return out
}

// ClaimResult reports what `claim` wrote.
type ClaimResult struct {
	Entry string // the entry as written, so the caller can echo it
	Owner string
}

// Claim writes the In-progress entry for a batch. It refuses any batch `next` would not offer, so
// the eligibility rules hold whether the agent uses the verb or the procedure by hand.
func Claim(targetDir, id, owner string) (ClaimResult, error) {
	var res ClaimResult
	qb, err := os.ReadFile(destPath(targetDir, queueRel))
	if err != nil {
		return res, fmt.Errorf("no %s here: %w", queueRel, err)
	}
	batches := ParseQueue(string(qb))
	var target *Batch
	for i := range batches {
		if strings.EqualFold(batches[i].ID, id) {
			target = &batches[i]
			break
		}
	}
	if target == nil {
		return res, fmt.Errorf("no `## Batch %s` section in %s", id, queueRel)
	}
	cb, err := os.ReadFile(destPath(targetDir, claimsRel))
	if err != nil {
		return res, fmt.Errorf("no %s here: %w", claimsRel, err)
	}
	claims, err := ParseClaims(string(cb))
	if err != nil {
		return res, err
	}
	if why := blockReason(*target, claims, completedIDs(targetDir, claims), map[string]string{}); why != "" {
		return res, fmt.Errorf("Batch %s is not claimable: %s", target.ID, why)
	}
	entry := []string{
		"### Batch " + target.ID + " — " + target.Title,
		"- Owner: " + owner,
		"- Started: " + time.Now().UTC().Format(stampFormat),
		"",
	}
	at := insertPoint(claims.lines, claims.progStart, claims.progEnd)
	out := insertBlock(claims.lines, at, entry)
	if err := writeFile(destPath(targetDir, claimsRel), []byte(joinLines(out))); err != nil {
		return res, err
	}
	res.Entry = strings.TrimRight(strings.Join(entry, "\n"), "\n")
	res.Owner = owner
	return res, nil
}

// FinishResult reports what `finish` changed, so the CLI can print it and the agent knows what is
// left to do by hand.
type FinishResult struct {
	Batch        string
	QueueRemoved bool
	Archived     []string // batch ids moved out of CLAIMS.md by the prune
	Wrote        []string // files touched
	NoSummary    bool
	NoParagraph  bool
}

// Finish moves a claimed batch to done: the CLAIMS.md entry gains Finished/Commit and the agent's
// summary and moves to the top of Completed, the batch section leaves BUILD_QUEUE.md, the agent's
// paragraph lands in BUILD_QUEUE_DONE.md, and Completed is pruned to its newest entries. Every file
// is parsed and rewritten in memory first, so a parse failure stops the whole command.
func Finish(targetDir, id, commit, summary, paragraph string) (FinishResult, error) {
	res := FinishResult{Batch: id, NoSummary: strings.TrimSpace(summary) == "", NoParagraph: strings.TrimSpace(paragraph) == ""}
	cb, err := os.ReadFile(destPath(targetDir, claimsRel))
	if err != nil {
		return res, fmt.Errorf("no %s here: %w", claimsRel, err)
	}
	claims, err := ParseClaims(string(cb))
	if err != nil {
		return res, err
	}
	entry := find(claims.InProgress, id)
	if entry == nil {
		if find(claims.Completed, id) != nil {
			return res, fmt.Errorf("Batch %s is already in `## Completed`", id)
		}
		return res, fmt.Errorf("Batch %s is not in `## In progress` in %s (claim it first)", id, claimsRel)
	}
	res.Batch = entry.ID

	// Rebuild the entry: Finished + Commit right after Started, the agent's summary at the end.
	body := completeEntry(claims.lines[entry.Start:entry.End], commit, summary)

	// Drop it from In progress, then re-insert at the top of Completed. Removing first keeps the
	// Completed range valid only if we recompute, so the whole file is reassembled in one pass.
	remaining := removeBlock(claims.lines, entry.Start, entry.End)
	shrunk := len(claims.lines) - len(remaining)
	shift := func(i int) int {
		if i > entry.Start {
			return i - shrunk
		}
		return i
	}
	compStart, compEnd := shift(claims.compStart), shift(claims.compEnd)
	at := insertPoint(remaining, compStart, compEnd)
	merged := insertBlock(remaining, at, body)

	// Prune: keep the newest CompletedRetention entries, archive the rest verbatim.
	pruned, archived, archivedIDs, err := pruneCompleted(merged)
	if err != nil {
		return res, err
	}
	res.Archived = archivedIDs

	// Queue: delete the batch section, if it is still there.
	var queueOut string
	queuePath := destPath(targetDir, queueRel)
	if qb, err := os.ReadFile(queuePath); err == nil {
		lines := splitLines(string(qb))
		for _, b := range ParseQueue(string(qb)) {
			if strings.EqualFold(b.ID, id) {
				lines = removeBlock(lines, b.Start, b.End)
				res.QueueRemoved = true
				break
			}
		}
		queueOut = joinLines(lines)
	}

	// Writes happen only after every edit above computed cleanly.
	if err := writeFile(destPath(targetDir, claimsRel), []byte(joinLines(pruned))); err != nil {
		return res, err
	}
	res.Wrote = append(res.Wrote, claimsRel)
	if res.QueueRemoved {
		if err := writeFile(queuePath, []byte(queueOut)); err != nil {
			return res, err
		}
		res.Wrote = append(res.Wrote, queueRel)
	}
	if len(archived) > 0 {
		if err := prependArchive(destPath(targetDir, claimsDoneRel), archived); err != nil {
			return res, err
		}
		res.Wrote = append(res.Wrote, claimsDoneRel)
	}
	if !res.NoParagraph {
		block := []string{"## Batch " + res.Batch + " — " + entryTitle(entry.Heading), strings.TrimRight(paragraph, "\n"), ""}
		if err := prependArchive(destPath(targetDir, queueDoneRel), block); err != nil {
			return res, err
		}
		res.Wrote = append(res.Wrote, queueDoneRel)
	}
	return res, nil
}

// completeEntry adds the Finished and Commit fields (right after Started, the order the template
// documents) and appends the agent's summary to the entry body.
func completeEntry(entry []string, commit, summary string) []string {
	out := append([]string{}, entry...)
	fields := []string{"- Finished: " + time.Now().UTC().Format(stampFormat)}
	if strings.TrimSpace(commit) != "" {
		fields = append(fields, "- Commit: "+strings.TrimSpace(commit))
	}
	at := 1
	for i, l := range out {
		if strings.HasPrefix(strings.TrimSpace(l), "- Started:") {
			at = i + 1
			break
		}
	}
	if at > len(out) {
		at = len(out)
	}
	merged := append([]string{}, out[:at]...)
	merged = append(merged, fields...)
	merged = append(merged, out[at:]...)
	if s := strings.TrimRight(summary, "\n"); strings.TrimSpace(s) != "" {
		merged = append(merged, "")
		merged = append(merged, splitLines(s)...)
	}
	for len(merged) > 0 && strings.TrimSpace(merged[len(merged)-1]) == "" {
		merged = merged[:len(merged)-1]
	}
	return merged
}

// entryTitle recovers the title from a `### Batch N — title` heading, for the archive heading.
func entryTitle(heading string) string {
	if m := claimEntryHeadRe.FindStringSubmatch(heading); m != nil {
		rest := strings.TrimSpace(heading[len(m[0]):])
		return strings.TrimSpace(strings.TrimLeft(rest, "—–-: "))
	}
	return ""
}

// pruneCompleted trims CLAIMS.md `## Completed` to CompletedRetention entries, returning the new
// file plus the archived entries verbatim, newest first.
func pruneCompleted(lines []string) (kept []string, archived []string, ids []string, err error) {
	c, err := ParseClaims(joinLines(lines))
	if err != nil {
		return nil, nil, nil, err
	}
	if len(c.Completed) <= CompletedRetention {
		return lines, nil, nil, nil
	}
	drop := c.Completed[CompletedRetention:]
	out := append([]string{}, lines...)
	// Remove from the bottom up so the earlier ranges stay valid.
	for i := len(drop) - 1; i >= 0; i-- {
		out = removeBlock(out, drop[i].Start, drop[i].End)
	}
	for _, e := range drop {
		ids = append(ids, e.ID)
		archived = append(archived, lines[e.Start:e.End]...)
		archived = append(archived, "")
	}
	return out, archived, ids, nil
}

// prependArchive inserts a block at the top of an archive file: after its header prose, before the
// first existing `## ` entry, since both archives are newest-first.
func prependArchive(path string, block []string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		b = nil
	}
	lines := splitLines(string(b))
	at := len(lines)
	inComment := false
	for i, l := range lines {
		// The shipped templates carry a worked example inside an HTML comment, whose `## Batch …`
		// line would otherwise look like the first entry and swallow the block.
		if inComment {
			if strings.Contains(l, "-->") {
				inComment = false
			}
			continue
		}
		if strings.Contains(l, "<!--") && !strings.Contains(l, "-->") {
			inComment = true
			continue
		}
		if qAnyH2Re.MatchString(l) {
			at = i
			break
		}
	}
	return writeFile(path, []byte(joinLines(insertBlock(lines, at, block))))
}

// splitLines / joinLines round-trip a file through a line slice without gaining or losing a
// trailing newline, so an untouched region comes back byte-identical.
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// insertBlock puts block at index at, separated from whatever is on either side by exactly one
// blank line. Blank-line tidying is deliberately local to the edit: the rest of the file is
// reassembled byte for byte, so a user's spacing elsewhere survives.
func insertBlock(lines []string, at int, block []string) []string {
	for len(block) > 0 && strings.TrimSpace(block[0]) == "" {
		block = block[1:]
	}
	for len(block) > 0 && strings.TrimSpace(block[len(block)-1]) == "" {
		block = block[:len(block)-1]
	}
	if len(block) == 0 {
		return lines
	}
	out := append([]string{}, lines[:at]...)
	if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
		out = append(out, "")
	}
	out = append(out, block...)
	if at < len(lines) {
		if strings.TrimSpace(lines[at]) != "" {
			out = append(out, "")
		}
		out = append(out, lines[at:]...)
	}
	return out
}

// removeBlock deletes the half-open range [start, end) and collapses the blank run left at the seam
// to a single blank line, so repeated finishes don't hollow out the file.
func removeBlock(lines []string, start, end int) []string {
	out := append([]string{}, lines[:start]...)
	out = append(out, lines[end:]...)
	i := start
	for i > 0 && strings.TrimSpace(out[i-1]) == "" {
		i--
	}
	j := i
	for j < len(out) && strings.TrimSpace(out[j]) == "" {
		j++
	}
	if j-i > 1 {
		keep := 1
		if i == 0 || j >= len(out) {
			keep = 0
		}
		out = append(out[:i+keep], out[j:]...)
	}
	return out
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// MarshalNext renders a NextReport as the `--json` payload.
func MarshalNext(rep NextReport) (string, error) {
	b, err := json.MarshalIndent(rep, "", "  ")
	return string(b), err
}

// QueuePath is where the verbs expect the queue, for callers that report paths.
func QueuePath(targetDir string) string { return filepath.Join(targetDir, queueRel) }

// ConfiguredAgents returns the agents recorded in config.agents, which is where `claim` gets the
// Owner field. A missing or unreadable config gives an empty list rather than an error, so the
// caller can say "not installed here" in its own words.
func ConfiguredAgents(targetDir string) []string {
	b, err := os.ReadFile(configPath(targetDir))
	if err != nil {
		return nil
	}
	var stamp map[string]any
	if json.Unmarshal(b, &stamp) != nil {
		return nil
	}
	return installedAgents(stamp)
}
