package note

import (
	"fmt"
	"regexp"
	"strconv"
)

// Location identifies what a note is attached to.
//
//   - General note:    File == "" (StartLine and EndLine are 0).
//   - Whole file:      File set, StartLine == 0.
//   - Single line:     File set, StartLine >= 1, EndLine == 0.
//   - Line block/range: File set, StartLine >= 1, EndLine >= StartLine.
type Location struct {
	File      string
	StartLine int
	EndLine   int
}

var (
	rangeSpec  = regexp.MustCompile(`^(.+):(\d+)-(\d+)$`)
	singleSpec = regexp.MustCompile(`^(.+):(\d+)$`)
)

// ParseSpec parses a location spec of the form:
//
//	path/to/file.go          -> whole file
//	path/to/file.go:14       -> single line
//	path/to/file.go:1-17     -> line block (1 through 17 inclusive)
//
// It returns an error for empty input or a malformed/negative range.
func ParseSpec(spec string) (Location, error) {
	if spec == "" {
		return Location{}, fmt.Errorf("empty location")
	}

	if m := rangeSpec.FindStringSubmatch(spec); m != nil {
		start, _ := strconv.Atoi(m[2])
		end, _ := strconv.Atoi(m[3])
		if start < 1 {
			return Location{}, fmt.Errorf("invalid start line %d in %q", start, spec)
		}
		if end < start {
			return Location{}, fmt.Errorf("end line %d is before start line %d in %q", end, start, spec)
		}
		return Location{File: m[1], StartLine: start, EndLine: end}, nil
	}

	if m := singleSpec.FindStringSubmatch(spec); m != nil {
		line, _ := strconv.Atoi(m[2])
		if line < 1 {
			return Location{}, fmt.Errorf("invalid line %d in %q", line, spec)
		}
		return Location{File: m[1], StartLine: line}, nil
	}

	return Location{File: spec}, nil
}

// IsRange reports whether the location spans more than one line.
func (l Location) IsRange() bool { return l.EndLine > l.StartLine }

// Label renders the location for display: "file:start-end", "file:line",
// "file", or "(general)".
func (l Location) Label() string {
	switch {
	case l.File == "":
		return "(general)"
	case l.IsRange():
		return fmt.Sprintf("%s:%d-%d", l.File, l.StartLine, l.EndLine)
	case l.StartLine > 0:
		return fmt.Sprintf("%s:%d", l.File, l.StartLine)
	default:
		return l.File
	}
}
