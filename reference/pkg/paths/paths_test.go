package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/projects", home + "/projects"},
		{"$home/projects", home + "/projects"},
		{"$HOME/projects", home + "/projects"},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", home},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Expand(tt.input)
			if result != tt.expected {
				t.Errorf("Expand(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCollapse(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{home + "/projects", "~/projects"},
		{home, "~"},
		{"/other/path", "/other/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Collapse(tt.input)
			if result != tt.expected {
				t.Errorf("Collapse(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandAll(t *testing.T) {
	home, _ := os.UserHomeDir()

	paths := []string{"~/a", "$HOME/b", "/c"}
	expanded := ExpandAll(paths)

	expected := []string{home + "/a", home + "/b", "/c"}
	for i, exp := range expected {
		if expanded[i] != exp {
			t.Errorf("ExpandAll[%d] = %q, want %q", i, expanded[i], exp)
		}
	}
}

func TestExpandAll_Empty(t *testing.T) {
	result := ExpandAll([]string{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Existing directory
	if !Exists(tmpDir) {
		t.Error("expected Exists to return true for existing dir")
	}

	// Existing file
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)
	if !Exists(tmpFile) {
		t.Error("expected Exists to return true for existing file")
	}

	// Non-existent
	if Exists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("expected Exists to return false for non-existent path")
	}
}

func TestIsDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Directory
	if !IsDir(tmpDir) {
		t.Error("expected IsDir to return true for directory")
	}

	// File
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)
	if IsDir(tmpFile) {
		t.Error("expected IsDir to return false for file")
	}

	// Non-existent
	if IsDir(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("expected IsDir to return false for non-existent path")
	}
}
