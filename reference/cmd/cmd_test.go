package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/storage"
)

// testSetup creates a temporary directory with test data and returns cleanup function
func testSetup(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "projector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create test projects.json (favorites)
	favorites := []*models.Project{
		{Name: "favorite1", RootPath: "/path/to/favorite1", Enabled: true, Tags: []string{"Work"}},
		{Name: "favorite2", RootPath: "/path/to/favorite2", Enabled: true, Tags: []string{"Personal"}},
	}
	favoritesData, _ := json.MarshalIndent(favorites, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "projects.json"), favoritesData, 0644)

	// Create test cache.json
	cache := storage.CachedProjects{
		Git: []*models.Project{
			{Name: "git-repo1", RootPath: "/path/to/git1", Enabled: true},
			{Name: "git-repo2", RootPath: "/path/to/git2", Enabled: true},
		},
		SVN: []*models.Project{
			{Name: "svn-repo1", RootPath: "/path/to/svn1", Enabled: true},
		},
		Mercurial: []*models.Project{
			{Name: "hg-repo1", RootPath: "/path/to/hg1", Enabled: true},
		},
		VSCode: []*models.Project{
			{Name: "vscode-ws1", RootPath: "/path/to/vscode1", Enabled: true},
		},
		Any: []*models.Project{
			{Name: "any-folder1", RootPath: "/path/to/any1", Enabled: true},
		},
	}
	cacheData, _ := json.MarshalIndent(cache, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "cache.json"), cacheData, 0644)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestLoadProjectsWithTypeFilters(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	store, err := storage.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	tests := []struct {
		name          string
		showFavorites bool
		showGit       bool
		showSVN       bool
		showMercurial bool
		showVSCode    bool
		showAny       bool
		wantCount     int
		wantNames     []string
	}{
		{
			name:      "show all (no filter)",
			wantCount: 8,
		},
		{
			name:          "show only favorites",
			showFavorites: true,
			wantCount:     2,
			wantNames:     []string{"favorite1", "favorite2"},
		},
		{
			name:      "show only git",
			showGit:   true,
			wantCount: 2,
			wantNames: []string{"git-repo1", "git-repo2"},
		},
		{
			name:      "show only svn",
			showSVN:   true,
			wantCount: 1,
			wantNames: []string{"svn-repo1"},
		},
		{
			name:          "show only mercurial",
			showMercurial: true,
			wantCount:     1,
			wantNames:     []string{"hg-repo1"},
		},
		{
			name:       "show only vscode",
			showVSCode: true,
			wantCount:  1,
			wantNames:  []string{"vscode-ws1"},
		},
		{
			name:      "show only any",
			showAny:   true,
			wantCount: 1,
			wantNames: []string{"any-folder1"},
		},
		{
			name:          "show favorites and git",
			showFavorites: true,
			showGit:       true,
			wantCount:     4,
			wantNames:     []string{"favorite1", "favorite2", "git-repo1", "git-repo2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var allProjects []*models.Project

			// Determine which types to show (same logic as in open.go and select.go)
			showAll := !tt.showFavorites && !tt.showGit && !tt.showSVN && !tt.showMercurial && !tt.showVSCode && !tt.showAny

			// Load favorites
			if showAll || tt.showFavorites {
				projects, err := store.LoadProjects()
				if err != nil {
					t.Fatalf("failed to load projects: %v", err)
				}
				allProjects = append(allProjects, projects.Projects...)
			}

			// Load cached auto-detected projects
			if showAll || tt.showGit || tt.showSVN || tt.showMercurial || tt.showVSCode || tt.showAny {
				cache, err := store.LoadCache()
				if err != nil {
					t.Fatalf("failed to load cache: %v", err)
				}
				if showAll || tt.showGit {
					allProjects = append(allProjects, cache.Git...)
				}
				if showAll || tt.showSVN {
					allProjects = append(allProjects, cache.SVN...)
				}
				if showAll || tt.showMercurial {
					allProjects = append(allProjects, cache.Mercurial...)
				}
				if showAll || tt.showVSCode {
					allProjects = append(allProjects, cache.VSCode...)
				}
				if showAll || tt.showAny {
					allProjects = append(allProjects, cache.Any...)
				}
			}

			if len(allProjects) != tt.wantCount {
				t.Errorf("got %d projects, want %d", len(allProjects), tt.wantCount)
			}

			if tt.wantNames != nil {
				gotNames := make(map[string]bool)
				for _, p := range allProjects {
					gotNames[p.Name] = true
				}
				for _, name := range tt.wantNames {
					if !gotNames[name] {
						t.Errorf("expected project %q not found", name)
					}
				}
			}
		})
	}
}

