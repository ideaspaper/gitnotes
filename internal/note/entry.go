package note

import (
	"encoding/csv"
	"strconv"
	"strings"
)

const numFields = 6

type Entry struct {
	File      string
	StartLine int
	EndLine   int
	Code      string
	Note      string
	Submitted bool
}

func (e Entry) Location() Location {
	return Location{File: e.File, StartLine: e.StartLine, EndLine: e.EndLine}
}

func Marshal(entries []Entry) (string, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)
	for _, e := range entries {
		rec := []string{
			e.File,
			intOrEmpty(e.StartLine),
			intOrEmpty(e.EndLine),
			e.Code,
			e.Note,
			boolOrEmpty(e.Submitted),
		}
		if err := w.Write(rec); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func Unmarshal(raw string) ([]Entry, error) {
	r := csv.NewReader(strings.NewReader(raw))
	r.FieldsPerRecord = numFields
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(records))
	for _, rec := range records {
		entries = append(entries, Entry{
			File:      rec[0],
			StartLine: atoiOrZero(rec[1]),
			EndLine:   atoiOrZero(rec[2]),
			Code:      rec[3],
			Note:      rec[4],
			Submitted: parseBool(rec[5]),
		})
	}
	return entries, nil
}

func intOrEmpty(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func atoiOrZero(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func boolOrEmpty(b bool) string {
	if b {
		return "true"
	}
	return ""
}

func parseBool(s string) bool {
	return strings.TrimSpace(s) == "true"
}
