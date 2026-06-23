// Package paths provides utilities for path manipulation,
// including expansion and collapsing of home directory references.
package paths

import (
	"os"
	"strings"
)

// Expand expands ~ and $home/$HOME to the actual home directory.
func Expand(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if strings.HasPrefix(path, "~") {
		path = strings.Replace(path, "~", home, 1)
	}
	if strings.HasPrefix(path, "$home") {
		path = strings.Replace(path, "$home", home, 1)
	}
	if strings.HasPrefix(path, "$HOME") {
		path = strings.Replace(path, "$HOME", home, 1)
	}

	return path
}

// Collapse replaces the home directory with ~.
func Collapse(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if strings.HasPrefix(path, home) {
		return strings.Replace(path, home, "~", 1)
	}

	return path
}

// ExpandAll expands ~ and $home/$HOME in all paths.
func ExpandAll(paths []string) []string {
	result := make([]string, len(paths))
	for i, path := range paths {
		result[i] = Expand(path)
	}
	return result
}

// Exists checks if a path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path exists and is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