func TestLoadProjectsWithTagFilter(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	store, err := storage.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Load all projects
	allProjects, err := store.LoadAllProjects()
	if err != nil {
		t.Fatalf("failed to load all projects: %v", err)
	}

	tests := []struct {
		name      string
		tag       string
		wantCount int
		wantNames []string
	}{
		{
			name:      "filter by Work tag",
			tag:       "Work",
			wantCount: 1,
			wantNames: []string{"favorite1"},
		},
		{
			name:      "filter by Personal tag",
			tag:       "Personal",
			wantCount: 1,
			wantNames: []string{"favorite2"},
		},
		{
			name:      "filter by non-existent tag",
			tag:       "NonExistent",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := make([]*models.Project, 0)
			for _, p := range allProjects {
				if p.HasTag(tt.tag) {
					filtered = append(filtered, p)
				}
			}

			if len(filtered) != tt.wantCount {
				t.Errorf("got %d projects, want %d", len(filtered), tt.wantCount)
			}

			if tt.wantNames != nil {
				gotNames := make(map[string]bool)
				for _, p := range filtered {
					gotNames[p.Name] = true
				}
				for _, name := range tt.wantNames {
					if !gotNames[name] {
						t.Errorf("expected project %q not found", name)
					}
				}
			}
		})
	}
}

func TestLoadProjectsCombinedFilters(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	store, err := storage.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Test: show only favorites with Work tag
	t.Run("favorites with Work tag", func(t *testing.T) {
		// Load only favorites
		projects, err := store.LoadProjects()
		if err != nil {
			t.Fatalf("failed to load projects: %v", err)
		}

		// Filter by tag
		filtered := make([]*models.Project, 0)
		for _, p := range projects.Projects {
			if p.HasTag("Work") {
				filtered = append(filtered, p)
			}
		}

		if len(filtered) != 1 {
			t.Errorf("got %d projects, want 1", len(filtered))
		}
		if len(filtered) > 0 && filtered[0].Name != "favorite1" {
			t.Errorf("got project %q, want favorite1", filtered[0].Name)
		}
	})
}

func TestClearCache(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	store, err := storage.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Verify cache exists before clearing
	cachePath := filepath.Join(tmpDir, "cache.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("expected cache.json to exist before clearing")
	}

	// Verify cache has data
	cache, err := store.LoadCache()
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}
	if len(cache.Git) != 2 {
		t.Errorf("expected 2 git repos in cache, got %d", len(cache.Git))
	}

	// Clear cache
	if err := store.ClearCache(); err != nil {
		t.Fatalf("failed to clear cache: %v", err)
	}

	// Verify cache file is removed
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("expected cache.json to be removed after clearing")
	}

	// Verify loading cache after clearing returns empty
	cache, err = store.LoadCache()
	if err != nil {
		t.Fatalf("failed to load cache after clearing: %v", err)
	}
	if len(cache.Git) != 0 {
		t.Errorf("expected empty git cache after clearing, got %d", len(cache.Git))
	}
}

func TestClearCachePreservesFavorites(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	store, err := storage.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Load favorites before clearing cache
	favoritesBefore, err := store.LoadProjects()
	if err != nil {
		t.Fatalf("failed to load projects: %v", err)
	}
	countBefore := favoritesBefore.Count()

	// Clear cache
	if err := store.ClearCache(); err != nil {
		t.Fatalf("failed to clear cache: %v", err)
	}

	// Verify favorites are preserved
	favoritesAfter, err := store.LoadProjects()
	if err != nil {
		t.Fatalf("failed to load projects after clearing cache: %v", err)
	}
	if favoritesAfter.Count() != countBefore {
		t.Errorf("expected %d favorites after clearing cache, got %d", countBefore, favoritesAfter.Count())
	}

	// Verify projects.json still exists
	projectsPath := filepath.Join(tmpDir, "projects.json")
	if _, err := os.Stat(projectsPath); os.IsNotExist(err) {
		t.Error("expected projects.json to exist after clearing cache")
	}
}

