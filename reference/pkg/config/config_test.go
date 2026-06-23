package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ideaspaper/projector/pkg/paths"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.SortList != SortByName {
		t.Errorf("expected SortByName, got %s", cfg.SortList)
	}
	if cfg.GroupList != true {
		t.Error("expected GroupList to be true")
	}
	if cfg.ShowColors != true {
		t.Error("expected ShowColors to be true")
	}
	if cfg.CheckInvalidPaths != true {
		t.Error("expected CheckInvalidPaths to be true")
	}
	if cfg.OpenInNewWindow != false {
		t.Error("expected OpenInNewWindow to be false")
	}
	if cfg.GitMaxDepth != 4 {
		t.Errorf("expected GitMaxDepth 4, got %d", cfg.GitMaxDepth)
	}
	if len(cfg.GitIgnoredFolders) == 0 {
		t.Error("expected default GitIgnoredFolders")
	}
}

func TestSortOrder_Values(t *testing.T) {
	tests := []struct {
		order    SortOrder
		expected string
	}{
		{SortBySaved, "Saved"},
		{SortByName, "Name"},
		{SortByPath, "Path"},
		{SortByRecent, "Recent"},
	}

	for _, tt := range tests {
		if string(tt.order) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.order)
		}
	}
}

func TestLoadConfigFromDir_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")

	// Should return defaults when config doesn't exist
	cfg, err := LoadConfigFromDir(configDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir failed: %v", err)
	}

	// Should have default values
	if cfg.SortList != SortByName {
		t.Errorf("expected default SortByName, got %s", cfg.SortList)
	}
}

func TestLoadConfigFromDir_ExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir, 0755)

	// Write a config file
	configContent := `{
		"sortList": "Path",
		"groupList": true,
		"showColors": false,
		"editor": "vim",
		"gitMaxDepthRecursion": 10
	}`
	os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configContent), 0644)

	cfg, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir failed: %v", err)
	}

	if cfg.SortList != SortByPath {
		t.Errorf("expected SortByPath, got %s", cfg.SortList)
	}
	if cfg.GroupList != true {
		t.Error("expected GroupList to be true")
	}
	if cfg.ShowColors != false {
		t.Error("expected ShowColors to be false")
	}
	if cfg.Editor != "vim" {
		t.Errorf("expected editor 'vim', got '%s'", cfg.Editor)
	}
	if cfg.GitMaxDepth != 10 {
		t.Errorf("expected GitMaxDepth 10, got %d", cfg.GitMaxDepth)
	}
}

func TestConfig_Save(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.configPath = configPath
	cfg.Editor = "cursor"

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("expected config file to exist")
	}

	// Load and verify
	loaded, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir failed: %v", err)
	}

	if loaded.Editor != "cursor" {
		t.Errorf("expected editor 'cursor', got '%s'", loaded.Editor)
	}
}

func TestConfig_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "newdir", "subdir")
	configPath := filepath.Join(configDir, "config.json")

	cfg := DefaultConfig()
	cfg.configPath = configPath

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("expected config file to be created")
	}
}

func TestConfig_GetProjectsLocation(t *testing.T) {
	home, _ := os.UserHomeDir()

	// Default location
	cfg := DefaultConfig()
	defaultLoc := cfg.GetProjectsLocation()
	expected := filepath.Join(home, ".projector")
	if defaultLoc != expected {
		t.Errorf("expected default location %s, got %s", expected, defaultLoc)
	}

	// Custom location
	cfg.ProjectsLocation = "~/custom/projects"
	customLoc := cfg.GetProjectsLocation()
	expectedCustom := home + "/custom/projects"
	if customLoc != expectedCustom {
		t.Errorf("expected custom location %s, got %s", expectedCustom, customLoc)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/projects", home + "/projects"},
		{"$home/projects", home + "/projects"},
		{"$HOME/projects", home + "/projects"},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", home},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := paths.Expand(tt.input)
			if result != tt.expected {
				t.Errorf("Expand(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadOrCreateConfig(t *testing.T) {
	// Should not fail even when config doesn't exist
	cfg, err := LoadOrCreateConfig()
	if err != nil {
		t.Fatalf("LoadOrCreateConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config to not be nil")
	}
}

func TestConfig_EnvironmentOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	// Set environment variable
	os.Setenv("PROJECTOR_EDITOR", "nvim")
	defer os.Unsetenv("PROJECTOR_EDITOR")

	cfg, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir failed: %v", err)
	}

	if cfg.Editor != "nvim" {
		t.Errorf("expected editor 'nvim' from env, got '%s'", cfg.Editor)
	}
}

func TestDetectDefaultEditor(t *testing.T) {
	// Save original EDITOR
	origEditor := os.Getenv("EDITOR")
	defer os.Setenv("EDITOR", origEditor)

	// Test with EDITOR set
	os.Setenv("EDITOR", "nano")
	editor := detectDefaultEditor()
	if editor != "nano" {
		t.Errorf("expected 'nano' from EDITOR env, got '%s'", editor)
	}

	// Test without EDITOR
	os.Unsetenv("EDITOR")
	editor = detectDefaultEditor()
	// Should return something (platform-dependent)
	if editor == "" {
		t.Error("expected a default editor")
	}
}

func TestConfig_AllFieldsSerialized(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.configPath = configPath

	// Set various fields
	cfg.SortList = SortByPath
	cfg.GroupList = true
	cfg.ShowColors = false
	cfg.Editor = "vim"
	cfg.OpenInNewWindow = true
	cfg.GitBaseFolders = []string{"~/git"}
	cfg.GitMaxDepth = 8
	cfg.SVNBaseFolders = []string{"~/svn"}
	cfg.MercurialBaseFolders = []string{"~/hg"}
	cfg.VSCodeBaseFolders = []string{"~/vscode"}
	cfg.AnyBaseFolders = []string{"~/any"}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load and verify all fields
	loaded, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir failed: %v", err)
	}

	if loaded.SortList != SortByPath {
		t.Errorf("SortList: expected Path, got %s", loaded.SortList)
	}
	if loaded.GroupList != true {
		t.Error("GroupList: expected true")
	}
	if loaded.ShowColors != false {
		t.Error("ShowColors: expected false")
	}
	if loaded.Editor != "vim" {
		t.Errorf("Editor: expected vim, got %s", loaded.Editor)
	}
	if loaded.OpenInNewWindow != true {
		t.Error("OpenInNewWindow: expected true")
	}
	if loaded.GitMaxDepth != 8 {
		t.Errorf("GitMaxDepth: expected 8, got %d", loaded.GitMaxDepth)
	}
	if len(loaded.GitBaseFolders) != 1 {
		t.Errorf("GitBaseFolders: expected 1, got %d", len(loaded.GitBaseFolders))
	}
}

func TestConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid JSON
	os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{invalid json}"), 0644)

	_, err := LoadConfigFromDir(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
