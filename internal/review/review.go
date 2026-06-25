package review

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"gitnotes/internal/gitcmd"
	"gitnotes/internal/note"
)

type Comment struct {
	Type      string `json:"type"`
	Path      string `json:"path,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	Line      int    `json:"line,omitempty"`
	Side      string `json:"side,omitempty"`
	InDiff    bool   `json:"in_diff"`
	Code      string `json:"code,omitempty"`
	Body      string `json:"body"`
	Submitted bool   `json:"submitted"`

	StartOldLine int `json:"start_old_line"`
	EndOldLine   int `json:"end_old_line"`
}

type Payload struct {
	Commit   string    `json:"commit"`
	Short    string    `json:"short"`
	Subject  string    `json:"subject"`
	BaseRef  string    `json:"base_ref"`
	BaseSHA  string    `json:"base_sha"`
	Comments []Comment `json:"comments"`
}

type Builder struct {
	git gitcmd.Runner
	mgr *note.Manager
}

func NewBuilder(g gitcmd.Runner, m *note.Manager) *Builder {
	return &Builder{git: g, mgr: m}
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

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
			comments = append(comments, Comment{Type: "general", Body: e.Note, Code: e.Code, Submitted: e.Submitted})
			continue
		}
		end := e.EndLine
		if end == 0 {
			end = e.StartLine
		}
		start := 0
		if e.EndLine > e.StartLine {
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
			Submitted:    e.Submitted,
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

func countOr1(s string) int {
	if s == "" {
		return 1
	}
	n, _ := strconv.Atoi(s)
	return n
}

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
