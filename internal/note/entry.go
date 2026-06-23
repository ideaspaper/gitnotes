package note

import (
	"encoding/csv"
	"strconv"
	"strings"
)

// numFields is the fixed column count of a note row.
const numFields = 5

// Entry is one note attached to a commit. A commit's note is a CSV document
// with one Entry per row, in column order: file, startLine, endLine, code, note.
//
// A zero StartLine means the entry is general (when File is empty) or
// file-level (when File is set). A zero EndLine means a single line.
type Entry struct {
	File      string
	StartLine int
	EndLine   int
	Code      string
	Note      string
}

// Location returns the entry's location.
func (e Entry) Location() Location {
	return Location{File: e.File, StartLine: e.StartLine, EndLine: e.EndLine}
}

// Marshal encodes entries as a CSV document. Multi-line code and notes are
// quoted by encoding/csv, so a round-trip through Unmarshal is lossless.
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

// Unmarshal decodes a CSV note document into entries. It returns an error when
// the input is not the expected 5-column CSV; callers treat that case as a
// hand-written (non-structured) note.
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
