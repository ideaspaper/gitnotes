package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/ideaspaper/projector/pkg/config"
	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/output"
	"github.com/ideaspaper/projector/pkg/storage"
)

// Editor constants for supported editors
const (
	EditorCode     = "code"
	EditorVSCode   = "vscode"
	EditorCursor   = "cursor"
	EditorSublime  = "subl"
	EditorSublAlt  = "sublime"
	EditorAtom     = "atom"
	EditorVim      = "vim"
	EditorNeoVim   = "nvim"
	EditorEmacs    = "emacs"
	EditorIdea     = "idea"
	EditorIntelliJ = "intellij"
	EditorWebStorm = "webstorm"
	EditorGoLand   = "goland"
	EditorPyCharm  = "pycharm"
	EditorOpen     = "open"     // macOS
	EditorXdgOpen  = "xdg-open" // Linux
	EditorExplorer = "explorer" // Windows
)

var (
	openNewWindow bool
	openEditor    string
	openTag       string
	openGrouped   bool
	openFavorites bool
	openGit       bool
	openSVN       bool
	openMercurial bool
	openVSCode    bool
	openAny       bool
)

// openCmd represents the open command
var openCmd = &cobra.Command{
	Use:   "open [project-name]",
	Short: "Open a project in your editor",
	Long: `Open a project in your configured editor (default: VS Code).

If no project name is provided, an interactive selection is shown.

Examples:
  # Open a project by name
  projector open myproject

  # Open in a new window
  projector open myproject --new-window

  # Open with a specific editor
  projector open myproject --editor vim

  # Filter interactive selection by tag
  projector open --tag Work`,
	Args: cobra.MaximumNArgs(1),
	RunE: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)

	openCmd.Flags().BoolVarP(&openNewWindow, "new-window", "n", false, "open in a new window")
	openCmd.Flags().StringVarP(&openEditor, "editor", "e", "", "editor to use (overrides config)")
	openCmd.Flags().StringVarP(&openTag, "tag", "t", "", "filter projects by tag")
	openCmd.Flags().BoolVarP(&openGrouped, "grouped", "g", false, "group projects by type")
	openCmd.Flags().BoolVar(&openFavorites, "favorites", false, "show only favorites")
	openCmd.Flags().BoolVar(&openGit, "git", false, "show only git repositories")
	openCmd.Flags().BoolVar(&openSVN, "svn", false, "show only svn repositories")
	openCmd.Flags().BoolVar(&openMercurial, "mercurial", false, "show only mercurial repositories")
	openCmd.Flags().BoolVar(&openVSCode, "vscode", false, "show only vscode workspaces")
	openCmd.Flags().BoolVar(&openAny, "any", false, "show only any-folder projects")
}

func runOpen(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.LoadOrCreateConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage
	store, err := storage.NewStorage(cfg.GetProjectsLocation())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Load projects with type filter
	filter := TypeFilter{
		Favorites: openFavorites,
		Git:       openGit,
		SVN:       openSVN,
		Mercurial: openMercurial,
		VSCode:    openVSCode,
		Any:       openAny,
	}
	allProjects, err := LoadFilteredProjects(store, filter)
	if err != nil {
		return err
	}

	if cfg.CheckInvalidPaths {
		MarkInvalidProjectsDisabled(allProjects)
	}

	if cfg.RemoveCurrentFromList {
		allProjects = RemoveCurrentProject(allProjects)
	}

	// Filter enabled only
	allProjects = FilterEnabled(allProjects)

	// Filter by tag if specified
	allProjects = FilterByTag(allProjects, openTag)

	if len(allProjects) == 0 {
		return fmt.Errorf("no projects found")
	}

	// Find project
	var selectedProject *models.Project

	if len(args) > 0 {
		projectName := args[0]
		var matches []*models.Project
		selectedProject, matches, err = FindProjectByQuery(allProjects, projectName, cfg.FilterOnFullPath)
		if err != nil {
			if len(matches) > 0 {
				formatter := output.NewFormatter(!noColor && cfg.ShowColors)
				ReportAmbiguousProjectMatches(os.Stdout, formatter, projectName, matches)
				return fmt.Errorf("please be more specific")
			}
			return err
		}
	} else {
		// Interactive selection
		selectedProject, err = selectProjectInteractive(cmd, allProjects, cfg)
		if err != nil {
			if errors.Is(err, errSelectionCancelled) {
				return nil
			}
			return err
		}
	}

	// Verify path exists
	if _, err := os.Stat(selectedProject.RootPath); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", selectedProject.RootPath)
	}

	// Determine editor
	editor := openEditor
	if editor == "" {
		editor = cfg.Editor
	}

	// Open project
	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	fmt.Println(formatter.FormatInfo(fmt.Sprintf("Opening '%s' in %s...", selectedProject.Name, editor)))

	return openInEditor(selectedProject.RootPath, editor, openNewWindow || cfg.OpenInNewWindow)
}

