package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ideaspaper/projector/pkg/config"
	"github.com/ideaspaper/projector/pkg/output"
	"github.com/ideaspaper/projector/pkg/storage"
)

// clearCacheCmd represents the clear-cache command
var clearCacheCmd = &cobra.Command{
	Use:   "clear-cache",
	Short: "Clear the cached auto-detected projects",
	Long: `Remove the cached auto-detected projects (Git, SVN, Mercurial, VS Code, Any).

This does not affect your saved favorites in projects.json.
After clearing the cache, run 'projector scan' to re-detect projects.`,
	Aliases: []string{"cc"},
	RunE:    runClearCache,
}

func init() {
	rootCmd.AddCommand(clearCacheCmd)
}

func runClearCache(cmd *cobra.Command, args []string) error {
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

	// Clear the cache
	if err := store.ClearCache(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Output
	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	fmt.Println(formatter.FormatSuccess("Cache cleared successfully"))

	return nil
}
