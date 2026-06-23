// Package output provides formatting utilities for CLI output,
// including colored text, project lists, and status messages.
package output

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/ideaspaper/projector/pkg/models"
)

// MaxPathDisplayLength is the maximum length for displaying truncated paths.
const MaxPathDisplayLength = 50

// Formatter handles output formatting
type Formatter struct {
	colored bool

	// Colors
	nameColor    *color.Color
	pathColor    *color.Color
	tagColor     *color.Color
	kindColor    *color.Color
	successColor *color.Color
	errorColor   *color.Color
	warnColor    *color.Color
	infoColor    *color.Color
}

// NewFormatter creates a new formatter
func NewFormatter(colored bool) *Formatter {
	return &Formatter{
		colored:      colored,
		nameColor:    color.New(color.FgWhite, color.Bold),
		pathColor:    color.New(color.FgCyan),
		tagColor:     color.New(color.FgMagenta),
		kindColor:    color.New(color.FgYellow),
		successColor: color.New(color.FgGreen),
		errorColor:   color.New(color.FgRed),
		warnColor:    color.New(color.FgYellow),
		infoColor:    color.New(color.FgBlue),
	}
}

// ListOptions configures how FormatProjectList displays projects
type ListOptions struct {
	ShowPath               bool // Show full path on separate line
	ShowIndex              bool // Show index numbers for selection
	Grouped                bool // Group by project kind
	ShowParentOnDuplicates bool // Append parent folder for duplicate names
}

// formatProjectItem formats a single project item
func (f *Formatter) formatProjectItem(p *models.Project, index int, opts ListOptions, indent string, duplicateNames map[string]bool) string {
	var sb strings.Builder

	sb.WriteString(indent)

	// Index (1-based for user display)
	if opts.ShowIndex {
		if f.colored {
			sb.WriteString(f.infoColor.Sprintf("[%d]", index))
		} else {
			sb.WriteString(fmt.Sprintf("[%d]", index))
		}
		sb.WriteString(" ")
	}

	// Name
	name := p.Name
	if opts.ShowParentOnDuplicates && duplicateNames[strings.ToLower(p.Name)] {
		parent := filepath.Base(filepath.Dir(p.RootPath))
		if parent != "." && parent != string(filepath.Separator) && parent != "" && parent != p.Name {
			name = fmt.Sprintf("%s (%s)", p.Name, parent)
		}
	}

	if f.colored {
		sb.WriteString(f.nameColor.Sprint(name))
	} else {
		sb.WriteString(name)
	}

	// Tags
	if len(p.Tags) > 0 {
		sb.WriteString(" ")
		if f.colored {
			sb.WriteString(f.tagColor.Sprintf("[%s]", strings.Join(p.Tags, ", ")))
		} else {
			sb.WriteString(fmt.Sprintf("[%s]", strings.Join(p.Tags, ", ")))
		}
	}

	// Disabled indicator
	if !p.Enabled {
		if f.colored {
			sb.WriteString(f.warnColor.Sprint(" (disabled)"))
		} else {
			sb.WriteString(" (disabled)")
		}
	}

	// Path
	path := p.RootPath
	if opts.ShowPath {
		// Full path on new line
		sb.WriteString("\n")
		sb.WriteString(indent)
		if opts.ShowIndex {
			sb.WriteString("    ") // Extra indent to align with name
		}
		if f.colored {
			sb.WriteString(f.pathColor.Sprint(path))
		} else {
			sb.WriteString(path)
		}
	} else {
		// Truncated path on same line
		sb.WriteString(" - ")
		if len(path) > MaxPathDisplayLength {
			path = "..." + path[len(path)-(MaxPathDisplayLength-3):]
		}
		if f.colored {
			sb.WriteString(f.pathColor.Sprint(path))
		} else {
			sb.WriteString(path)
		}
	}

	return sb.String()
}

// FormatProjectList formats a list of projects with the given options
// Returns the formatted string and a slice mapping display index (1-based) to project
func (f *Formatter) FormatProjectList(projects []*models.Project, opts ListOptions) (string, []*models.Project) {
	if len(projects) == 0 {
		return f.FormatInfo("No projects found."), nil
	}

	var sb strings.Builder
	indexedProjects := make([]*models.Project, 0, len(projects))
	currentIndex := 1 // 1-based index
	duplicateNames := make(map[string]bool)
	nameCounts := make(map[string]int)
	for _, p := range projects {
		nameCounts[strings.ToLower(p.Name)]++
	}
	for name, count := range nameCounts {
		if count > 1 {
			duplicateNames[name] = true
		}
	}

	if opts.Grouped {
		// Group by kind
		groups := make(map[models.ProjectKind][]*models.Project)
		for _, p := range projects {
			groups[p.Kind] = append(groups[p.Kind], p)
		}

		kindOrder := []models.ProjectKind{
			models.KindFavorite,
			models.KindGit,
			models.KindSVN,
			models.KindMercurial,
			models.KindVSCode,
			models.KindAny,
		}

		for _, kind := range kindOrder {
			ps, ok := groups[kind]
			if !ok || len(ps) == 0 {
				continue
			}

			// Group header
			header := f.getKindHeader(kind)
			if f.colored {
				sb.WriteString(f.kindColor.Sprint(header))
			} else {
				sb.WriteString(header)
			}
			sb.WriteString("\n")

			for _, p := range ps {
				sb.WriteString(f.formatProjectItem(p, currentIndex, opts, "  ", duplicateNames))
				sb.WriteString("\n")
				indexedProjects = append(indexedProjects, p)
				currentIndex++
			}
			sb.WriteString("\n")
		}
	} else {
		for _, p := range projects {
			sb.WriteString(f.formatProjectItem(p, currentIndex, opts, "", duplicateNames))
			sb.WriteString("\n")
			indexedProjects = append(indexedProjects, p)
			currentIndex++
		}
	}

	return strings.TrimSuffix(sb.String(), "\n"), indexedProjects
}

// getKindHeader returns the header for a project kind
func (f *Formatter) getKindHeader(kind models.ProjectKind) string {
	switch kind {
	case models.KindFavorite:
		return "Favorites"
	case models.KindGit:
		return "Git Repositories"
	case models.KindSVN:
		return "SVN Repositories"
	case models.KindMercurial:
		return "Mercurial Repositories"
	case models.KindVSCode:
		return "VS Code Workspaces"
	case models.KindAny:
		return "Other Projects"
	default:
		return "Projects"
	}
}

// FormatSuccess formats a success message
func (f *Formatter) FormatSuccess(msg string) string {
	if f.colored {
		return f.successColor.Sprint("✓ " + msg)
	}
	return "✓ " + msg
}

// FormatError formats an error message
func (f *Formatter) FormatError(msg string) string {
	if f.colored {
		return f.errorColor.Sprint("✗ " + msg)
	}
	return "✗ " + msg
}

// FormatWarning formats a warning message
func (f *Formatter) FormatWarning(msg string) string {
	if f.colored {
		return f.warnColor.Sprint("⚠ " + msg)
	}
	return "⚠ " + msg
}

// FormatInfo formats an info message
func (f *Formatter) FormatInfo(msg string) string {
	if f.colored {
		return f.infoColor.Sprint("ℹ " + msg)
	}
	return "ℹ " + msg
}