func TestClearCacheNonExistent(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	// Remove cache file first
	cachePath := filepath.Join(tmpDir, "cache.json")
	os.Remove(cachePath)

	store, err := storage.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Clearing non-existent cache should not error
	if err := store.ClearCache(); err != nil {
		t.Errorf("ClearCache on non-existent cache failed: %v", err)
	}
}

// Tests for helper functions

func TestTypeFilter_ShowAll(t *testing.T) {
	tests := []struct {
		name   string
		filter TypeFilter
		want   bool
	}{
		{
			name:   "empty filter shows all",
			filter: TypeFilter{},
			want:   true,
		},
		{
			name:   "favorites only",
			filter: TypeFilter{Favorites: true},
			want:   false,
		},
		{
			name:   "git only",
			filter: TypeFilter{Git: true},
			want:   false,
		},
		{
			name:   "multiple filters",
			filter: TypeFilter{Favorites: true, Git: true},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.ShowAll(); got != tt.want {
				t.Errorf("ShowAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadFilteredProjects(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	store, err := storage.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	tests := []struct {
		name      string
		filter    TypeFilter
		wantCount int
	}{
		{
			name:      "load all",
			filter:    TypeFilter{},
			wantCount: 8,
		},
		{
			name:      "favorites only",
			filter:    TypeFilter{Favorites: true},
			wantCount: 2,
		},
		{
			name:      "git only",
			filter:    TypeFilter{Git: true},
			wantCount: 2,
		},
		{
			name:      "svn only",
			filter:    TypeFilter{SVN: true},
			wantCount: 1,
		},
		{
			name:      "mercurial only",
			filter:    TypeFilter{Mercurial: true},
			wantCount: 1,
		},
		{
			name:      "vscode only",
			filter:    TypeFilter{VSCode: true},
			wantCount: 1,
		},
		{
			name:      "any only",
			filter:    TypeFilter{Any: true},
			wantCount: 1,
		},
		{
			name:      "favorites and git",
			filter:    TypeFilter{Favorites: true, Git: true},
			wantCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := LoadFilteredProjects(store, tt.filter)
			if err != nil {
				t.Fatalf("LoadFilteredProjects failed: %v", err)
			}
			if len(projects) != tt.wantCount {
				t.Errorf("got %d projects, want %d", len(projects), tt.wantCount)
			}
		})
	}
}

func TestLoadFilteredProjects_InvalidCacheReturnsHelpfulError(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	store, err := storage.NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	cachePath := filepath.Join(tmpDir, "cache.json")
	if err := os.WriteFile(cachePath, []byte("{invalid-json"), 0644); err != nil {
		t.Fatalf("failed to write invalid cache: %v", err)
	}

	_, err = LoadFilteredProjects(store, TypeFilter{})
	if err == nil {
		t.Fatal("expected LoadFilteredProjects to fail for invalid cache")
	}

	if !strings.Contains(err.Error(), "failed to load cached projects") {
		t.Fatalf("expected helpful cache error, got %v", err)
	}

	if !strings.Contains(err.Error(), "projector clear-cache") {
		t.Fatalf("expected cache recovery hint, got %v", err)
	}
}

func TestMarkInvalidProjectsDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "valid")
	if err := os.MkdirAll(validPath, 0755); err != nil {
		t.Fatalf("failed to create valid path: %v", err)
	}

	projects := []*models.Project{
		{Name: "valid", RootPath: validPath, Enabled: true},
		{Name: "missing", RootPath: filepath.Join(tmpDir, "missing"), Enabled: true},
	}

	MarkInvalidProjectsDisabled(projects)

	if !projects[0].Enabled {
		t.Fatal("expected valid project to remain enabled")
	}
	if projects[1].Enabled {
		t.Fatal("expected missing project to be disabled")
	}
}

func TestExcludeProjectByPath(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "current")
	otherPath := filepath.Join(tmpDir, "other")

	projects := []*models.Project{
		{Name: "current", RootPath: currentPath, Enabled: true},
		{Name: "other", RootPath: otherPath, Enabled: true},
	}

	filtered := ExcludeProjectByPath(projects, filepath.Join(currentPath, "."))
	if len(filtered) != 1 {
		t.Fatalf("expected 1 project after exclusion, got %d", len(filtered))
	}
	if filtered[0].Name != "other" {
		t.Fatalf("expected remaining project to be other, got %s", filtered[0].Name)
	}
}

