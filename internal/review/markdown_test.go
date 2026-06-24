package review

import (
	"strings"
	"testing"

	"gitnotes/internal/note"
)

func TestMarkdown(t *testing.T) {
	entries := []note.Entry{
		{File: "cli.go", StartLine: 3, Code: "const appName = \"gitnotes\"", Note: "good", Submitted: true},
		{Note: "overall LGTM"},
	}
	md := Markdown("abc1234", "Add cli", entries)

	for _, want := range []string{
		"# Review notes (abc1234 — Add cli)",
		"2 note(s).",
		"## #1 — `cli.go:3`",
		"```go\nconst appName = \"gitnotes\"\n```",
		"## #2 — `(general)`",
		"overall LGTM",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("Markdown() missing %q in:\n%s", want, md)
		}
	}

	if strings.ContainsAny(md, "✓✗") {
		t.Errorf("Markdown() should not include a submitted marker:\n%s", md)
	}

	codeAt := strings.Index(md, "```go")
	noteAt := strings.Index(md, "good")
	if codeAt < 0 || noteAt < 0 || codeAt > noteAt {
		t.Errorf("expected code block before the note text; code@%d note@%d", codeAt, noteAt)
	}
}

func TestMarkdownEmpty(t *testing.T) {
	if md := Markdown("abc", "", nil); !strings.Contains(md, "_No notes._") {
		t.Errorf("empty Markdown() = %q, want a no-notes marker", md)
	}
}

func TestFenceForEscapesBackticks(t *testing.T) {
	code := "a\n```\nb"
	if fence := fenceFor(code); len(fence) <= 3 {
		t.Errorf("fenceFor(%q) = %q, want longer than ```", code, fence)
	}
	if fence := fenceFor("plain code"); fence != "```" {
		t.Errorf("fenceFor(plain) = %q, want ```", fence)
	}
}
