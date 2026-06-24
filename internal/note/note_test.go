package note

import (
	"context"
	"reflect"
	"testing"
)

func TestParseSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    Location
		wantErr bool
	}{
		{name: "range", spec: "pkg/config/config.go:1-17", want: Location{File: "pkg/config/config.go", StartLine: 1, EndLine: 17}},
		{name: "single line", spec: "pkg/config/config.go:14", want: Location{File: "pkg/config/config.go", StartLine: 14}},
		{name: "whole file", spec: "pkg/config/config.go", want: Location{File: "pkg/config/config.go"}},
		{name: "path with colon-ish name", spec: "a/b.go:5", want: Location{File: "a/b.go", StartLine: 5}},
		{name: "empty", spec: "", wantErr: true},
		{name: "zero line", spec: "f.go:0", wantErr: true},
		{name: "reversed range", spec: "f.go:10-3", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSpec(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseSpec(%q) = %+v, want error", tt.spec, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSpec(%q) unexpected error: %v", tt.spec, err)
			}
			if got != tt.want {
				t.Errorf("ParseSpec(%q) = %+v, want %+v", tt.spec, got, tt.want)
			}
		})
	}
}

func TestLocationLabel(t *testing.T) {
	tests := []struct {
		loc  Location
		want string
	}{
		{Location{}, "(general)"},
		{Location{File: "f.go"}, "f.go"},
		{Location{File: "f.go", StartLine: 14}, "f.go:14"},
		{Location{File: "f.go", StartLine: 1, EndLine: 17}, "f.go:1-17"},
	}
	for _, tt := range tests {
		if got := tt.loc.Label(); got != tt.want {
			t.Errorf("Label(%+v) = %q, want %q", tt.loc, got, tt.want)
		}
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	entries := []Entry{
		{File: "pkg/config/config.go", StartLine: 1, EndLine: 17, Code: "package config\n\nimport (\n\t\"fmt, with comma\"\n)", Note: "needs a doc comment"},
		{File: "pkg/config/config.go", StartLine: 14, Code: `"github.com/spf13/viper"`, Note: "quoted \"value\" here"},
		{Note: "general note, with comma\nand newline"},
	}
	encoded, err := Marshal(entries)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := Unmarshal(encoded)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, entries) {
		t.Errorf("round trip mismatch:\n got = %+v\nwant = %+v", got, entries)
	}
}

func TestUnmarshalRejectsNonCSV(t *testing.T) {

	if _, err := Unmarshal("just a plain note\n"); err == nil {
		t.Fatal("expected error for non-CSV note")
	}
}

type fakeGit struct{ body string }

func (f fakeGit) ReadNote(context.Context, string) (string, bool) { return "", false }
func (f fakeGit) WriteNote(context.Context, string, string) error { return nil }
func (f fakeGit) RemoveNote(context.Context, string) error        { return nil }
func (f fakeGit) ShowFile(context.Context, string, string) (string, bool) {
	return f.body, true
}

func TestCaptureCode(t *testing.T) {
	src := "line1\nline2\nline3\nline4\n"
	m := &Manager{git: fakeGit{body: src}}
	ctx := context.Background()

	cases := []struct {
		name string
		loc  Location
		want string
	}{
		{"single", Location{File: "f", StartLine: 2}, "line2"},
		{"range", Location{File: "f", StartLine: 2, EndLine: 4}, "line2\nline3\nline4"},
		{"whole file captures nothing", Location{File: "f"}, ""},
		{"general", Location{}, ""},
		{"out of range start", Location{File: "f", StartLine: 99}, ""},
		{"clamped end", Location{File: "f", StartLine: 3, EndLine: 99}, "line3\nline4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := m.CaptureCode(ctx, "HEAD", tc.loc)
			if err != nil {
				t.Fatalf("CaptureCode: %v", err)
			}
			if got != tc.want {
				t.Errorf("CaptureCode(%+v) = %q, want %q", tc.loc, got, tc.want)
			}
		})
	}
}
