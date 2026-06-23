// Package config provides configuration loading and management for projector,
// including default values, file-based configuration, and environment variable overrides.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"

	"github.com/ideaspaper/projector/pkg/paths"
)

const (
	configFileName = "config"
	configFileType = "json"
)

// SortOrder defines how projects are sorted
type SortOrder string

const (
	SortBySaved  SortOrder = "Saved"
	SortByName   SortOrder = "Name"
	SortByPath   SortOrder = "Path"
	SortByRecent SortOrder = "Recent"
)

// Config represents the application configuration
type Config struct {
	// Display settings
	SortList                     SortOrder `json:"sortList" mapstructure:"sortList"`
	GroupList                    bool      `json:"groupList" mapstructure:"groupList"`
	ShowColors                   bool      `json:"showColors" mapstructure:"showColors"`
	CheckInvalidPaths            bool      `json:"checkInvalidPathsBeforeListing" mapstructure:"checkInvalidPathsBeforeListing"`
	ShowParentOnDuplicates       bool      `json:"showParentFolderInfoOnDuplicates" mapstructure:"showParentFolderInfoOnDuplicates"`
	FilterOnFullPath             bool      `json:"filterOnFullPath" mapstructure:"filterOnFullPath"`
	RemoveCurrentFromList        bool      `json:"removeCurrentProjectFromList" mapstructure:"removeCurrentProjectFromList"`
	CacheProjectsBetweenSessions bool      `json:"cacheProjectsBetweenSessions" mapstructure:"cacheProjectsBetweenSessions"`
	IgnoreProjectsWithinProjects bool      `json:"ignoreProjectsWithinProjects" mapstructure:"ignoreProjectsWithinProjects"`
	SupportSymlinks              bool      `json:"supportSymlinksOnBaseFolders" mapstructure:"supportSymlinksOnBaseFolders"`

	// Editor settings
	Editor          string `json:"editor" mapstructure:"editor"`
	OpenInNewWindow bool   `json:"openInNewWindow" mapstructure:"openInNewWindow"`

	// Git settings
	GitBaseFolders    []string `json:"gitBaseFolders" mapstructure:"gitBaseFolders"`
	GitIgnoredFolders []string `json:"gitIgnoredFolders" mapstructure:"gitIgnoredFolders"`
	GitMaxDepth       int      `json:"gitMaxDepthRecursion" mapstructure:"gitMaxDepthRecursion"`

	// SVN settings
	SVNBaseFolders    []string `json:"svnBaseFolders" mapstructure:"svnBaseFolders"`
	SVNIgnoredFolders []string `json:"svnIgnoredFolders" mapstructure:"svnIgnoredFolders"`
	SVNMaxDepth       int      `json:"svnMaxDepthRecursion" mapstructure:"svnMaxDepthRecursion"`

	// Mercurial settings
	MercurialBaseFolders    []string `json:"hgBaseFolders" mapstructure:"hgBaseFolders"`
	MercurialIgnoredFolders []string `json:"hgIgnoredFolders" mapstructure:"hgIgnoredFolders"`
	MercurialMaxDepth       int      `json:"hgMaxDepthRecursion" mapstructure:"hgMaxDepthRecursion"`

	// VSCode workspace settings
	VSCodeBaseFolders    []string `json:"vscodeBaseFolders" mapstructure:"vscodeBaseFolders"`
	VSCodeIgnoredFolders []string `json:"vscodeIgnoredFolders" mapstructure:"vscodeIgnoredFolders"`
	VSCodeMaxDepth       int      `json:"vscodeMaxDepthRecursion" mapstructure:"vscodeMaxDepthRecursion"`

	// Any folder settings
	AnyBaseFolders    []string `json:"anyBaseFolders" mapstructure:"anyBaseFolders"`
	AnyIgnoredFolders []string `json:"anyIgnoredFolders" mapstructure:"anyIgnoredFolders"`
	AnyMaxDepth       int      `json:"anyMaxDepthRecursion" mapstructure:"anyMaxDepthRecursion"`

	// Custom projects location
	ProjectsLocation string `json:"projectsLocation" mapstructure:"projectsLocation"`

	// Internal
	v          *viper.Viper `json:"-" mapstructure:"-"`
	configPath string       `json:"-" mapstructure:"-"`
}

// DefaultConfig returns a new config with default values
func DefaultConfig() *Config {
	return &Config{
		SortList:                     SortByName,
		GroupList:                    true,
		ShowColors:                   true,
		CheckInvalidPaths:            true,
		ShowParentOnDuplicates:       false,
		FilterOnFullPath:             false,
		RemoveCurrentFromList:        true,
		CacheProjectsBetweenSessions: true,
		IgnoreProjectsWithinProjects: false,
		SupportSymlinks:              false,

		Editor:          detectDefaultEditor(),
		OpenInNewWindow: false,

		GitBaseFolders:    []string{},
		GitIgnoredFolders: []string{"node_modules", "out", "typings", "test", ".haxelib", "vendor"},
		GitMaxDepth:       4,

		SVNBaseFolders:    []string{},
		SVNIgnoredFolders: []string{"node_modules", "out", "typings", "test"},
		SVNMaxDepth:       4,

		MercurialBaseFolders:    []string{},
		MercurialIgnoredFolders: []string{"node_modules", "out", "typings", "test", ".haxelib"},
		MercurialMaxDepth:       4,

		VSCodeBaseFolders:    []string{},
		VSCodeIgnoredFolders: []string{"node_modules", "out", "typings", "test"},
		VSCodeMaxDepth:       4,

		AnyBaseFolders:    []string{},
		AnyIgnoredFolders: []string{"node_modules", "out", "typings", "test"},
		AnyMaxDepth:       4,

		ProjectsLocation: "",
	}
}

