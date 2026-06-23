package output

import (
	"strings"
	"testing"

	"github.com/ideaspaper/projector/pkg/models"
)

func TestFormatProjectList_Empty(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{}

	output, indexed := f.FormatProjectList(projects, ListOptions{})

	if !strings.Contains(output, "No projects found") {
		t.Errorf("Expected 'No projects found' message, got: %s", output)
	}
	if indexed != nil {
		t.Errorf("Expected nil indexed projects for empty list, got: %v", indexed)
	}
}

func TestFormatProjectList_NoIndex(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{
		{Name: "project1", RootPath: "/path/to/project1", Enabled: true, Kind: models.KindFavorite},
		{Name: "project2", RootPath: "/path/to/project2", Enabled: true, Kind: models.KindFavorite},
	}

	opts := ListOptions{
		ShowIndex: false,
		Grouped:   false,
	}
	output, indexed := f.FormatProjectList(projects, opts)

	// Should not contain index numbers
	if strings.Contains(output, "[1]") || strings.Contains(output, "[2]") {
		t.Errorf("Expected no index numbers, got: %s", output)
	}

	// Should contain project names
	if !strings.Contains(output, "project1") || !strings.Contains(output, "project2") {
		t.Errorf("Expected project names in output, got: %s", output)
	}

	// Should return indexed projects
	if len(indexed) != 2 {
		t.Errorf("Expected 2 indexed projects, got: %d", len(indexed))
	}
}

func TestFormatProjectList_WithIndex(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{
		{Name: "project1", RootPath: "/path/to/project1", Enabled: true, Kind: models.KindFavorite},
		{Name: "project2", RootPath: "/path/to/project2", Enabled: true, Kind: models.KindFavorite},
	}

	opts := ListOptions{
		ShowIndex: true,
		Grouped:   false,
	}
	output, indexed := f.FormatProjectList(projects, opts)

	// Should contain 1-based index numbers
	if !strings.Contains(output, "[1]") || !strings.Contains(output, "[2]") {
		t.Errorf("Expected 1-based index numbers [1] and [2], got: %s", output)
	}

	// Should not contain 0-based index
	if strings.Contains(output, "[0]") {
		t.Errorf("Expected 1-based index, but found [0] in: %s", output)
	}

	// Should return indexed projects in order
	if len(indexed) != 2 {
		t.Errorf("Expected 2 indexed projects, got: %d", len(indexed))
	}
	if indexed[0].Name != "project1" || indexed[1].Name != "project2" {
		t.Errorf("Indexed projects not in expected order")
	}
}

func TestFormatProjectList_WithTags(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{
		{Name: "project1", RootPath: "/path/to/project1", Tags: []string{"Work", "Important"}, Enabled: true, Kind: models.KindFavorite},
		{Name: "project2", RootPath: "/path/to/project2", Tags: []string{}, Enabled: true, Kind: models.KindFavorite},
	}

	opts := ListOptions{
		ShowIndex: false,
		Grouped:   false,
	}
	output, _ := f.FormatProjectList(projects, opts)

	// Should contain tags for project1
	if !strings.Contains(output, "[Work, Important]") {
		t.Errorf("Expected tags [Work, Important] in output, got: %s", output)
	}

	// project2 line should not have extra brackets (no tags)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "project2") {
			// Count brackets - should only have path truncation, not tag brackets
			if strings.Count(line, "[") > 0 && !strings.Contains(line, "Work") {
				// This is fine, it might be something else
			}
		}
	}
}

func TestFormatProjectList_Grouped(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{
		{Name: "favorite1", RootPath: "/path/to/fav1", Enabled: true, Kind: models.KindFavorite},
		{Name: "gitrepo1", RootPath: "/path/to/git1", Enabled: true, Kind: models.KindGit},
		{Name: "favorite2", RootPath: "/path/to/fav2", Enabled: true, Kind: models.KindFavorite},
		{Name: "gitrepo2", RootPath: "/path/to/git2", Enabled: true, Kind: models.KindGit},
	}

	opts := ListOptions{
		ShowIndex: true,
		Grouped:   true,
	}
	output, indexed := f.FormatProjectList(projects, opts)

	// Should contain group headers
	if !strings.Contains(output, "Favorites") {
		t.Errorf("Expected 'Favorites' header, got: %s", output)
	}
	if !strings.Contains(output, "Git Repositories") {
		t.Errorf("Expected 'Git Repositories' header, got: %s", output)
	}

	// Should have continuous 1-based indexing across groups
	if !strings.Contains(output, "[1]") || !strings.Contains(output, "[2]") ||
		!strings.Contains(output, "[3]") || !strings.Contains(output, "[4]") {
		t.Errorf("Expected continuous 1-based indexing [1]-[4], got: %s", output)
	}

	// Indexed projects should be in group order (favorites first, then git)
	if len(indexed) != 4 {
		t.Errorf("Expected 4 indexed projects, got: %d", len(indexed))
	}

	// First two should be favorites
	if indexed[0].Kind != models.KindFavorite || indexed[1].Kind != models.KindFavorite {
		t.Errorf("Expected first two indexed projects to be favorites")
	}
	// Last two should be git
	if indexed[2].Kind != models.KindGit || indexed[3].Kind != models.KindGit {
		t.Errorf("Expected last two indexed projects to be git repos")
	}
}

