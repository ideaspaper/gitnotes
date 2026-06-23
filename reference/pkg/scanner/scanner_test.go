package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/paths"
)

func TestNewScanner(t *testing.T) {
	s := NewScanner(ScannerGit)

	if s.scannerType != ScannerGit {
		t.Errorf("expected scanner type ScannerGit, got %s", s.scannerType)
	}
	if s.maxDepth != 4 {
		t.Errorf("expected default maxDepth 4, got %d", s.maxDepth)
	}
	if len(s.baseFolders) != 0 {
		t.Error("expected empty base folders")
	}
}

func TestScanner_SetBaseFolders(t *testing.T) {
	s := NewScanner(ScannerGit)

	home, _ := os.UserHomeDir()
	s.SetBaseFolders([]string{"~/projects", "/absolute/path"})

	if len(s.baseFolders) != 2 {
		t.Fatalf("expected 2 base folders, got %d", len(s.baseFolders))
	}
	if s.baseFolders[0] != home+"/projects" {
		t.Errorf("expected expanded path, got %s", s.baseFolders[0])
	}
	if s.baseFolders[1] != "/absolute/path" {
		t.Errorf("expected /absolute/path, got %s", s.baseFolders[1])
	}
}

func TestScanner_SetIgnoredFolders(t *testing.T) {
	s := NewScanner(ScannerGit)
	s.SetIgnoredFolders([]string{"node_modules", "vendor"})

	if len(s.ignoredFolders) != 2 {
		t.Errorf("expected 2 ignored folders, got %d", len(s.ignoredFolders))
	}
}

func TestScanner_SetMaxDepth(t *testing.T) {
	s := NewScanner(ScannerGit)
	s.SetMaxDepth(10)

	if s.maxDepth != 10 {
		t.Errorf("expected maxDepth 10, got %d", s.maxDepth)
	}
}

func TestScanner_ScanGit(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create a git repo
	gitRepo := filepath.Join(tmpDir, "my-git-repo")
	os.MkdirAll(filepath.Join(gitRepo, ".git"), 0755)

	// Create a non-git folder
	normalDir := filepath.Join(tmpDir, "normal-folder")
	os.MkdirAll(normalDir, 0755)

	// Create nested git repo
	nestedRepo := filepath.Join(tmpDir, "parent", "nested-repo")
	os.MkdirAll(filepath.Join(nestedRepo, ".git"), 0755)

	s := NewScanner(ScannerGit)
	s.SetBaseFolders([]string{tmpDir})
	s.SetMaxDepth(4)

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 git repos, got %d", len(projects))
	}

	// Verify project properties
	for _, p := range projects {
		if p.Kind != models.KindGit {
			t.Errorf("expected kind KindGit, got %s", p.Kind)
		}
		if !p.Enabled {
			t.Error("expected project to be enabled")
		}
	}
}

func TestScanner_ScanSVN(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an SVN repo
	svnRepo := filepath.Join(tmpDir, "svn-repo")
	os.MkdirAll(filepath.Join(svnRepo, ".svn"), 0755)

	s := NewScanner(ScannerSVN)
	s.SetBaseFolders([]string{tmpDir})

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("expected 1 svn repo, got %d", len(projects))
	}

	if projects[0].Kind != models.KindSVN {
		t.Errorf("expected kind KindSVN, got %s", projects[0].Kind)
	}
}

func TestScanner_ScanMercurial(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Mercurial repo
	hgRepo := filepath.Join(tmpDir, "hg-repo")
	os.MkdirAll(filepath.Join(hgRepo, ".hg"), 0755)

	s := NewScanner(ScannerMercurial)
	s.SetBaseFolders([]string{tmpDir})

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("expected 1 hg repo, got %d", len(projects))
	}

	if projects[0].Kind != models.KindMercurial {
		t.Errorf("expected kind KindMercurial, got %s", projects[0].Kind)
	}
}

func TestScanner_ScanVSCode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a VS Code workspace
	workspaceDir := filepath.Join(tmpDir, "my-workspace")
	os.MkdirAll(workspaceDir, 0755)
	os.WriteFile(filepath.Join(workspaceDir, "project.code-workspace"), []byte("{}"), 0644)

	s := NewScanner(ScannerVSCode)
	s.SetBaseFolders([]string{tmpDir})

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("expected 1 vscode workspace, got %d", len(projects))
	}

	if projects[0].Kind != models.KindVSCode {
		t.Errorf("expected kind KindVSCode, got %s", projects[0].Kind)
	}
}

func TestScanner_ScanIgnoresFolders(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a git repo inside node_modules (should be ignored)
	nodeModules := filepath.Join(tmpDir, "node_modules", "some-package")
	os.MkdirAll(filepath.Join(nodeModules, ".git"), 0755)

	// Create a normal git repo
	gitRepo := filepath.Join(tmpDir, "my-repo")
	os.MkdirAll(filepath.Join(gitRepo, ".git"), 0755)

	s := NewScanner(ScannerGit)
	s.SetBaseFolders([]string{tmpDir})
	s.SetIgnoredFolders([]string{"node_modules"})

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("expected 1 git repo (node_modules should be ignored), got %d", len(projects))
	}
}

