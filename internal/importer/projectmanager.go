package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bal16/projx/internal/model"
	"github.com/bal16/projx/internal/platform"
)

// ProjectManagerImporter reads projects from Alessandro Fragnani's
// "Project Manager" VS Code extension (projects.json).
type ProjectManagerImporter struct {
	Variant string // code, codium, cursor
}

func (p ProjectManagerImporter) Name() string { return "projectmanager-" + p.Variant }

func (p ProjectManagerImporter) Available() bool {
	_, err := os.Stat(projectManagerPath(p.Variant))
	return err == nil
}

func (p ProjectManagerImporter) Import() ([]model.Project, error) {
	path := projectManagerPath(p.Variant)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read projects.json: %w", err)
	}
	var raw []struct {
		Name     string   `json:"name"`
		RootPath string   `json:"rootPath"`
		Paths    []string `json:"paths"`
		Tags     []string `json:"tags"`
		Enabled  bool     `json:"enabled"`
		Profile  string   `json:"profile"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse projects.json: %w", err)
	}

	var out []model.Project
	for _, r := range raw {
		if !r.Enabled {
			continue
		}
		if r.RootPath == "" {
			continue
		}
		name := r.Name
		if name == "" {
			name = filepath.Base(r.RootPath)
		}
		tags := r.Tags
		if tags == nil {
			tags = []string{}
		}
		out = append(out, model.Project{
			Name:    name,
			Path:    r.RootPath,
			Profile: r.Profile,
			Tags:    tags,
		})
		// Multi-root: additional paths
		for _, extra := range r.Paths {
			if extra == "" || extra == r.RootPath {
				continue
			}
			out = append(out, model.Project{
				Name:    name + " (extra)",
				Path:    extra,
				Profile: r.Profile,
				Tags:    append([]string{}, tags...),
			})
		}
	}
	return out, nil
}

func projectManagerPath(variant string) string {
	if v := os.Getenv("PROJX_PM_JSON"); v != "" {
		return v
	}
	pp, err := platform.VSCodeProjectManagerJSON(variant)
	if err != nil {
		return ""
	}
	return pp
}
