package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/paths"
)

func TestNewStorage(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	store, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	if store.GetBasePath() != tmpDir {
		t.Errorf("expected base path %s, got %s", tmpDir, store.GetBasePath())
	}
}

func TestNewStorage_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newdir", "subdir")

	store, err := NewStorage(newDir)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	if _, err := os.Stat(store.GetBasePath()); os.IsNotExist(err) {
		t.Error("expected storage directory to be created")
	}
}

func TestNewStorage_DefaultPath(t *testing.T) {
	// Save and restore HOME
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	store, err := NewStorage("")
	if err != nil {
		t.Fatalf("NewStorage with empty path failed: %v", err)
	}

	expected := filepath.Join(tmpDir, ".projector")
	if store.GetBasePath() != expected {
		t.Errorf("expected base path %s, got %s", expected, store.GetBasePath())
	}
}

func TestStorage_GetProjectsPath(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStorage(tmpDir)

	expected := filepath.Join(tmpDir, "projects.json")
	if store.GetProjectsPath() != expected {
		t.Errorf("expected projects path %s, got %s", expected, store.GetProjectsPath())
	}
}

func TestStorage_SaveAndLoadProjects(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// Create project list
	pl := models.NewProjectList(models.KindFavorite)
	p1 := models.NewProject("project1", "/path/to/project1")
	p1.Tags = []string{"Work", "Go"}
	pl.Add(p1)

	p2 := models.NewProject("project2", "/path/to/project2")
	p2.Enabled = false
	pl.Add(p2)

	// Save
	if err := store.SaveProjects(pl); err != nil {
		t.Fatalf("SaveProjects failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(store.GetProjectsPath()); os.IsNotExist(err) {
		t.Fatal("expected projects.json to exist")
	}

	// Load
	loaded, err := store.LoadProjects()
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	if loaded.Count() != 2 {
		t.Errorf("expected 2 projects, got %d", loaded.Count())
	}

	// Verify first project
	lp1 := loaded.FindByName("project1")
	if lp1 == nil {
		t.Fatal("expected to find project1")
	}
	if lp1.RootPath != "/path/to/project1" {
		t.Errorf("expected path '/path/to/project1', got '%s'", lp1.RootPath)
	}
	if len(lp1.Tags) != 2 || lp1.Tags[0] != "Work" {
		t.Errorf("expected tags [Work, Go], got %v", lp1.Tags)
	}
	if lp1.Kind != models.KindFavorite {
		t.Errorf("expected kind KindFavorite, got %s", lp1.Kind)
	}

	// Verify second project
	lp2 := loaded.FindByName("project2")
	if lp2 == nil {
		t.Fatal("expected to find project2")
	}
	if lp2.Enabled {
		t.Error("expected project2 to be disabled")
	}
}

func TestStorage_LoadProjects_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStorage(tmpDir)

	// Load from non-existent file should return empty list
	pl, err := store.LoadProjects()
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	if pl.Count() != 0 {
		t.Errorf("expected 0 projects, got %d", pl.Count())
	}
}

func TestStorage_SaveAndLoadCache(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// Create cache
	cache := &CachedProjects{
		Git: []*models.Project{
			{Name: "git-repo1", RootPath: "/git/repo1", Enabled: true},
			{Name: "git-repo2", RootPath: "/git/repo2", Enabled: true},
		},
		SVN: []*models.Project{
			{Name: "svn-repo", RootPath: "/svn/repo", Enabled: true},
		},
		Mercurial: []*models.Project{},
		VSCode:    []*models.Project{},
		Any:       []*models.Project{},
	}

	// Save
	if err := store.SaveCache(cache); err != nil {
		t.Fatalf("SaveCache failed: %v", err)
	}

	// Load
	loaded, err := store.LoadCache()
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if len(loaded.Git) != 2 {
		t.Errorf("expected 2 git repos, got %d", len(loaded.Git))
	}
	if len(loaded.SVN) != 1 {
		t.Errorf("expected 1 svn repo, got %d", len(loaded.SVN))
	}

	// Verify kind is set
	if loaded.Git[0].Kind != models.KindGit {
		t.Errorf("expected kind KindGit, got %s", loaded.Git[0].Kind)
	}
	if loaded.SVN[0].Kind != models.KindSVN {
		t.Errorf("expected kind KindSVN, got %s", loaded.SVN[0].Kind)
	}
}

func TestStorage_LoadCache_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStorage(tmpDir)

	cache, err := store.LoadCache()
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if cache.Git != nil && len(cache.Git) > 0 {
		t.Error("expected empty git cache")
	}
}

func TestStorage_ClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStorage(tmpDir)

	// Create and save cache
	cache := &CachedProjects{
		Git: []*models.Project{
			{Name: "repo", RootPath: "/repo", Enabled: true},
		},
	}
	store.SaveCache(cache)

	cachePath := filepath.Join(tmpDir, "cache.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("expected cache file to exist")
	}

	// Clear
	if err := store.ClearCache(); err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}

	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("expected cache file to be deleted")
	}
}

func TestStorage_ClearCache_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStorage(tmpDir)

	// Should not error on non-existent cache
	if err := store.ClearCache(); err != nil {
		t.Errorf("ClearCache on non-existent file failed: %v", err)
	}
}

func TestStorage_PathExpansionOnLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStorage(tmpDir)

	home, _ := os.UserHomeDir()

	// Save with absolute path
	pl := models.NewProjectList(models.KindFavorite)
	pl.Add(models.NewProject("test", home+"/test/project"))

	if err := store.SaveProjects(pl); err != nil {
		t.Fatalf("SaveProjects failed: %v", err)
	}

	// Read raw file to verify path was collapsed
	data, _ := os.ReadFile(store.GetProjectsPath())
	if !strings.Contains(string(data), "~/test/project") {
		t.Error("expected path to be collapsed to ~ in saved file")
	}

	// Load and verify path is expanded
	loaded, _ := store.LoadProjects()
	p := loaded.FindByName("test")
	if p.RootPath != home+"/test/project" {
		t.Errorf("expected expanded path, got %s", p.RootPath)
	}
}

// Tests for paths package functions (used by storage)
func TestPaths_Expand(t *testing.T) {
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

func TestPaths_Collapse(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{home + "/projects", "~/projects"},
		{home, "~"},
		{"/other/path", "/other/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := paths.Collapse(tt.input)
			if result != tt.expected {
				t.Errorf("Collapse(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPaths_Exists(t *testing.T) {
	tmpDir := t.TempDir()

	// Existing directory
	if !paths.Exists(tmpDir) {
		t.Error("expected Exists to return true for existing dir")
	}

	// Existing file
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)
	if !paths.Exists(tmpFile) {
		t.Error("expected Exists to return true for existing file")
	}

	// Non-existent
	if paths.Exists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("expected Exists to return false for non-existent path")
	}
}
