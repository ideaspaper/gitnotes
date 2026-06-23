// Package models provides data structures for representing projects,
// project lists, and project kinds used throughout the projector application.
package models

import "strings"

// ProjectKind represents the type/source of a project
type ProjectKind string

const (
	KindFavorite  ProjectKind = "favorites"
	KindGit       ProjectKind = "git"
	KindSVN       ProjectKind = "svn"
	KindMercurial ProjectKind = "mercurial"
	KindVSCode    ProjectKind = "vscode"
	KindAny       ProjectKind = "any"
)

// Project represents a saved project
type Project struct {
	Name     string      `json:"name"`
	RootPath string      `json:"rootPath"`
	Tags     []string    `json:"tags"`
	Enabled  bool        `json:"enabled"`
	Kind     ProjectKind `json:"-"` // Internal use only, not persisted
}

// NewProject creates a new enabled project with the given name and path
func NewProject(name, rootPath string) *Project {
	return &Project{
		Name:     name,
		RootPath: rootPath,
		Tags:     []string{},
		Enabled:  true,
		Kind:     KindFavorite,
	}
}

// HasTag checks if a project has a specific tag
func (p *Project) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the project if not already present
func (p *Project) AddTag(tag string) {
	if !p.HasTag(tag) {
		p.Tags = append(p.Tags, tag)
	}
}

// RemoveTag removes a tag from the project
func (p *Project) RemoveTag(tag string) {
	for i, t := range p.Tags {
		if t == tag {
			p.Tags = append(p.Tags[:i], p.Tags[i+1:]...)
			return
		}
	}
}

// ProjectList represents a collection of projects
type ProjectList struct {
	Projects []*Project
	Kind     ProjectKind
}

// NewProjectList creates a new empty project list
func NewProjectList(kind ProjectKind) *ProjectList {
	return &ProjectList{
		Projects: []*Project{},
		Kind:     kind,
	}
}

// Add adds a project to the list
func (pl *ProjectList) Add(project *Project) {
	project.Kind = pl.Kind
	pl.Projects = append(pl.Projects, project)
}

// Remove removes a project by name (case-insensitive)
func (pl *ProjectList) Remove(name string) bool {
	for i, p := range pl.Projects {
		if strings.EqualFold(p.Name, name) {
			pl.Projects = append(pl.Projects[:i], pl.Projects[i+1:]...)
			return true
		}
	}
	return false
}

// FindByName finds a project by its name (case-insensitive)
func (pl *ProjectList) FindByName(name string) *Project {
	for _, p := range pl.Projects {
		if strings.EqualFold(p.Name, name) {
			return p
		}
	}
	return nil
}

// FindByPath finds a project by its root path
func (pl *ProjectList) FindByPath(path string) *Project {
	for _, p := range pl.Projects {
		if p.RootPath == path {
			return p
		}
	}
	return nil
}

// FilterByTag returns projects that have a specific tag
func (pl *ProjectList) FilterByTag(tag string) []*Project {
	var result []*Project
	for _, p := range pl.Projects {
		if p.HasTag(tag) {
			result = append(result, p)
		}
	}
	return result
}

// FilterEnabled returns only enabled projects
func (pl *ProjectList) FilterEnabled() []*Project {
	var result []*Project
	for _, p := range pl.Projects {
		if p.Enabled {
			result = append(result, p)
		}
	}
	return result
}

// Count returns the number of projects
func (pl *ProjectList) Count() int {
	return len(pl.Projects)
}