func TestFilterEnabled(t *testing.T) {
	projects := []*models.Project{
		{Name: "enabled1", Enabled: true},
		{Name: "disabled1", Enabled: false},
		{Name: "enabled2", Enabled: true},
		{Name: "disabled2", Enabled: false},
	}

	filtered := FilterEnabled(projects)

	if len(filtered) != 2 {
		t.Errorf("expected 2 enabled projects, got %d", len(filtered))
	}

	for _, p := range filtered {
		if !p.Enabled {
			t.Errorf("expected all filtered projects to be enabled, got disabled: %s", p.Name)
		}
	}
}

func TestFilterByTag(t *testing.T) {
	projects := []*models.Project{
		{Name: "work1", Tags: []string{"Work"}},
		{Name: "personal1", Tags: []string{"Personal"}},
		{Name: "work2", Tags: []string{"Work", "Go"}},
		{Name: "notags", Tags: []string{}},
	}

	tests := []struct {
		name      string
		tag       string
		wantCount int
		wantNames []string
	}{
		{
			name:      "filter by Work",
			tag:       "Work",
			wantCount: 2,
			wantNames: []string{"work1", "work2"},
		},
		{
			name:      "filter by Personal",
			tag:       "Personal",
			wantCount: 1,
			wantNames: []string{"personal1"},
		},
		{
			name:      "filter by Go",
			tag:       "Go",
			wantCount: 1,
			wantNames: []string{"work2"},
		},
		{
			name:      "empty tag returns all",
			tag:       "",
			wantCount: 4,
		},
		{
			name:      "non-existent tag",
			tag:       "NonExistent",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterByTag(projects, tt.tag)
			if len(filtered) != tt.wantCount {
				t.Errorf("got %d projects, want %d", len(filtered), tt.wantCount)
			}

			if tt.wantNames != nil {
				gotNames := make(map[string]bool)
				for _, p := range filtered {
					gotNames[p.Name] = true
				}
				for _, name := range tt.wantNames {
					if !gotNames[name] {
						t.Errorf("expected project %q not found", name)
					}
				}
			}
		})
	}
}

func TestFindProjectByName(t *testing.T) {
	projects := []*models.Project{
		{Name: "my-project"},
		{Name: "another-project"},
		{Name: "my-other-project"},
		{Name: "test"},
	}

	tests := []struct {
		name        string
		searchName  string
		wantProject string
		wantMatches int
		wantErr     bool
	}{
		{
			name:        "exact match",
			searchName:  "my-project",
			wantProject: "my-project",
			wantErr:     false,
		},
		{
			name:        "exact match case insensitive",
			searchName:  "MY-PROJECT",
			wantProject: "my-project",
			wantErr:     false,
		},
		{
			name:        "single partial match",
			searchName:  "test",
			wantProject: "test",
			wantErr:     false,
		},
		{
			name:        "multiple partial matches",
			searchName:  "my",
			wantMatches: 2,
			wantErr:     true,
		},
		{
			name:       "no match",
			searchName: "nonexistent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, matches, err := FindProjectByName(projects, tt.searchName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.wantMatches > 0 && len(matches) != tt.wantMatches {
					t.Errorf("expected %d matches, got %d", tt.wantMatches, len(matches))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if project == nil {
					t.Errorf("expected project, got nil")
				} else if project.Name != tt.wantProject {
					t.Errorf("got project %q, want %q", project.Name, tt.wantProject)
				}
			}
		})
	}
}

func TestFindProjectByQuery_FilterOnFullPath(t *testing.T) {
	projects := []*models.Project{
		{Name: "api", RootPath: "/work/backend/api"},
		{Name: "docs", RootPath: "/notes/docs"},
	}

	project, matches, err := FindProjectByQuery(projects, "backend/api", true)
	if err != nil {
		t.Fatalf("expected path-aware query to succeed, got %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no ambiguous matches, got %d", len(matches))
	}
	if project == nil || project.Name != "api" {
		t.Fatalf("expected api project from full path query, got %#v", project)
	}

	_, _, err = FindProjectByQuery(projects, "backend/api", false)
	if err == nil {
		t.Fatal("expected full path query to fail when path matching is disabled")
	}
}
