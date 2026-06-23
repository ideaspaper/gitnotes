// Package storage provides JSON-based persistence for projects and cache,
// including thread-safe operations for loading, saving, and managing
// favorite projects and auto-detected repository caches.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ideaspaper/projector/pkg/models"
	"github.com/ideaspaper/projector/pkg/paths"
)

const (
	projectsFileName = "projects.json"
	cacheFileName    = "cache.json"
)

// Storage handles persistence of projects
type Storage struct {
	basePath string
	mu       sync.RWMutex
}

// CachedProjects holds auto-detected project caches
type CachedProjects struct {
	Git       []*models.Project `json:"git,omitempty"`
	SVN       []*models.Project `json:"svn,omitempty"`
	Mercurial []*models.Project `json:"mercurial,omitempty"`
	VSCode    []*models.Project `json:"vscode,omitempty"`
	Any       []*models.Project `json:"any,omitempty"`
}

// NewStorage creates a new storage instance
func NewStorage(basePath string) (*Storage, error) {
	if basePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(home, ".projector")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &Storage{
		basePath: basePath,
	}, nil
}

// GetBasePath returns the storage base path
func (s *Storage) GetBasePath() string {
	return s.basePath
}

// GetProjectsPath returns the path to projects.json
func (s *Storage) GetProjectsPath() string {
	return filepath.Join(s.basePath, projectsFileName)
}

// LoadProjects loads saved (favorite) projects from projects.json
func (s *Storage) LoadProjects() (*models.ProjectList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectList := models.NewProjectList(models.KindFavorite)
	projectsPath := s.GetProjectsPath()

	data, err := os.ReadFile(projectsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return projectList, nil
		}
		return nil, fmt.Errorf("failed to read projects file: %w", err)
	}

	var projects []*models.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse projects file: %w", err)
	}

	for _, p := range projects {
		p.Kind = models.KindFavorite
		p.RootPath = paths.Expand(p.RootPath)
		projectList.Projects = append(projectList.Projects, p)
	}

	return projectList, nil
}

// SaveProjects saves favorite projects to projects.json
func (s *Storage) SaveProjects(projects *models.ProjectList) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Prepare projects for saving (collapse paths)
	saveProjects := make([]*models.Project, len(projects.Projects))
	for i, p := range projects.Projects {
		saveProjects[i] = &models.Project{
			Name:     p.Name,
			RootPath: paths.Collapse(p.RootPath),
			Tags:     p.Tags,
			Enabled:  p.Enabled,
		}
	}

	data, err := json.MarshalIndent(saveProjects, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize projects: %w", err)
	}

	if err := os.WriteFile(s.GetProjectsPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write projects file: %w", err)
	}

	return nil
}

// LoadCache loads cached auto-detected projects
func (s *Storage) LoadCache() (*CachedProjects, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cachePath := filepath.Join(s.basePath, cacheFileName)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &CachedProjects{}, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache CachedProjects
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Expand paths and set kinds
	for _, p := range cache.Git {
		p.RootPath = paths.Expand(p.RootPath)
		p.Kind = models.KindGit
	}
	for _, p := range cache.SVN {
		p.RootPath = paths.Expand(p.RootPath)
		p.Kind = models.KindSVN
	}
	for _, p := range cache.Mercurial {
		p.RootPath = paths.Expand(p.RootPath)
		p.Kind = models.KindMercurial
	}
	for _, p := range cache.VSCode {
		p.RootPath = paths.Expand(p.RootPath)
		p.Kind = models.KindVSCode
	}
	for _, p := range cache.Any {
		p.RootPath = paths.Expand(p.RootPath)
		p.Kind = models.KindAny
	}

	return &cache, nil
}

// SaveCache saves cached auto-detected projects
func (s *Storage) SaveCache(cache *CachedProjects) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Collapse paths before saving
	saveCacheProjects := func(projects []*models.Project) []*models.Project {
		result := make([]*models.Project, len(projects))
		for i, p := range projects {
			result[i] = &models.Project{
				Name:     p.Name,
				RootPath: paths.Collapse(p.RootPath),
				Tags:     p.Tags,
				Enabled:  p.Enabled,
			}
		}
		return result
	}

	saveCache := &CachedProjects{
		Git:       saveCacheProjects(cache.Git),
		SVN:       saveCacheProjects(cache.SVN),
		Mercurial: saveCacheProjects(cache.Mercurial),
		VSCode:    saveCacheProjects(cache.VSCode),
		Any:       saveCacheProjects(cache.Any),
	}

	data, err := json.MarshalIndent(saveCache, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize cache: %w", err)
	}

	cachePath := filepath.Join(s.basePath, cacheFileName)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// ClearCache removes the cache file
func (s *Storage) ClearCache() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cachePath := filepath.Join(s.basePath, cacheFileName)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}

// LoadAllProjects loads all projects from both favorites and cache
func (s *Storage) LoadAllProjects() ([]*models.Project, error) {
	var allProjects []*models.Project

	// Load favorites
	projects, err := s.LoadProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}
	allProjects = append(allProjects, projects.Projects...)

	// Load cache (errors are non-fatal for cache)
	cache, _ := s.LoadCache()
	if cache != nil {
		allProjects = append(allProjects, cache.Git...)
		allProjects = append(allProjects, cache.SVN...)
		allProjects = append(allProjects, cache.Mercurial...)
		allProjects = append(allProjects, cache.VSCode...)
		allProjects = append(allProjects, cache.Any...)
	}

	return allProjects, nil
}