// selectProjectInteractive shows an interactive selection menu
func selectProjectInteractive(cmd *cobra.Command, projects []*models.Project, cfg *config.Config) (*models.Project, error) {
	// Sort according to config
	sortProjects(projects, cfg.SortList)

	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	fmt.Println("Select a project to open:")
	fmt.Println()

	// Determine grouping: flag takes precedence if explicitly set
	grouped := cfg.GroupList
	if cmd.Flags().Changed("grouped") {
		grouped = openGrouped
	}

	// Use grouped display based on config
	opts := output.ListOptions{
		ShowPath:               false,
		ShowIndex:              true,
		Grouped:                grouped,
		ShowParentOnDuplicates: cfg.ShowParentOnDuplicates,
	}
	listOutput, indexedProjects := formatter.FormatProjectList(projects, opts)
	fmt.Println(listOutput)
	fmt.Println()

	fmt.Print("Enter project number (or 'q' to quit): ")
	input, err := ReadUserInput()
	if err != nil {
		return nil, err
	}

	if input == "q" || input == "Q" {
		return nil, errSelectionCancelled
	}

	var index int
	if _, err := fmt.Sscanf(input, "%d", &index); err != nil {
		return nil, fmt.Errorf("invalid selection")
	}

	// Convert 1-based input to 0-based index
	index--
	if index < 0 || index >= len(indexedProjects) {
		return nil, fmt.Errorf("invalid selection: index out of range")
	}

	return indexedProjects[index], nil
}

// openInEditor opens a path in the specified editor
func openInEditor(path, editor string, newWindow bool) error {
	var cmd *exec.Cmd

	switch editor {
	case EditorCode, EditorVSCode:
		args := []string{path}
		if newWindow {
			args = append([]string{"--new-window"}, args...)
		}
		cmd = exec.Command(EditorCode, args...)

	case EditorCursor:
		args := []string{path}
		if newWindow {
			args = append([]string{"--new-window"}, args...)
		}
		cmd = exec.Command(EditorCursor, args...)

	case EditorSublime, EditorSublAlt:
		args := []string{path}
		if newWindow {
			args = append([]string{"--new-window"}, args...)
		}
		cmd = exec.Command(EditorSublime, args...)

	case EditorAtom:
		args := []string{path}
		if newWindow {
			args = append([]string{"--new-window"}, args...)
		}
		cmd = exec.Command(EditorAtom, args...)

	case EditorVim, EditorNeoVim:
		cmd = exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

	case EditorEmacs:
		cmd = exec.Command(EditorEmacs, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

	case EditorIdea, EditorIntelliJ:
		cmd = exec.Command(EditorIdea, path)

	case EditorWebStorm:
		cmd = exec.Command(EditorWebStorm, path)

	case EditorGoLand:
		cmd = exec.Command(EditorGoLand, path)

	case EditorPyCharm:
		cmd = exec.Command(EditorPyCharm, path)

	case EditorOpen:
		// macOS open command
		cmd = exec.Command(EditorOpen, path)

	case EditorXdgOpen:
		// Linux open command
		cmd = exec.Command(EditorXdgOpen, path)

	case EditorExplorer:
		// Windows Explorer
		cmd = exec.Command(EditorExplorer, path)

	default:
		// Try to run the editor directly
		cmd = exec.Command(editor, path)
	}

	// For GUI editors, don't wait
	if editor != EditorVim && editor != EditorNeoVim && editor != EditorEmacs {
		return cmd.Start()
	}

	return cmd.Run()
}