func TestScanner_ScanRespectsMaxDepth(t *testing.T) {
	tmpDir := t.TempDir()

	// Create repo at depth 1
	repo1 := filepath.Join(tmpDir, "repo1")
	os.MkdirAll(filepath.Join(repo1, ".git"), 0755)

	// Create repo at depth 3
	repo3 := filepath.Join(tmpDir, "a", "b", "repo3")
	os.MkdirAll(filepath.Join(repo3, ".git"), 0755)

	// Create repo at depth 5 (should be skipped with maxDepth 4)
	repo5 := filepath.Join(tmpDir, "a", "b", "c", "d", "repo5")
	os.MkdirAll(filepath.Join(repo5, ".git"), 0755)

	s := NewScanner(ScannerGit)
	s.SetBaseFolders([]string{tmpDir})
	s.SetMaxDepth(4)

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find repo1 and repo3, but not repo5
	if len(projects) != 2 {
		t.Errorf("expected 2 repos with maxDepth 4, got %d", len(projects))
	}
}

func TestScanner_ScanNonExistentBasePath(t *testing.T) {
	s := NewScanner(ScannerGit)
	s.SetBaseFolders([]string{"/nonexistent/path/that/does/not/exist"})

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan should not fail for non-existent path: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestScanner_DeduplicatesNames(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two repos with same name in different directories
	repo1 := filepath.Join(tmpDir, "dir1", "api")
	os.MkdirAll(filepath.Join(repo1, ".git"), 0755)

	repo2 := filepath.Join(tmpDir, "dir2", "api")
	os.MkdirAll(filepath.Join(repo2, ".git"), 0755)

	s := NewScanner(ScannerGit)
	s.SetBaseFolders([]string{tmpDir})

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	// Check names are deduplicated
	names := make(map[string]bool)
	for _, p := range projects {
		if names[p.Name] {
			t.Errorf("duplicate name found: %s", p.Name)
		}
		names[p.Name] = true
	}

	// One should be "api" and one should be "api-2"
	if !names["api"] {
		t.Error("expected to find 'api'")
	}
	if !names["api-2"] {
		t.Error("expected to find 'api-2'")
	}
}

func TestScanner_IgnoreWithinProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create parent repo
	parentRepo := filepath.Join(tmpDir, "parent-repo")
	os.MkdirAll(filepath.Join(parentRepo, ".git"), 0755)

	// Create nested repo inside parent
	nestedRepo := filepath.Join(parentRepo, "packages", "nested-repo")
	os.MkdirAll(filepath.Join(nestedRepo, ".git"), 0755)

	// Test with ignoreWithinProjects = false (default)
	s := NewScanner(ScannerGit)
	s.SetBaseFolders([]string{tmpDir})
	s.SetIgnoreWithinProjects(false)

	projects, _ := s.Scan()
	if len(projects) != 2 {
		t.Errorf("expected 2 projects (nested allowed), got %d", len(projects))
	}

	// Test with ignoreWithinProjects = true
	s2 := NewScanner(ScannerGit)
	s2.SetBaseFolders([]string{tmpDir})
	s2.SetIgnoreWithinProjects(true)

	projects2, _ := s2.Scan()
	if len(projects2) != 1 {
		t.Errorf("expected 1 project (nested ignored), got %d", len(projects2))
	}
}

func TestScanner_IgnoreWithinProjectsPrunesDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based pruning test is not reliable on windows")
	}

	tmpDir := t.TempDir()
	parentRepo := filepath.Join(tmpDir, "parent-repo")
	if err := os.MkdirAll(filepath.Join(parentRepo, ".git"), 0755); err != nil {
		t.Fatalf("failed to create parent repo: %v", err)
	}

	blockedDir := filepath.Join(parentRepo, "blocked")
	if err := os.MkdirAll(blockedDir, 0755); err != nil {
		t.Fatalf("failed to create blocked dir: %v", err)
	}
	if err := os.Chmod(blockedDir, 0); err != nil {
		t.Fatalf("failed to chmod blocked dir: %v", err)
	}
	defer os.Chmod(blockedDir, 0755)

	var errorCount int
	s := NewScanner(ScannerGit)
	s.SetBaseFolders([]string{tmpDir})
	s.SetIgnoreWithinProjects(true)
	s.SetErrorHandler(func(path string, err error) {
		errorCount++
	})

	projects, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if errorCount != 0 {
		t.Fatalf("expected scanner to prune descendant traversal, got %d errors", errorCount)
	}
}