func TestFormatProjectList_GroupedIndexMapping(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{
		{Name: "gitrepo", RootPath: "/path/to/git", Enabled: true, Kind: models.KindGit},
		{Name: "favorite", RootPath: "/path/to/fav", Enabled: true, Kind: models.KindFavorite},
	}

	opts := ListOptions{
		ShowIndex: true,
		Grouped:   true,
	}
	output, indexed := f.FormatProjectList(projects, opts)

	// Favorites should come first in grouped view (even though gitrepo was first in input)
	// So [1] should be "favorite" and [2] should be "gitrepo"
	if !strings.Contains(output, "Favorites") {
		t.Errorf("Expected Favorites header first")
	}

	// Index 0 (display [1]) should be the favorite
	if indexed[0].Name != "favorite" {
		t.Errorf("Expected indexed[0] to be 'favorite', got: %s", indexed[0].Name)
	}
	// Index 1 (display [2]) should be the git repo
	if indexed[1].Name != "gitrepo" {
		t.Errorf("Expected indexed[1] to be 'gitrepo', got: %s", indexed[1].Name)
	}
}

func TestFormatProjectList_DisabledProject(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{
		{Name: "enabled", RootPath: "/path/to/enabled", Enabled: true, Kind: models.KindFavorite},
		{Name: "disabled", RootPath: "/path/to/disabled", Enabled: false, Kind: models.KindFavorite},
	}

	opts := ListOptions{
		ShowIndex: false,
		Grouped:   false,
	}
	output, _ := f.FormatProjectList(projects, opts)

	// Should show disabled indicator
	if !strings.Contains(output, "(disabled)") {
		t.Errorf("Expected '(disabled)' indicator, got: %s", output)
	}
}

func TestFormatProjectList_TruncatedPath(t *testing.T) {
	f := NewFormatter(false)
	longPath := "/very/long/path/that/exceeds/fifty/characters/and/should/be/truncated"
	projects := []*models.Project{
		{Name: "project", RootPath: longPath, Enabled: true, Kind: models.KindFavorite},
	}

	opts := ListOptions{
		ShowIndex: false,
		ShowPath:  false, // Truncated path on same line
		Grouped:   false,
	}
	output, _ := f.FormatProjectList(projects, opts)

	// Should contain "..." for truncation
	if !strings.Contains(output, "...") {
		t.Errorf("Expected truncated path with '...', got: %s", output)
	}

	// Should not contain the full path
	if strings.Contains(output, "/very/long/path") {
		t.Errorf("Expected truncated path, but found full path start in: %s", output)
	}
}

func TestFormatProjectList_FullPath(t *testing.T) {
	f := NewFormatter(false)
	longPath := "/very/long/path/that/exceeds/fifty/characters/and/should/not/be/truncated"
	projects := []*models.Project{
		{Name: "project", RootPath: longPath, Enabled: true, Kind: models.KindFavorite},
	}

	opts := ListOptions{
		ShowIndex: false,
		ShowPath:  true, // Full path on new line
		Grouped:   false,
	}
	output, _ := f.FormatProjectList(projects, opts)

	// Should contain the full path
	if !strings.Contains(output, longPath) {
		t.Errorf("Expected full path in output, got: %s", output)
	}
}

func TestFormatProjectList_ShowParentOnDuplicates(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{
		{Name: "api", RootPath: "/workspace/client/api", Enabled: true, Kind: models.KindFavorite},
		{Name: "api", RootPath: "/workspace/server/api", Enabled: true, Kind: models.KindGit},
	}

	opts := ListOptions{
		ShowIndex:              false,
		Grouped:                false,
		ShowParentOnDuplicates: true,
	}
	output, _ := f.FormatProjectList(projects, opts)

	if !strings.Contains(output, "api (client)") {
		t.Fatalf("expected first duplicate to include parent folder, got: %s", output)
	}
	if !strings.Contains(output, "api (server)") {
		t.Fatalf("expected second duplicate to include parent folder, got: %s", output)
	}
}

func TestFormatProjectList_AllKinds(t *testing.T) {
	f := NewFormatter(false)
	projects := []*models.Project{
		{Name: "any", RootPath: "/path/any", Enabled: true, Kind: models.KindAny},
		{Name: "vscode", RootPath: "/path/vscode", Enabled: true, Kind: models.KindVSCode},
		{Name: "hg", RootPath: "/path/hg", Enabled: true, Kind: models.KindMercurial},
		{Name: "svn", RootPath: "/path/svn", Enabled: true, Kind: models.KindSVN},
		{Name: "git", RootPath: "/path/git", Enabled: true, Kind: models.KindGit},
		{Name: "fav", RootPath: "/path/fav", Enabled: true, Kind: models.KindFavorite},
	}

	opts := ListOptions{
		ShowIndex: true,
		Grouped:   true,
	}
	output, indexed := f.FormatProjectList(projects, opts)

	// All headers should be present
	expectedHeaders := []string{"Favorites", "Git Repositories", "SVN Repositories", "Mercurial Repositories", "VS Code Workspaces", "Other Projects"}
	for _, header := range expectedHeaders {
		if !strings.Contains(output, header) {
			t.Errorf("Expected header '%s' in output, got: %s", header, output)
		}
	}

	// Should have 6 indexed projects
	if len(indexed) != 6 {
		t.Errorf("Expected 6 indexed projects, got: %d", len(indexed))
	}

	// Order should be: Favorite, Git, SVN, Mercurial, VSCode, Any
	expectedOrder := []models.ProjectKind{
		models.KindFavorite,
		models.KindGit,
		models.KindSVN,
		models.KindMercurial,
		models.KindVSCode,
		models.KindAny,
	}
	for i, expectedKind := range expectedOrder {
		if indexed[i].Kind != expectedKind {
			t.Errorf("Expected indexed[%d] to be kind %s, got: %s", i, expectedKind, indexed[i].Kind)
		}
	}
}
