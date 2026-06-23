// Package scanner provides functionality for detecting and scanning
// repositories (Git, SVN, Mercurial), VS Code workspaces, and arbitrary
// folders within specified base directories.
package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/paths"
)

// ScannerType represents the type of repository scanner
type ScannerType string

const (
	ScannerGit       ScannerType = "git"
	ScannerSVN       ScannerType = "svn"
	ScannerMercurial ScannerType = "mercurial"
	ScannerVSCode    ScannerType = "vscode"
	ScannerAny       ScannerType = "any"
)

// ErrorHandler is a callback for handling scan errors
type ErrorHandler func(path string, err error)

// Scanner scans directories for projects
type Scanner struct {
	baseFolders          []string
	ignoredFolders       []string
	maxDepth             int
	scannerType          ScannerType
	ignoreWithinProjects bool
	supportSymlinks      bool
	errorHandler         ErrorHandler
}

// NewScanner creates a new project scanner
func NewScanner(scannerType ScannerType) *Scanner {
	return &Scanner{
		baseFolders:          []string{},
		ignoredFolders:       []string{"node_modules", "out", "typings", "test", "vendor"},
		maxDepth:             4,
		scannerType:          scannerType,
		ignoreWithinProjects: false,
		supportSymlinks:      false,
	}
}

// SetBaseFolders sets the base folders to scan
func (s *Scanner) SetBaseFolders(folders []string) {
	s.baseFolders = paths.ExpandAll(folders)
}

// SetIgnoredFolders sets the folders to ignore during scanning
func (s *Scanner) SetIgnoredFolders(folders []string) {
	s.ignoredFolders = folders
}

// SetMaxDepth sets the maximum recursion depth
func (s *Scanner) SetMaxDepth(depth int) {
	s.maxDepth = depth
}

// SetIgnoreWithinProjects sets whether to ignore projects within other projects
func (s *Scanner) SetIgnoreWithinProjects(ignore bool) {
	s.ignoreWithinProjects = ignore
}

// SetSupportSymlinks sets whether to follow symlinks
func (s *Scanner) SetSupportSymlinks(support bool) {
	s.supportSymlinks = support
}

// SetErrorHandler sets the callback for handling scan errors
func (s *Scanner) SetErrorHandler(handler ErrorHandler) {
	s.errorHandler = handler
}

// logError calls the error handler if set
func (s *Scanner) logError(path string, err error) {
	if s.errorHandler != nil {
		s.errorHandler(path, err)
	}
}

// Scan scans all base folders for projects
func (s *Scanner) Scan() ([]*models.Project, error) {
	var projects []*models.Project
	seen := make(map[string]bool)

	for _, baseFolder := range s.baseFolders {
		if _, err := os.Stat(baseFolder); os.IsNotExist(err) {
			s.logError(baseFolder, fmt.Errorf("base folder does not exist: %w", err))
			continue
		}

		found, err := s.scanFolder(baseFolder, 0, false)
		if err != nil {
			s.logError(baseFolder, fmt.Errorf("failed to scan folder: %w", err))
			continue
		}

		for _, project := range found {
			if !seen[project.RootPath] {
				seen[project.RootPath] = true
				projects = append(projects, project)
			}
		}
	}

	// Deduplicate names by adding suffix
	deduplicateNames(projects)

	return projects, nil
}

// deduplicateNames adds numeric suffixes to projects with duplicate names
// e.g., "api", "api-2", "api-3"
func deduplicateNames(projects []*models.Project) {
	// Count occurrences of each name
	nameCount := make(map[string]int)
	for _, p := range projects {
		nameCount[p.Name]++
	}

	// For names that appear more than once, add suffixes
	nameSeen := make(map[string]int)
	for _, p := range projects {
		if nameCount[p.Name] > 1 {
			nameSeen[p.Name]++
			if nameSeen[p.Name] == 1 {
				// Keep first occurrence as-is
				continue
			}
			// Add suffix for subsequent occurrences
			p.Name = fmt.Sprintf("%s-%d", p.Name, nameSeen[p.Name])
		}
	}
}

// scanFolder recursively scans a folder for projects
func (s *Scanner) scanFolder(folder string, depth int, insideProject bool) ([]*models.Project, error) {
	var projects []*models.Project

	if depth > s.maxDepth {
		return projects, nil
	}

	// Check if current folder is a project of this type
	isProject := s.isProject(folder)

	if isProject {
		if !s.ignoreWithinProjects || !insideProject {
			project := &models.Project{
				Name:     filepath.Base(folder),
				RootPath: folder,
				Tags:     []string{},
				Enabled:  true,
				Kind:     s.getProjectKind(),
			}
			projects = append(projects, project)
		}
		insideProject = true
	}

	if s.ignoreWithinProjects && insideProject {
		return projects, nil
	}

	// Scan subdirectories
	entries, err := os.ReadDir(folder)
	if err != nil {
		s.logError(folder, fmt.Errorf("failed to read directory: %w", err))
		return projects, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden directories (except .vscode for workspace detection)
		if strings.HasPrefix(name, ".") && name != ".vscode" {
			continue
		}

		// Skip ignored folders
		if s.isIgnored(name) {
			continue
		}

		subPath := filepath.Join(folder, name)

		// Handle symlinks
		if entry.Type()&os.ModeSymlink != 0 {
			if !s.supportSymlinks {
				continue
			}
			// Resolve symlink
			resolved, err := filepath.EvalSymlinks(subPath)
			if err != nil {
				s.logError(subPath, fmt.Errorf("failed to resolve symlink: %w", err))
				continue
			}
			subPath = resolved
		}

		subProjects, err := s.scanFolder(subPath, depth+1, insideProject)
		if err != nil {
			s.logError(subPath, fmt.Errorf("failed to scan subfolder: %w", err))
			continue
		}
		projects = append(projects, subProjects...)
	}

	return projects, nil
}

// isProject checks if a folder is a project of the scanner's type
func (s *Scanner) isProject(folder string) bool {
	switch s.scannerType {
	case ScannerGit:
		return dirExists(filepath.Join(folder, ".git"))
	case ScannerSVN:
		return dirExists(filepath.Join(folder, ".svn"))
	case ScannerMercurial:
		return dirExists(filepath.Join(folder, ".hg"))
	case ScannerVSCode:
		return fileExistsWithExt(folder, ".code-workspace")
	case ScannerAny:
		return true // Any folder counts as a project
	default:
		return false
	}
}

// isIgnored checks if a folder name should be ignored
func (s *Scanner) isIgnored(name string) bool {
	for _, ignored := range s.ignoredFolders {
		// Support simple glob patterns
		if strings.Contains(ignored, "*") {
			matched, _ := filepath.Match(ignored, name)
			if matched {
				return true
			}
		} else if name == ignored {
			return true
		}
	}
	return false
}

// getProjectKind returns the project kind for this scanner
func (s *Scanner) getProjectKind() models.ProjectKind {
	switch s.scannerType {
	case ScannerGit:
		return models.KindGit
	case ScannerSVN:
		return models.KindSVN
	case ScannerMercurial:
		return models.KindMercurial
	case ScannerVSCode:
		return models.KindVSCode
	case ScannerAny:
		return models.KindAny
	default:
		return models.KindFavorite
	}
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// fileExistsWithExt checks if any file with the given extension exists in the folder
func fileExistsWithExt(folder, ext string) bool {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ext) {
			return true
		}
	}
	return false
}