// detectDefaultEditor detects the default editor based on environment
func detectDefaultEditor() string {
	// Check EDITOR environment variable first
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Check for VS Code using exec.LookPath (more portable)
	if _, err := exec.LookPath("code"); err == nil {
		return "code"
	}

	// Fallback options
	switch runtime.GOOS {
	case "darwin":
		return "open"
	case "windows":
		return "code"
	default:
		return "xdg-open"
	}
}

// setDefaults sets default values in viper
func setDefaults(v *viper.Viper) {
	cfg := DefaultConfig()

	v.SetDefault("sortList", cfg.SortList)
	v.SetDefault("groupList", cfg.GroupList)
	v.SetDefault("showColors", cfg.ShowColors)
	v.SetDefault("checkInvalidPathsBeforeListing", cfg.CheckInvalidPaths)
	v.SetDefault("showParentFolderInfoOnDuplicates", cfg.ShowParentOnDuplicates)
	v.SetDefault("filterOnFullPath", cfg.FilterOnFullPath)
	v.SetDefault("removeCurrentProjectFromList", cfg.RemoveCurrentFromList)
	v.SetDefault("cacheProjectsBetweenSessions", cfg.CacheProjectsBetweenSessions)
	v.SetDefault("ignoreProjectsWithinProjects", cfg.IgnoreProjectsWithinProjects)
	v.SetDefault("supportSymlinksOnBaseFolders", cfg.SupportSymlinks)

	v.SetDefault("editor", cfg.Editor)
	v.SetDefault("openInNewWindow", cfg.OpenInNewWindow)

	v.SetDefault("gitBaseFolders", cfg.GitBaseFolders)
	v.SetDefault("gitIgnoredFolders", cfg.GitIgnoredFolders)
	v.SetDefault("gitMaxDepthRecursion", cfg.GitMaxDepth)

	v.SetDefault("svnBaseFolders", cfg.SVNBaseFolders)
	v.SetDefault("svnIgnoredFolders", cfg.SVNIgnoredFolders)
	v.SetDefault("svnMaxDepthRecursion", cfg.SVNMaxDepth)

	v.SetDefault("hgBaseFolders", cfg.MercurialBaseFolders)
	v.SetDefault("hgIgnoredFolders", cfg.MercurialIgnoredFolders)
	v.SetDefault("hgMaxDepthRecursion", cfg.MercurialMaxDepth)

	v.SetDefault("vscodeBaseFolders", cfg.VSCodeBaseFolders)
	v.SetDefault("vscodeIgnoredFolders", cfg.VSCodeIgnoredFolders)
	v.SetDefault("vscodeMaxDepthRecursion", cfg.VSCodeMaxDepth)

	v.SetDefault("anyBaseFolders", cfg.AnyBaseFolders)
	v.SetDefault("anyIgnoredFolders", cfg.AnyIgnoredFolders)
	v.SetDefault("anyMaxDepthRecursion", cfg.AnyMaxDepth)

	v.SetDefault("projectsLocation", cfg.ProjectsLocation)
}

// LoadConfig loads configuration from the default path
func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".projector")
	return LoadConfigFromDir(configDir)
}

// LoadConfigFromDir loads configuration from a specific directory
func LoadConfigFromDir(dir string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure Viper
	v.SetConfigName(configFileName)
	v.SetConfigType(configFileType)
	v.AddConfigPath(dir)

	// Allow environment variable overrides with prefix PROJECTOR_
	v.SetEnvPrefix("PROJECTOR")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	configPath := filepath.Join(dir, configFileName+"."+configFileType)

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults
			cfg := DefaultConfig()
			cfg.v = v
			cfg.configPath = configPath
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into Config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.v = v
	cfg.configPath = configPath

	return cfg, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	if c.configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		c.configPath = filepath.Join(homeDir, ".projector", configFileName+"."+configFileType)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProjectsLocation returns the effective projects location
func (c *Config) GetProjectsLocation() string {
	if c.ProjectsLocation != "" {
		return paths.Expand(c.ProjectsLocation)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".projector")
}

// LoadOrCreateConfig loads existing config or creates a new one with defaults.
// If the config file cannot be read (other than not existing), a warning is printed
// to stderr and default config is returned.
func LoadOrCreateConfig() (*Config, error) {
	cfg, err := LoadConfig()
	if err != nil {
		// Log warning to stderr so users are aware of config issues
		fmt.Fprintf(os.Stderr, "Warning: failed to load config, using defaults: %v\n", err)
		return DefaultConfig(), nil
	}
	return cfg, nil
}
