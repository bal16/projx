// Package model defines the core domain types for projx.
package model

import "time"

// Project represents a tracked project/workspace.
type Project struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Editor     string    `json:"editor,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Group      string    `json:"group,omitempty"`
	Favorite   bool      `json:"favorite"`
	LastOpened time.Time `json:"last_opened,omitempty"`
	GitBranch  string    `json:"git_branch,omitempty"`
	GitDirty   int       `json:"git_dirty,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Remote     string    `json:"remote,omitempty"` // ssh host if remote
}

// HasTag reports whether the project carries the given tag.
func (p *Project) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag if not already present.
func (p *Project) AddTag(tag string) {
	if !p.HasTag(tag) {
		p.Tags = append(p.Tags, tag)
	}
}

// RemoveTag removes a tag if present.
func (p *Project) RemoveTag(tag string) {
	out := p.Tags[:0]
	for _, t := range p.Tags {
		if t != tag {
			out = append(out, t)
		}
	}
	p.Tags = out
}