func TestScanner_SkipsHiddenDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hidden directory with a git repo
	hiddenDir := filepath.Join(tmpDir, ".hidden", "repo")
	os.MkdirAll(filepath.Join(hiddenDir, ".git"), 0755)

	// Create a normal git repo
	normalRepo := filepath.Join(tmpDir, "normal-repo")
	os.MkdirAll(filepath.Join(normalRepo, ".git"), 0755)

	s := NewScanner(ScannerGit)
	s.SetBaseFolders([]string{tmpDir})

	projects, _ := s.Scan()

	// Should only find normal-repo, not the one in .hidden
	if len(projects) != 1 {
		t.Errorf("expected 1 project (hidden should be skipped), got %d", len(projects))
	}
}

func TestScanner_IsIgnored_GlobPattern(t *testing.T) {
	s := NewScanner(ScannerGit)
	s.SetIgnoredFolders([]string{"node_modules", "*.log", "test*"})

	tests := []struct {
		name     string
		expected bool
	}{
		{"node_modules", true},
		{"vendor", false},
		{"error.log", true},
		{"test", true},
		{"testing", true},
		{"mytest", false}, // glob doesn't match middle
		{"logs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.isIgnored(tt.name); got != tt.expected {
				t.Errorf("isIgnored(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory
	testDir := filepath.Join(tmpDir, "testdir")
	os.Mkdir(testDir, 0755)

	// Create a file
	testFile := filepath.Join(tmpDir, "testfile")
	os.WriteFile(testFile, []byte("test"), 0644)

	if !dirExists(testDir) {
		t.Error("expected dirExists to return true for directory")
	}

	if dirExists(testFile) {
		t.Error("expected dirExists to return false for file")
	}

	if dirExists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("expected dirExists to return false for non-existent path")
	}
}

func TestFileExistsWithExt(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	os.WriteFile(filepath.Join(tmpDir, "project.code-workspace"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Readme"), 0644)

	if !fileExistsWithExt(tmpDir, ".code-workspace") {
		t.Error("expected to find .code-workspace file")
	}

	if !fileExistsWithExt(tmpDir, ".md") {
		t.Error("expected to find .md file")
	}

	if fileExistsWithExt(tmpDir, ".json") {
		t.Error("expected not to find .json file")
	}
}

func TestExpandPaths(t *testing.T) {
	home, _ := os.UserHomeDir()

	inputPaths := []string{"~/projects", "$HOME/work", "/absolute"}
	expanded := paths.ExpandAll(inputPaths)

	if expanded[0] != home+"/projects" {
		t.Errorf("expected %s/projects, got %s", home, expanded[0])
	}
	if expanded[1] != home+"/work" {
		t.Errorf("expected %s/work, got %s", home, expanded[1])
	}
	if expanded[2] != "/absolute" {
		t.Errorf("expected /absolute, got %s", expanded[2])
	}
}

func TestScannerType_Values(t *testing.T) {
	tests := []struct {
		scannerType ScannerType
		expected    string
	}{
		{ScannerGit, "git"},
		{ScannerSVN, "svn"},
		{ScannerMercurial, "mercurial"},
		{ScannerVSCode, "vscode"},
		{ScannerAny, "any"},
	}

	for _, tt := range tests {
		if string(tt.scannerType) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.scannerType)
		}
	}
}

func TestScanner_GetProjectKind(t *testing.T) {
	tests := []struct {
		scannerType  ScannerType
		expectedKind models.ProjectKind
	}{
		{ScannerGit, models.KindGit},
		{ScannerSVN, models.KindSVN},
		{ScannerMercurial, models.KindMercurial},
		{ScannerVSCode, models.KindVSCode},
		{ScannerAny, models.KindAny},
	}

	for _, tt := range tests {
		t.Run(string(tt.scannerType), func(t *testing.T) {
			s := NewScanner(tt.scannerType)
			if got := s.getProjectKind(); got != tt.expectedKind {
				t.Errorf("getProjectKind() = %s, want %s", got, tt.expectedKind)
			}
		})
	}
}

func TestScanner_SetErrorHandler(t *testing.T) {
	s := NewScanner(ScannerGit)

	var capturedPath string
	var capturedErr error

	handler := func(path string, err error) {
		capturedPath = path
		capturedErr = err
	}

	s.SetErrorHandler(handler)

	// Set a non-existent base folder to trigger an error
	s.SetBaseFolders([]string{"/nonexistent/path/that/does/not/exist"})
	s.Scan()

	// Verify the error handler was called
	if capturedPath != "/nonexistent/path/that/does/not/exist" {
		t.Errorf("expected error handler to be called with path, got: %s", capturedPath)
	}
	if capturedErr == nil {
		t.Error("expected error handler to be called with non-nil error")
	}
}

func TestScanner_ErrorHandlerNotSet(t *testing.T) {
	s := NewScanner(ScannerGit)

	// Should not panic when error handler is not set
	s.SetBaseFolders([]string{"/nonexistent/path"})
	_, err := s.Scan()

	// Scan should complete without error even though base folder doesn't exist
	if err != nil {
		t.Errorf("expected Scan to succeed, got error: %v", err)
	}
}
