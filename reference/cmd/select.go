package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ideaspaper/projector/pkg/config"
	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/output"
	"github.com/ideaspaper/projector/pkg/storage"
)

var (
	selectTag       string
	selectGrouped   bool
	selectFavorites bool
	selectGit       bool
	selectSVN       bool
	selectMercurial bool
	selectVSCode    bool
	selectAny       bool
)

// selectCmd represents the select command
var selectCmd = &cobra.Command{
	Use:   "select [project-name]",
	Short: "Select a project and output its path",
	Long: `Select a project and output its path to stdout.

If no project name is provided, an interactive selection is shown.
This is useful for scripting and shell integration.

Examples:
  # Interactive selection
  projector select

  # Select by name
  projector select myproject

  # Filter interactive selection by tag
  projector select --tag Work

Shell function for cd:
  pjcd() {
    local dir
    dir=$(projector select)
    [ -n "$dir" ] && [ -d "$dir" ] && cd "$dir"
  }`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSelect,
}

func init() {
	rootCmd.AddCommand(selectCmd)

	selectCmd.Flags().StringVarP(&selectTag, "tag", "t", "", "filter projects by tag")
	selectCmd.Flags().BoolVarP(&selectGrouped, "grouped", "g", false, "group projects by type")
	selectCmd.Flags().BoolVar(&selectFavorites, "favorites", false, "show only favorites")
	selectCmd.Flags().BoolVar(&selectGit, "git", false, "show only git repositories")
	selectCmd.Flags().BoolVar(&selectSVN, "svn", false, "show only svn repositories")
	selectCmd.Flags().BoolVar(&selectMercurial, "mercurial", false, "show only mercurial repositories")
	selectCmd.Flags().BoolVar(&selectVSCode, "vscode", false, "show only vscode workspaces")
	selectCmd.Flags().BoolVar(&selectAny, "any", false, "show only any-folder projects")
}

func runSelect(cmd *cobra.Command, args []string) error {
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
		Favorites: selectFavorites,
		Git:       selectGit,
		SVN:       selectSVN,
		Mercurial: selectMercurial,
		VSCode:    selectVSCode,
		Any:       selectAny,
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
	allProjects = FilterByTag(allProjects, selectTag)

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
				ReportAmbiguousProjectMatches(os.Stderr, formatter, projectName, matches)
				return fmt.Errorf("please be more specific")
			}
			return err
		}
	} else {
		// Interactive selection
		selectedProject, err = selectProjectForSelect(cmd, allProjects, cfg)
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

	// Output the path to stdout
	fmt.Println(selectedProject.RootPath)
	return nil
}

// selectProjectForSelect shows an interactive selection menu for the select command
// It writes prompts to /dev/tty so only the path goes to stdout
func selectProjectForSelect(cmd *cobra.Command, projects []*models.Project, cfg *config.Config) (*models.Project, error) {
	// Sort according to config
	sortProjects(projects, cfg.SortList)

	// Open /dev/tty for interactive output (works even when stdout is redirected)
	var tty *os.File
	var err error
	if runtime.GOOS == "windows" {
		tty, err = os.OpenFile("CON", os.O_WRONLY, 0)
	} else {
		tty, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	}
	if err != nil {
		// Fallback to stderr if /dev/tty is not available
		tty = os.Stderr
	} else {
		defer tty.Close()
		// Force color output since we're writing to a terminal
		if cfg.ShowColors && !noColor {
			color.NoColor = false
		}
	}

	// Display list to tty
	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	fmt.Fprintln(tty, "Select a project:")
	fmt.Fprintln(tty)

	// Determine grouping: flag takes precedence if explicitly set
	grouped := cfg.GroupList
	if cmd.Flags().Changed("grouped") {
		grouped = selectGrouped
	}

	// Use grouped display based on config
	opts := output.ListOptions{
		ShowPath:               false,
		ShowIndex:              true,
		Grouped:                grouped,
		ShowParentOnDuplicates: cfg.ShowParentOnDuplicates,
	}
	listOutput, indexedProjects := formatter.FormatProjectList(projects, opts)
	fmt.Fprintln(tty, listOutput)
	fmt.Fprintln(tty)

	// Read selection (prompt to tty)
	fmt.Fprint(tty, "Enter project number (or 'q' to quit): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)

	if input == "q" || input == "Q" {
		return nil, errSelectionCancelled
	}

	index, err := strconv.Atoi(input)
	if err != nil {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	// Convert 1-based input to 0-based index
	index--
	if index < 0 || index >= len(indexedProjects) {
		return nil, fmt.Errorf("invalid selection: %d", index+1)
	}

	return indexedProjects[index], nil
}
