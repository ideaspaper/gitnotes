package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideaspaper/projector/pkg/config"
	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/output"
	"github.com/ideaspaper/projector/pkg/scanner"
	"github.com/ideaspaper/projector/pkg/storage"
)

var (
	// list command flags
	listTag       string
	listShowPath  bool
	listGrouped   bool
	listAll       bool
	listFavorites bool
	listGit       bool
	listSVN       bool
	listMercurial bool
	listVSCode    bool
	listAny       bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved and detected projects",
	Long: `List all projects, including saved favorites and auto-detected repositories.

Examples:
  # List all projects
  projector list

  # List only favorites
  projector list --favorites

  # List only git repositories
  projector list --git

  # Filter by tag
  projector list --tag Work

  # Show project paths
  projector list --path

  # Group by project type
  projector list --grouped`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listTag, "tag", "t", "", "filter projects by tag")
	listCmd.Flags().BoolVarP(&listShowPath, "path", "p", false, "show project paths")
	listCmd.Flags().BoolVarP(&listGrouped, "grouped", "g", false, "group projects by type")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "include disabled projects")
	listCmd.Flags().BoolVar(&listFavorites, "favorites", false, "show only favorites")
	listCmd.Flags().BoolVar(&listGit, "git", false, "show only git repositories")
	listCmd.Flags().BoolVar(&listSVN, "svn", false, "show only svn repositories")
	listCmd.Flags().BoolVar(&listMercurial, "mercurial", false, "show only mercurial repositories")
	listCmd.Flags().BoolVar(&listVSCode, "vscode", false, "show only vscode workspaces")
	listCmd.Flags().BoolVar(&listAny, "any", false, "show only any-folder projects")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.LoadOrCreateConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logVerbose(cfg, "Loading projects with filters: favorites=%v git=%v svn=%v mercurial=%v vscode=%v any=%v",
		listFavorites, listGit, listSVN, listMercurial, listVSCode, listAny)

	// Initialize storage
	store, err := storage.NewStorage(cfg.GetProjectsLocation())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Load projects with type filter
	filter := TypeFilter{
		Favorites: listFavorites,
		Git:       listGit,
		SVN:       listSVN,
		Mercurial: listMercurial,
		VSCode:    listVSCode,
		Any:       listAny,
	}
	allProjects, err := LoadFilteredProjects(store, filter)
	if err != nil {
		return err
	}

	logVerbose(cfg, "Loaded %d projects before filtering", len(allProjects))

	if cfg.CheckInvalidPaths {
		MarkInvalidProjectsDisabled(allProjects)
	}

	if cfg.RemoveCurrentFromList {
		allProjects = RemoveCurrentProject(allProjects)
	}

	// Filter by enabled
	if !listAll {
		allProjects = FilterEnabled(allProjects)
	}

	// Filter by tag
	allProjects = FilterByTag(allProjects, listTag)

	logVerbose(cfg, "After filtering: %d projects", len(allProjects))

	// Sort projects
	sortProjects(allProjects, cfg.SortList)

	// Override grouping from flag or config
	// Flag takes precedence if explicitly set
	grouped := cfg.GroupList
	if cmd.Flags().Changed("grouped") {
		grouped = listGrouped
	}

	// Format and display
	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	opts := output.ListOptions{
		ShowPath:               listShowPath,
		ShowIndex:              false,
		Grouped:                grouped,
		ShowParentOnDuplicates: cfg.ShowParentOnDuplicates,
	}
	listOutput, _ := formatter.FormatProjectList(allProjects, opts)
	fmt.Println(listOutput)

	return nil
}

// sortProjects sorts projects according to the specified order
func sortProjects(projects []*models.Project, order config.SortOrder) {
	switch order {
	case config.SortByName:
		sort.Slice(projects, func(i, j int) bool {
			return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
		})
	case config.SortByPath:
		sort.Slice(projects, func(i, j int) bool {
			return strings.ToLower(projects[i].RootPath) < strings.ToLower(projects[j].RootPath)
		})
	case config.SortBySaved, config.SortByRecent:
		// Keep original order for saved/recent
	}
}

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan [paths...]",
	Short: "Scan directories for projects",
	Long: `Scan directories for Git, SVN, Mercurial repositories, VS Code workspaces,
or any folder.

Examples:
  # Scan for git repositories in ~/projects
  projector scan --git ~/projects

  # Scan for all types in configured base folders
  projector scan --all

  # Scan for git repos with custom depth
  projector scan --git --depth 5 ~/code`,
	RunE: runScan,
}

var (
	scanGit       bool
	scanSVN       bool
	scanMercurial bool
	scanVSCode    bool
	scanAny       bool
	scanAll       bool
	scanDepth     int
)

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().BoolVar(&scanGit, "git", false, "scan for git repositories")
	scanCmd.Flags().BoolVar(&scanSVN, "svn", false, "scan for svn repositories")
	scanCmd.Flags().BoolVar(&scanMercurial, "mercurial", false, "scan for mercurial repositories")
	scanCmd.Flags().BoolVar(&scanVSCode, "vscode", false, "scan for vscode workspaces")
	scanCmd.Flags().BoolVar(&scanAny, "any", false, "scan for any folder")
	scanCmd.Flags().BoolVarP(&scanAll, "all", "a", false, "scan for all types")
	scanCmd.Flags().IntVarP(&scanDepth, "depth", "d", 0, "maximum scan depth (0 = use config default)")
}

