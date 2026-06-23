package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideaspaper/projector/pkg/config"
	"github.com/ideaspaper/projector/pkg/output"
	"github.com/ideaspaper/projector/pkg/storage"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:     "remove <project-name>",
	Short:   "Remove a project from favorites",
	Long:    `Remove a project from your saved favorites by name.`,
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ExactArgs(1),
	RunE:    runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	projectName := args[0]

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

	// Load projects
	projects, err := store.LoadProjects()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	// Find and remove project
	if !projects.Remove(projectName) {
		return fmt.Errorf("project '%s' not found", projectName)
	}

	// Save
	if err := store.SaveProjects(projects); err != nil {
		return fmt.Errorf("failed to save projects: %w", err)
	}

	// Output
	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	fmt.Println(formatter.FormatSuccess(fmt.Sprintf("Removed project '%s'", projectName)))

	return nil
}

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit <project-name>",
	Short: "Edit a project's properties",
	Long: `Edit a project's name, path, tags, or enabled state.

Examples:
  # Rename a project
  projector edit myproject --name "New Name"

  # Update path
  projector edit myproject --path ~/new/path

  # Toggle enabled state
  projector edit myproject --enabled=false

  # Add tags
  projector edit myproject --add-tag Work --add-tag Important

  # Remove a tag
  projector edit myproject --remove-tag Old`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

var (
	editName       string
	editPath       string
	editEnabled    string
	editAddTags    []string
	editRemoveTags []string
)

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVar(&editName, "name", "", "new project name")
	editCmd.Flags().StringVar(&editPath, "path", "", "new project path")
	editCmd.Flags().StringVar(&editEnabled, "enabled", "", "enable/disable project (true/false)")
	editCmd.Flags().StringSliceVar(&editAddTags, "add-tag", []string{}, "add a tag to the project (can be used multiple times)")
	editCmd.Flags().StringSliceVar(&editRemoveTags, "remove-tag", []string{}, "remove a tag from the project (can be used multiple times)")
}

func runEdit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

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

	// Load projects
	projects, err := store.LoadProjects()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	// Find project
	project := projects.FindByName(projectName)
	if project == nil {
		return fmt.Errorf("project '%s' not found", projectName)
	}

	// Apply changes
	changed := false

	if editName != "" {
		// Check for name conflict
		if existing := projects.FindByName(editName); existing != nil && existing != project {
			return fmt.Errorf("project with name '%s' already exists", editName)
		}
		project.Name = editName
		changed = true
	}

	if editPath != "" {
		// Resolve to absolute path
		absPath, err := filepath.Abs(editPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
		// Check if path exists
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("path does not exist: %s", absPath)
		}
		if !info.IsDir() {
			return fmt.Errorf("path is not a directory: %s", absPath)
		}
		project.RootPath = absPath
		changed = true
	}

	if editEnabled != "" {
		enabled, err := strconv.ParseBool(editEnabled)
		if err != nil {
			return fmt.Errorf("--enabled must be a boolean value (true, false, 1, 0, etc.): %w", err)
		}
		project.Enabled = enabled
		changed = true
	}

	// Add tags
	for _, tag := range editAddTags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if project.HasTag(tag) {
			return fmt.Errorf("project already has tag '%s'", tag)
		}
		project.AddTag(tag)
		changed = true
	}

	// Remove tags
	for _, tag := range editRemoveTags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !project.HasTag(tag) {
			return fmt.Errorf("project does not have tag '%s'", tag)
		}
		project.RemoveTag(tag)
		changed = true
	}

	if !changed {
		return fmt.Errorf("no changes specified (use --name, --path, --enabled, --add-tag, or --remove-tag)")
	}

	// Save
	if err := store.SaveProjects(projects); err != nil {
		return fmt.Errorf("failed to save projects: %w", err)
	}

	// Output
	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	fmt.Println(formatter.FormatSuccess(fmt.Sprintf("Updated project '%s'", project.Name)))

	return nil
}

// tagsCmd represents the tags command
var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags in use",
	Long:  `List all unique tags currently used by projects.`,
	RunE:  runTags,
}

func init() {
	rootCmd.AddCommand(tagsCmd)
}

func runTags(cmd *cobra.Command, args []string) error {
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

	// Load projects
	projects, err := store.LoadProjects()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	// Collect unique tags from all projects
	tagSet := make(map[string]struct{})
	for _, p := range projects.Projects {
		for _, tag := range p.Tags {
			tagSet[tag] = struct{}{}
		}
	}

	formatter := output.NewFormatter(!noColor && cfg.ShowColors)

	if len(tagSet) == 0 {
		fmt.Println(formatter.FormatInfo("No tags in use"))
		return nil
	}

	// Convert to sorted slice
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	fmt.Println("Tags in use:")
	for _, tag := range tags {
		fmt.Printf("  - %s\n", tag)
	}

	return nil
}
