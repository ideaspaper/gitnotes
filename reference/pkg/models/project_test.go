package models

import (
	"testing"
)

func TestNewProject(t *testing.T) {
	p := NewProject("test-project", "/path/to/project")

	if p.Name != "test-project" {
		t.Errorf("expected name 'test-project', got '%s'", p.Name)
	}
	if p.RootPath != "/path/to/project" {
		t.Errorf("expected path '/path/to/project', got '%s'", p.RootPath)
	}
	if !p.Enabled {
		t.Error("expected project to be enabled by default")
	}
	if p.Kind != KindFavorite {
		t.Errorf("expected kind KindFavorite, got '%s'", p.Kind)
	}
	if len(p.Tags) != 0 {
		t.Errorf("expected empty tags, got %v", p.Tags)
	}
}

func TestProject_HasTag(t *testing.T) {
	p := &Project{
		Name:     "test",
		RootPath: "/test",
		Tags:     []string{"Work", "Go", "Backend"},
		Enabled:  true,
	}

	tests := []struct {
		tag      string
		expected bool
	}{
		{"Work", true},
		{"Go", true},
		{"Backend", true},
		{"Personal", false},
		{"work", false}, // case sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			if got := p.HasTag(tt.tag); got != tt.expected {
				t.Errorf("HasTag(%q) = %v, want %v", tt.tag, got, tt.expected)
			}
		})
	}
}

func TestProject_AddTag(t *testing.T) {
	p := NewProject("test", "/test")

	// Add a tag
	p.AddTag("Work")
	if !p.HasTag("Work") {
		t.Error("expected project to have tag 'Work' after AddTag")
	}
	if len(p.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(p.Tags))
	}

	// Add same tag again (should not duplicate)
	p.AddTag("Work")
	if len(p.Tags) != 1 {
		t.Errorf("expected 1 tag after duplicate add, got %d", len(p.Tags))
	}

	// Add another tag
	p.AddTag("Personal")
	if len(p.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(p.Tags))
	}
}

func TestProject_RemoveTag(t *testing.T) {
	p := &Project{
		Name:     "test",
		RootPath: "/test",
		Tags:     []string{"Work", "Go", "Backend"},
		Enabled:  true,
	}

	// Remove existing tag
	p.RemoveTag("Go")
	if p.HasTag("Go") {
		t.Error("expected tag 'Go' to be removed")
	}
	if len(p.Tags) != 2 {
		t.Errorf("expected 2 tags after removal, got %d", len(p.Tags))
	}

	// Remove non-existent tag (should not panic)
	p.RemoveTag("NonExistent")
	if len(p.Tags) != 2 {
		t.Errorf("expected 2 tags after removing non-existent, got %d", len(p.Tags))
	}

	// Remove remaining tags
	p.RemoveTag("Work")
	p.RemoveTag("Backend")
	if len(p.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(p.Tags))
	}
}

func TestNewProjectList(t *testing.T) {
	pl := NewProjectList(KindGit)

	if pl.Kind != KindGit {
		t.Errorf("expected kind KindGit, got '%s'", pl.Kind)
	}
	if len(pl.Projects) != 0 {
		t.Errorf("expected empty projects list, got %d", len(pl.Projects))
	}
}

func TestProjectList_Add(t *testing.T) {
	pl := NewProjectList(KindGit)

	p1 := NewProject("project1", "/path1")
	p2 := NewProject("project2", "/path2")

	pl.Add(p1)
	if pl.Count() != 1 {
		t.Errorf("expected count 1, got %d", pl.Count())
	}
	if p1.Kind != KindGit {
		t.Errorf("expected project kind to be set to KindGit, got '%s'", p1.Kind)
	}

	pl.Add(p2)
	if pl.Count() != 2 {
		t.Errorf("expected count 2, got %d", pl.Count())
	}
}

func TestProjectList_Remove(t *testing.T) {
	pl := NewProjectList(KindFavorite)
	pl.Add(NewProject("project1", "/path1"))
	pl.Add(NewProject("project2", "/path2"))
	pl.Add(NewProject("project3", "/path3"))

	// Remove existing project
	if !pl.Remove("project2") {
		t.Error("expected Remove to return true for existing project")
	}
	if pl.Count() != 2 {
		t.Errorf("expected count 2 after removal, got %d", pl.Count())
	}
	if pl.FindByName("project2") != nil {
		t.Error("expected project2 to be removed")
	}

	// Remove non-existent project
	if pl.Remove("nonexistent") {
		t.Error("expected Remove to return false for non-existent project")
	}
	if pl.Count() != 2 {
		t.Errorf("expected count still 2, got %d", pl.Count())
	}

	// Remove is case-insensitive
	pl.Add(NewProject("TestProject", "/test"))
	if !pl.Remove("testproject") {
		t.Error("expected Remove to be case-insensitive")
	}
}

