package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bal16/projx/internal/model"
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

	editor := "code"
	switch p.Variant {
	case "codium":
		editor = "codium"
	case "cursor":
		editor = "cursor"
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
			Editor:  editor,
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
				Editor:  editor,
				Profile: r.Profile,
				Tags:    append([]string{}, tags...),
			})
		}
	}
	return out, nil
}

func projectManagerPath(variant string) string {
	home, _ := os.UserHomeDir()
	dirs := map[string]string{
		"code":   filepath.Join(home, ".config", "Code", "User", "globalStorage", "alefragnani.project-manager", "projects.json"),
		"codium": filepath.Join(home, ".config", "VSCodium", "User", "globalStorage", "alefragnani.project-manager", "projects.json"),
		"cursor": filepath.Join(home, ".config", "Cursor", "User", "globalStorage", "alefragnani.project-manager", "projects.json"),
	}
	if p, ok := dirs[variant]; ok {
		return p
	}
	return dirs["code"]
}
