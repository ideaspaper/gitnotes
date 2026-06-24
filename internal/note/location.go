package note

import (
	"fmt"
	"regexp"
	"strconv"
)

type Location struct {
	File      string
	StartLine int
	EndLine   int
}

var (
	rangeSpec  = regexp.MustCompile(`^(.+):(\d+)-(\d+)$`)
	singleSpec = regexp.MustCompile(`^(.+):(\d+)$`)
)

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

func (l Location) IsRange() bool { return l.EndLine > l.StartLine }

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
