// Package review turns a commit's notes into review comments and posts them to
// a pull/merge request via the gh / glab CLIs.
package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"gitnotes/internal/gitcmd"
	"gitnotes/internal/note"
)

// Comment is one review comment derived from a note entry.
type Comment struct {
	Type      string `json:"type"` // "line" or "general"
	Path      string `json:"path,omitempty"`
	StartLine int    `json:"start_line,omitempty"` // new-side start; 0 for a single line
	Line      int    `json:"line,omitempty"`       // new-side anchor line (end of a range)
	Side      string `json:"side,omitempty"`
	InDiff    bool   `json:"in_diff"`
	Code      string `json:"code,omitempty"`
	Body      string `json:"body"`

	// Old-side line positions of the start/end lines, needed to build GitLab
	// line_range line codes (sha1(path)_<old>_<new>). Zero is a valid value
	// (e.g. an added line in a new file), so these are always emitted.
	StartOldLine int `json:"start_old_line"`
	EndOldLine   int `json:"end_old_line"`
}

// Payload is the full export for a commit's review.
type Payload struct {
	Commit   string    `json:"commit"`
	Short    string    `json:"short"`
	Subject  string    `json:"subject"`
	BaseRef  string    `json:"base_ref"`
	BaseSHA  string    `json:"base_sha"`
	Comments []Comment `json:"comments"`
}

// Builder assembles a Payload from HEAD's notes.
type Builder struct {
	git gitcmd.Runner
	mgr *note.Manager
}

// NewBuilder returns a Builder.
func NewBuilder(g gitcmd.Runner, m *note.Manager) *Builder {
	return &Builder{git: g, mgr: m}
}

// hunkHeader matches a unified-diff hunk header `@@ -oldStart,oldCount
// +newStart,newCount @@`, capturing all four numbers (counts may be absent).
var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// Build computes the review payload for HEAD against base (default "HEAD^").
// base is the PR/MR target ref used to decide each comment's InDiff.
func (b *Builder) Build(ctx context.Context, base string) (Payload, error) {
	if base == "" {
		base = "HEAD^"
	}
	if !b.git.CommitExists(ctx, base) {
		return Payload{}, fmt.Errorf("cannot resolve base %q to a commit", base)
	}

	sha, err := b.git.FullHash(ctx, "HEAD")
	if err != nil {
		return Payload{}, err
	}
	short, err := b.git.ShortHash(ctx, "HEAD")
	if err != nil {
		return Payload{}, err
	}
	subject, err := b.git.Subject(ctx, sha)
	if err != nil {
		return Payload{}, err
	}
	baseSHA, err := b.git.MergeBase(ctx, base, "HEAD")
	if err != nil {
		return Payload{}, err
	}

	entries, err := b.mgr.Read(ctx, sha)
	if err != nil {
		return Payload{}, err
	}

	// Cache the changed-line map per file so each file is diffed once. The map
	// keys are new-side line numbers in the diff; the value is each line's
	// old-side position (for GitLab line codes).
	changedByFile := make(map[string]map[int]int)
	changed := func(file string) map[int]int {
		if set, ok := changedByFile[file]; ok {
			return set
		}
		set := b.changedLines(ctx, base, file)
		changedByFile[file] = set
		return set
	}

	comments := make([]Comment, 0, len(entries))
	for _, e := range entries {
		if e.File == "" {
			comments = append(comments, Comment{Type: "general", Body: e.Note, Code: e.Code})
			continue
		}
		end := e.EndLine
		if end == 0 {
			end = e.StartLine
		}
		start := 0
		if e.EndLine > e.StartLine { // a true range; single lines omit start_line
			start = e.StartLine
		}
		set := changed(e.File)
		comments = append(comments, Comment{
			Type:         "line",
			Path:         e.File,
			StartLine:    start,
			Line:         end,
			Side:         "RIGHT",
			InDiff:       rangeInDiff(set, e.StartLine, end),
			Code:         e.Code,
			Body:         e.Note,
			StartOldLine: set[e.StartLine],
			EndOldLine:   set[end],
		})
	}

	return Payload{
		Commit:   sha,
		Short:    short,
		Subject:  subject,
		BaseRef:  base,
		BaseSHA:  baseSHA,
		Comments: comments,
	}, nil
}

// changedLines maps each new-side line that base...HEAD changes in file to its
// old-side position. An added line's old position is `oldStart + oldCount` of
// its hunk — the old line it follows — which is what GitLab encodes in a line
// code (sha1(path)_<old>_<new>).
func (b *Builder) changedLines(ctx context.Context, base, file string) map[int]int {
	set := make(map[int]int)
	out, err := b.git.DiffUnified0(ctx, base, file)
	if err != nil {
		return set
	}
	for _, line := range splitLines(out) {
		m := hunkHeader.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		oldStart, _ := strconv.Atoi(m[1])
		oldCount := countOr1(m[2])
		newStart, _ := strconv.Atoi(m[3])
		newCount := countOr1(m[4])
		oldPos := oldStart + oldCount
		for i := range newCount {
			set[newStart+i] = oldPos
		}
	}
	return set
}

// countOr1 parses a hunk count group: absent means 1 line, "0" means 0.
func countOr1(s string) int {
	if s == "" {
		return 1
	}
	n, _ := strconv.Atoi(s)
	return n
}

// rangeInDiff reports whether every line in [start, end] is in the changed set.
func rangeInDiff(changed map[int]int, start, end int) bool {
	if len(changed) == 0 {
		return false
	}
	for l := start; l <= end; l++ {
		if _, ok := changed[l]; !ok {
			return false
		}
	}
	return true
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// Export writes the payload as indented JSON to path, overwriting it.
func Export(p Payload, path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding payload: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// Load reads a previously exported payload from path.
func Load(path string) (Payload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Payload{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var p Payload
	if err := json.Unmarshal(data, &p); err != nil {
		return Payload{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return p, nil
}
