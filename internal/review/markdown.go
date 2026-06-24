package review

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitnotes/internal/note"
)

func ExportMarkdown(short, subject string, entries []note.Entry, path string) error {
	if err := os.WriteFile(path, []byte(Markdown(short, subject, entries)), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func Markdown(short, subject string, entries []note.Entry) string {
	var b strings.Builder
	title := short
	if subject != "" {
		title = short + " — " + subject
	}
	fmt.Fprintf(&b, "# Review notes (%s)\n\n", title)

	if len(entries) == 0 {
		b.WriteString("_No notes._\n")
		return b.String()
	}

	fmt.Fprintf(&b, "%d note(s).\n", len(entries))
	for i, e := range entries {
		mark := "✗"
		if e.Submitted {
			mark = "✓"
		}
		fmt.Fprintf(&b, "\n## #%d — `%s` %s\n", i, e.Location().Label(), mark)
		if body := strings.TrimSpace(e.Note); body != "" {
			fmt.Fprintf(&b, "\n%s\n", body)
		}
		if code := strings.TrimRight(e.Code, "\n"); strings.TrimSpace(code) != "" {
			fence := fenceFor(code)
			fmt.Fprintf(&b, "\n%s%s\n%s\n%s\n", fence, langForFile(e.File), code, fence)
		}
	}
	return b.String()
}

func langForFile(file string) string {
	return strings.TrimPrefix(filepath.Ext(file), ".")
}

func fenceFor(code string) string {
	longest, cur := 0, 0
	for _, r := range code {
		if r == '`' {
			cur++
			if cur > longest {
				longest = cur
			}
		} else {
			cur = 0
		}
	}
	return strings.Repeat("`", max(longest+1, 3))
}