func TestProjectList_FindByName(t *testing.T) {
	pl := NewProjectList(KindFavorite)
	pl.Add(NewProject("alpha", "/alpha"))
	pl.Add(NewProject("beta", "/beta"))
	pl.Add(NewProject("gamma", "/gamma"))

	// Find existing
	p := pl.FindByName("beta")
	if p == nil {
		t.Fatal("expected to find project 'beta'")
	}
	if p.Name != "beta" {
		t.Errorf("expected name 'beta', got '%s'", p.Name)
	}

	// Find non-existent
	if pl.FindByName("delta") != nil {
		t.Error("expected nil for non-existent project")
	}

	// Case insensitive
	if pl.FindByName("Beta") == nil {
		t.Error("expected FindByName to be case-insensitive")
	}
	if pl.FindByName("BETA") == nil {
		t.Error("expected FindByName to be case-insensitive for BETA")
	}
}

func TestProjectList_FindByPath(t *testing.T) {
	pl := NewProjectList(KindFavorite)
	pl.Add(NewProject("project1", "/path/to/project1"))
	pl.Add(NewProject("project2", "/path/to/project2"))

	// Find existing
	p := pl.FindByPath("/path/to/project1")
	if p == nil {
		t.Fatal("expected to find project by path")
	}
	if p.Name != "project1" {
		t.Errorf("expected name 'project1', got '%s'", p.Name)
	}

	// Find non-existent
	if pl.FindByPath("/nonexistent") != nil {
		t.Error("expected nil for non-existent path")
	}
}

func TestProjectList_FilterByTag(t *testing.T) {
	pl := NewProjectList(KindFavorite)

	p1 := NewProject("project1", "/path1")
	p1.Tags = []string{"Work", "Go"}
	pl.Add(p1)

	p2 := NewProject("project2", "/path2")
	p2.Tags = []string{"Personal", "Go"}
	pl.Add(p2)

	p3 := NewProject("project3", "/path3")
	p3.Tags = []string{"Work", "Python"}
	pl.Add(p3)

	// Filter by "Work"
	workProjects := pl.FilterByTag("Work")
	if len(workProjects) != 2 {
		t.Errorf("expected 2 Work projects, got %d", len(workProjects))
	}

	// Filter by "Go"
	goProjects := pl.FilterByTag("Go")
	if len(goProjects) != 2 {
		t.Errorf("expected 2 Go projects, got %d", len(goProjects))
	}

	// Filter by "Personal"
	personalProjects := pl.FilterByTag("Personal")
	if len(personalProjects) != 1 {
		t.Errorf("expected 1 Personal project, got %d", len(personalProjects))
	}

	// Filter by non-existent tag
	noneProjects := pl.FilterByTag("NonExistent")
	if len(noneProjects) != 0 {
		t.Errorf("expected 0 projects for non-existent tag, got %d", len(noneProjects))
	}
}

func TestProjectList_FilterEnabled(t *testing.T) {
	pl := NewProjectList(KindFavorite)

	p1 := NewProject("enabled1", "/path1")
	p1.Enabled = true
	pl.Add(p1)

	p2 := NewProject("disabled", "/path2")
	p2.Enabled = false
	pl.Add(p2)

	p3 := NewProject("enabled2", "/path3")
	p3.Enabled = true
	pl.Add(p3)

	enabled := pl.FilterEnabled()
	if len(enabled) != 2 {
		t.Errorf("expected 2 enabled projects, got %d", len(enabled))
	}

	for _, p := range enabled {
		if !p.Enabled {
			t.Errorf("FilterEnabled returned disabled project: %s", p.Name)
		}
	}
}

func TestProjectList_Count(t *testing.T) {
	pl := NewProjectList(KindFavorite)

	if pl.Count() != 0 {
		t.Errorf("expected count 0 for empty list, got %d", pl.Count())
	}

	pl.Add(NewProject("p1", "/p1"))
	if pl.Count() != 1 {
		t.Errorf("expected count 1, got %d", pl.Count())
	}

	pl.Add(NewProject("p2", "/p2"))
	pl.Add(NewProject("p3", "/p3"))
	if pl.Count() != 3 {
		t.Errorf("expected count 3, got %d", pl.Count())
	}
}

func TestProjectKind_Values(t *testing.T) {
	// Ensure constants have expected string values
	tests := []struct {
		kind     ProjectKind
		expected string
	}{
		{KindFavorite, "favorites"},
		{KindGit, "git"},
		{KindSVN, "svn"},
		{KindMercurial, "mercurial"},
		{KindVSCode, "vscode"},
		{KindAny, "any"},
	}

	for _, tt := range tests {
		if string(tt.kind) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.kind)
		}
	}
}
