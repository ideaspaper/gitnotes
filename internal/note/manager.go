package note

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gitnotes/internal/gitcmd"
)

// git is the narrow view of gitcmd.Runner the manager depends on.
type git interface {
	ReadNote(ctx context.Context, commit string) (string, bool)
	WriteNote(ctx context.Context, commit, content string) error
	RemoveNote(ctx context.Context, commit string) error
	ShowFile(ctx context.Context, commit, file string) (string, bool)
}

// Manager performs note CRUD against a commit, storing entries as CSV.
type Manager struct {
	git git
}

// NewManager returns a Manager backed by the given git runner.
func NewManager(g gitcmd.Runner) *Manager {
	return &Manager{git: g}
}

// Read returns the entries attached to commit. A commit with no note yields
// nil. A note that is not the expected CSV (e.g. hand-written with `git notes
// add`) is surfaced as a single general entry so it is never silently lost.
func (m *Manager) Read(ctx context.Context, commit string) ([]Entry, error) {
	raw, ok := m.git.ReadNote(ctx, commit)
	if !ok {
		return nil, nil
	}
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	entries, err := Unmarshal(raw)
	if err != nil {
		return []Entry{{Note: strings.TrimRight(raw, "\n")}}, nil
	}
	return entries, nil
}

// Write persists entries to commit. An empty slice removes the note entirely.
func (m *Manager) Write(ctx context.Context, commit string, entries []Entry) error {
	if len(entries) == 0 {
		if err := m.git.RemoveNote(ctx, commit); err != nil {
			return fmt.Errorf("removing note: %w", err)
		}
		return nil
	}
	content, err := Marshal(entries)
	if err != nil {
		return fmt.Errorf("encoding note: %w", err)
	}
	if err := m.git.WriteNote(ctx, commit, content); err != nil {
		return fmt.Errorf("writing note: %w", err)
	}
	return nil
}

// Add captures the code at loc (as of commit) and appends a new entry. It
// returns the full entry list after the append.
func (m *Manager) Add(ctx context.Context, commit string, loc Location, text string) ([]Entry, error) {
	code, err := m.CaptureCode(ctx, commit, loc)
	if err != nil {
		return nil, err
	}
	existing, err := m.Read(ctx, commit)
	if err != nil {
		return nil, err
	}
	entries := append(existing, Entry{
		File:      loc.File,
		StartLine: loc.StartLine,
		EndLine:   loc.EndLine,
		Code:      code,
		Note:      text,
	})
	if err := m.Write(ctx, commit, entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// CaptureCode returns the source lines covered by loc, read from the file as of
// commit and falling back to the working tree. A general note (no file) yields
// an empty string; a whole-file location also captures nothing (only ranges and
// single lines snapshot code).
func (m *Manager) CaptureCode(ctx context.Context, commit string, loc Location) (string, error) {
	if loc.File == "" || loc.StartLine == 0 {
		return "", nil
	}
	lines, err := m.fileLines(ctx, commit, loc.File)
	if err != nil {
		return "", err
	}
	end := loc.EndLine
	if end == 0 {
		end = loc.StartLine
	}
	if loc.StartLine > len(lines) {
		return "", nil
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[loc.StartLine-1:end], "\n"), nil
}

// fileLines returns the lines of file as of commit, falling back to the working
// tree when the file is absent from the commit.
func (m *Manager) fileLines(ctx context.Context, commit, file string) ([]string, error) {
	if content, ok := m.git.ShowFile(ctx, commit, file); ok {
		return splitLines(content), nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s at %s or in the working tree: %w", file, commit, err)
	}
	return splitLines(string(data)), nil
}

// splitLines splits content into lines without a trailing empty element.
func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}