func runScan(cmd *cobra.Command, args []string) error {
	// Validate depth flag
	if scanDepth < 0 {
		return fmt.Errorf("--depth must be a non-negative integer, got %d", scanDepth)
	}

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

	// Determine what to scan
	if !scanGit && !scanSVN && !scanMercurial && !scanVSCode && !scanAny && !scanAll {
		scanAll = true
	}

	formatter := output.NewFormatter(!noColor && cfg.ShowColors)
	cache := &storage.CachedProjects{}

	// Scan Git
	if scanAll || scanGit {
		baseFolders := cfg.GitBaseFolders
		if len(args) > 0 {
			baseFolders = args
		}
		if len(baseFolders) > 0 {
			s := scanner.NewScanner(scanner.ScannerGit)
			s.SetBaseFolders(baseFolders)
			s.SetIgnoredFolders(cfg.GitIgnoredFolders)
			depth := cfg.GitMaxDepth
			if scanDepth > 0 {
				depth = scanDepth
			}
			s.SetMaxDepth(depth)
			s.SetIgnoreWithinProjects(cfg.IgnoreProjectsWithinProjects)
			s.SetSupportSymlinks(cfg.SupportSymlinks)

			projects, err := s.Scan()
			if err != nil {
				fmt.Println(formatter.FormatWarning(fmt.Sprintf("Error scanning Git repositories: %v", err)))
			} else {
				cache.Git = projects
				fmt.Println(formatter.FormatInfo(fmt.Sprintf("Found %d Git repositories", len(projects))))
			}
		}
	}

	// Scan SVN
	if scanAll || scanSVN {
		baseFolders := cfg.SVNBaseFolders
		if len(args) > 0 {
			baseFolders = args
		}
		if len(baseFolders) > 0 {
			s := scanner.NewScanner(scanner.ScannerSVN)
			s.SetBaseFolders(baseFolders)
			s.SetIgnoredFolders(cfg.SVNIgnoredFolders)
			depth := cfg.SVNMaxDepth
			if scanDepth > 0 {
				depth = scanDepth
			}
			s.SetMaxDepth(depth)

			projects, err := s.Scan()
			if err != nil {
				fmt.Println(formatter.FormatWarning(fmt.Sprintf("Error scanning SVN repositories: %v", err)))
			} else {
				cache.SVN = projects
				fmt.Println(formatter.FormatInfo(fmt.Sprintf("Found %d SVN repositories", len(projects))))
			}
		}
	}

	// Scan Mercurial
	if scanAll || scanMercurial {
		baseFolders := cfg.MercurialBaseFolders
		if len(args) > 0 {
			baseFolders = args
		}
		if len(baseFolders) > 0 {
			s := scanner.NewScanner(scanner.ScannerMercurial)
			s.SetBaseFolders(baseFolders)
			s.SetIgnoredFolders(cfg.MercurialIgnoredFolders)
			depth := cfg.MercurialMaxDepth
			if scanDepth > 0 {
				depth = scanDepth
			}
			s.SetMaxDepth(depth)

			projects, err := s.Scan()
			if err != nil {
				fmt.Println(formatter.FormatWarning(fmt.Sprintf("Error scanning Mercurial repositories: %v", err)))
			} else {
				cache.Mercurial = projects
				fmt.Println(formatter.FormatInfo(fmt.Sprintf("Found %d Mercurial repositories", len(projects))))
			}
		}
	}

	// Scan VSCode
	if scanAll || scanVSCode {
		baseFolders := cfg.VSCodeBaseFolders
		if len(args) > 0 {
			baseFolders = args
		}
		if len(baseFolders) > 0 {
			s := scanner.NewScanner(scanner.ScannerVSCode)
			s.SetBaseFolders(baseFolders)
			s.SetIgnoredFolders(cfg.VSCodeIgnoredFolders)
			depth := cfg.VSCodeMaxDepth
			if scanDepth > 0 {
				depth = scanDepth
			}
			s.SetMaxDepth(depth)

			projects, err := s.Scan()
			if err != nil {
				fmt.Println(formatter.FormatWarning(fmt.Sprintf("Error scanning VS Code workspaces: %v", err)))
			} else {
				cache.VSCode = projects
				fmt.Println(formatter.FormatInfo(fmt.Sprintf("Found %d VS Code workspaces", len(projects))))
			}
		}
	}

	// Scan Any
	if scanAll || scanAny {
		baseFolders := cfg.AnyBaseFolders
		if len(args) > 0 {
			baseFolders = args
		}
		if len(baseFolders) > 0 {
			s := scanner.NewScanner(scanner.ScannerAny)
			s.SetBaseFolders(baseFolders)
			s.SetIgnoredFolders(cfg.AnyIgnoredFolders)
			depth := cfg.AnyMaxDepth
			if scanDepth > 0 {
				depth = scanDepth
			}
			s.SetMaxDepth(depth)

			projects, err := s.Scan()
			if err != nil {
				fmt.Println(formatter.FormatWarning(fmt.Sprintf("Error scanning folders: %v", err)))
			} else {
				cache.Any = projects
				fmt.Println(formatter.FormatInfo(fmt.Sprintf("Found %d folders", len(projects))))
			}
		}
	}

	// Save cache
	if cfg.CacheProjectsBetweenSessions {
		if err := store.SaveCache(cache); err != nil {
			return fmt.Errorf("failed to save cache: %w", err)
		}
		fmt.Println(formatter.FormatSuccess("Cache updated"))
	}

	return nil
}
