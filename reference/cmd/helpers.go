// Package cmd provides CLI command implementations for projector.
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ideaspaper/projector/pkg/config"
	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/output"
	"github.com/ideaspaper/projector/pkg/paths"
	"github.com/ideaspaper/projector/pkg/storage"
)

var errSelectionCancelled = errors.New("selection cancelled")

// TypeFilter specifies which project types to include when loading projects.
type TypeFilter struct {
	Favorites bool
	Git       bool
	SVN       bool
	Mercurial bool
	VSCode    bool
	Any       bool
}

// ShowAll returns true if no specific type filter is set.
func (f TypeFilter) ShowAll() bool {
	return !f.Favorites && !f.Git && !f.SVN && !f.Mercurial && !f.VSCode && !f.Any
}

// LoadFilteredProjects loads projects from storage based on the given type filter.
// It returns all matching projects from both favorites and cache.
func LoadFilteredProjects(store *storage.Storage, filter TypeFilter) ([]*models.Project, error) {
	var allProjects []*models.Project
	showAll := filter.ShowAll()

	// Load favorites
	if showAll || filter.Favorites {
		projects, err := store.LoadProjects()
		if err != nil {
			return nil, fmt.Errorf("failed to load projects: %w", err)
		}
		allProjects = append(allProjects, projects.Projects...)
	}

	// Load cached auto-detected projects
	if showAll || filter.Git || filter.SVN || filter.Mercurial || filter.VSCode || filter.Any {
		cache, err := store.LoadCache()
		if err != nil {
			return nil, fmt.Errorf("failed to load cached projects: %w (run 'projector clear-cache' or 'projector scan' to rebuild the cache)", err)
		}

		if showAll || filter.Git {
			allProjects = append(allProjects, cache.Git...)
		}
		if showAll || filter.SVN {
			allProjects = append(allProjects, cache.SVN...)
		}
		if showAll || filter.Mercurial {
			allProjects = append(allProjects, cache.Mercurial...)
		}
		if showAll || filter.VSCode {
			allProjects = append(allProjects, cache.VSCode...)
		}
		if showAll || filter.Any {
			allProjects = append(allProjects, cache.Any...)
		}
	}

	return allProjects, nil
}

// MarkInvalidProjectsDisabled marks any projects with inaccessible paths as disabled.
func MarkInvalidProjectsDisabled(projects []*models.Project) {
	for _, p := range projects {
		if _, err := os.Stat(p.RootPath); err != nil {
			p.Enabled = false
		}
	}
}

// ExcludeProjectByPath removes a project with the given path from the list.
func ExcludeProjectByPath(projects []*models.Project, path string) []*models.Project {
	normalizedPath := normalizeProjectPath(path)
	filtered := make([]*models.Project, 0, len(projects))
	for _, p := range projects {
		if normalizeProjectPath(p.RootPath) == normalizedPath {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

// RemoveCurrentProject removes the current working directory from the project list.
func RemoveCurrentProject(projects []*models.Project) []*models.Project {
	currentPath, err := os.Getwd()
	if err != nil {
		return projects
	}

	return ExcludeProjectByPath(projects, currentPath)
}

func normalizeProjectPath(path string) string {
	normalized := filepath.Clean(paths.Expand(path))
	if resolved, err := filepath.EvalSymlinks(normalized); err == nil {
		normalized = resolved
	}

	if runtime.GOOS == "windows" {
		return strings.ToLower(normalized)
	}

	return normalized
}

// FilterEnabled returns only enabled projects from the given list.
func FilterEnabled(projects []*models.Project) []*models.Project {
	filtered := make([]*models.Project, 0, len(projects))
	for _, p := range projects {
		if p.Enabled {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FilterByTag returns only projects that have the specified tag.
func FilterByTag(projects []*models.Project, tag string) []*models.Project {
	if tag == "" {
		return projects
	}
	filtered := make([]*models.Project, 0)
	for _, p := range projects {
		if p.HasTag(tag) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FindProjectByName finds a project by name with exact or partial matching.
// Returns the matched project and any error.
// If multiple partial matches are found, returns an error with the matches.
func FindProjectByName(projects []*models.Project, name string) (*models.Project, []*models.Project, error) {
	return FindProjectByQuery(projects, name, false)
}

// FindProjectByQuery finds a project by name and optionally by full path.
func FindProjectByQuery(projects []*models.Project, query string, filterOnFullPath bool) (*models.Project, []*models.Project, error) {
	// First try exact match (case-insensitive)
	for _, p := range projects {
		if strings.EqualFold(p.Name, query) {
			return p, nil, nil
		}
	}

	if filterOnFullPath {
		normalizedQuery := normalizeProjectPath(query)
		for _, p := range projects {
			if normalizeProjectPath(p.RootPath) == normalizedQuery {
				return p, nil, nil
			}
		}
	}

	// Try partial match
	var matches []*models.Project
	queryLower := strings.ToLower(query)
	normalizedQuery := strings.ToLower(normalizeProjectPath(query))
	for _, p := range projects {
		if strings.Contains(strings.ToLower(p.Name), queryLower) {
			matches = append(matches, p)
			continue
		}

		if filterOnFullPath && strings.Contains(strings.ToLower(normalizeProjectPath(p.RootPath)), normalizedQuery) {
			matches = append(matches, p)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil, nil
	} else if len(matches) > 1 {
		return nil, matches, fmt.Errorf("multiple projects match '%s'", query)
	}

	return nil, nil, fmt.Errorf("project '%s' not found", query)
}

// ReportAmbiguousProjectMatches prints a user-facing list of ambiguous project matches.
func ReportAmbiguousProjectMatches(writer io.Writer, formatter *output.Formatter, query string, matches []*models.Project) {
	fmt.Fprintln(writer, formatter.FormatWarning(fmt.Sprintf("Multiple projects match '%s':", query)))
	for _, p := range matches {
		fmt.Fprintf(writer, "  - %s (%s)\n", p.Name, p.RootPath)
	}
}

// ReadUserInput reads a line of input from stdin, handling edge cases properly.
func ReadUserInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// logVerbose prints a message if verbose mode is enabled.
func logVerbose(cfg *config.Config, format string, args ...interface{}) {
	if verbose {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}
