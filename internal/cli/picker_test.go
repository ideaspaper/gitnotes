package cli

import (
	"bytes"
	"strings"
	"testing"

	"gitnotes/internal/note"
)

func TestPickerFuzzyFilter(t *testing.T) {
	entries := []note.Entry{
		{File: "internal/cli/commands.go", StartLine: 42, Note: "use slog here"},
		{Note: "overall looks fine"},
		{File: "internal/note/entry.go", StartLine: 20, EndLine: 34, Note: "needs a doc comment"},
	}
	m := newPickerModel("", entries)

	all := m.filter("")
	if len(all) != 3 || all[0] != 0 || all[1] != 1 || all[2] != 2 {
		t.Fatalf("empty filter = %v, want [0 1 2]", all)
	}

	if got := m.filter("slog"); len(got) == 0 || m.rows[got[0]].idx != 0 {
		t.Errorf("filter(slog) top = %v, want entry 0", got)
	}

	if got := m.filter("entrygo"); len(got) == 0 || m.rows[got[0]].idx != 2 {
		t.Errorf("filter(entrygo) top = %v, want entry 2", got)
	}

	if got := m.filter("slg"); len(got) == 0 || m.rows[got[0]].idx != 0 {
		t.Errorf("filter(slg) typo-tolerant top = %v, want entry 0", got)
	}

	if got := m.filter("zzzzznope"); len(got) != 0 {
		t.Errorf("filter(no match) = %v, want empty", got)
	}
}

func TestFilterByID(t *testing.T) {
	entries := []note.Entry{
		{Note: "alpha note"},
		{Note: "beta note"},
		{Note: "gamma note"},
	}
	m := newPickerModel("", entries)

	for _, id := range []string{"3", "#3"} {
		got := m.filter(id)
		if len(got) == 0 || m.rows[got[0]].idx != 2 {
			t.Errorf("filter(%q) top = %v, want entry idx 2 (note #3)", id, got)
		}
	}
}

func TestFilterMatchesCode(t *testing.T) {
	entries := []note.Entry{
		{File: "main.go", StartLine: 5, Note: "entry point", Code: "fmt.Println(\"hi\")"},
		{Note: "general note"},
	}
	m := newPickerModel("", entries)

	if got := m.filter("println"); len(got) != 1 || m.rows[got[0]].idx != 0 {
		t.Errorf("filter(println) = %v, want only entry 0 (matched via captured code)", got)
	}
}

func TestFilterRejectsScattered(t *testing.T) {
	entries := []note.Entry{
		{File: "README.md", Note: "fix typo in usage"},
		{File: "internal/note/entry.go", Note: "needs a doc comment"},
	}
	m := newPickerModel("", entries)

	got := m.filter("readme")
	if len(got) != 1 || m.rows[got[0]].idx != 0 {
		t.Errorf("filter(readme) = %v, want only entry 0 (scattered match in entry 1 rejected)", got)
	}
}

func TestPickerView(t *testing.T) {
	entries := []note.Entry{
		{File: "f.go", StartLine: 2, Note: "fix this", Submitted: true},
		{Note: "general note"},
	}
	m := newPickerModel("abc123  subject", entries)
	out := m.View()
	for _, want := range []string{"abc123  subject", "LOCATION", "SUBMITTED", "f.go:2", "general note", "✓", "✗"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderDetail(t *testing.T) {
	var buf bytes.Buffer
	a := &app{out: &buf}
	a.renderDetail(3, note.Entry{
		File:      "main.go",
		StartLine: 10,
		EndLine:   12,
		Code:      "func main() {\n\tfmt.Println(\"hi\")\n}",
		Note:      "needs a doc comment",
		Submitted: true,
	})
	out := buf.String()
	for _, want := range []string{"Note #4", "main.go:10-12", "Submitted:", "yes", "func main()", "needs a doc comment"} {
		if !strings.Contains(out, want) {
			t.Errorf("renderDetail() missing %q in:\n%s", want, out)
		}
	}
}
