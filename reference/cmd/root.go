package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// version is set at build time via ldflags
	version = "dev"

	// Global flags
	noColor bool
	verbose bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "projector",
	Short: "A CLI project manager inspired by VS Code Project Manager",
	Long: `Projector is a command-line project manager that helps you easily access
your projects, no matter where they are located.

You can define your own Projects (Favorites), or auto-detect Git, Mercurial,
SVN repositories, VS Code workspaces, or any folder.

Examples:
  # Add current directory as a project
  projector add

  # Add a specific directory with a custom name
  projector add ~/projects/myapp --name "My App"

  # List all projects
  projector list

  # Open a project in your editor
  projector open myproject

  # Scan for Git repositories
  projector scan --git ~/projects

  # Filter projects by tag
  projector list --tag Work`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
