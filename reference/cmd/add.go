package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideaspaper/projector/pkg/config"
	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/output"
	"github.com/ideaspaper/projector/pkg/storage"
)

var (
	// add command flags
	addName    string
	addTags    []string
	addEnabled bool
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Add a project to your favorites",
	Long: `Add a folder as a project to your favorites.

If no path is provided, the current directory is used.

Examples:
  # Add current directory as a project
  projector add

  # Add a specific directory
  projector add ~/projects/myapp

  # Add with a custom name
  projector add ~/projects/myapp --name "My Application"

  # Add with tags
  projector add --name "Work Project" --tag Work --tag Important`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addName, "name", "n", "", "project name (defaults to folder name)")
	addCmd.Flags().StringSliceVarP(&addTags, "tag", "t", []string{}, "tags for the project (can be used multiple times)")
	addCmd.Flags().BoolVar(&addEnabled, "enabled", true, "whether the project is enabled")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.LoadOrCreateConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine the path
	var projectPath string
	if len(args) > 0 {
		projectPath = args[0]
	} else {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Resolve to absolute path
	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(projectPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", projectPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", projectPath)
	}

	// Determine project name
	name := addName
	if name == "" {
		name = filepath.Base(projectPath)
	}

	// Load existing projects
	store, err := storage.NewStorage(cfg.GetProjectsLocation())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	projects, err := store.LoadProjects()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	// Check if project already exists
	for _, p := range projects.Projects {
		if p.RootPath == projectPath {
			return fmt.Errorf("project already exists: %s", p.Name)
		}
		if strings.EqualFold(p.Name, name) {
			return fmt.Errorf("project with name '%s' already exists", name)
		}
	}

	// Create new project
	project := &models.Project{
		Name:     name,
		RootPath: projectPath,
		Tags:     addTags,
		Enabled:  addEnabled,
		Kind:     models.KindFavorite,
	}

	// Add to list
	projects.Add(project)

	// Save
	if err := store.SaveProjects(projects); err != nil {
		return fmt.Errorf("failed to save projects: %w", err)
	}

	// Output
	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	fmt.Println(formatter.FormatSuccess(fmt.Sprintf("Added project '%s' at %s", name, projectPath)))

	return nil
}
