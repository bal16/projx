// Package importer seeds projx from existing editor state (one-time migration).
package importer

import (
	"fmt"

	"github.com/bal16/projx/internal/model"
)

// Importer seeds projects from an external editor.
type Importer interface {
	Name() string
	Available() bool
	Import() ([]model.Project, error)
}

// All returns all known importers.
func All() []Importer {
	return []Importer{
		&ZedImporter{},
		&VSCodeImporter{Variant: "code"},
		&VSCodeImporter{Variant: "codium"},
		&VSCodeImporter{Variant: "cursor"},
		&ProjectManagerImporter{Variant: "code"},
		&ProjectManagerImporter{Variant: "codium"},
		&ProjectManagerImporter{Variant: "cursor"},
	}
}

// ByName returns an importer by name.
func ByName(name string) (Importer, error) {
	// Aliases for convenience.
	switch name {
	case "projectmanager", "pm", "pm-ext":
		name = "projectmanager-code"
	case "projectmanager-codium":
		name = "projectmanager-codium"
	case "projectmanager-cursor":
		name = "projectmanager-cursor"
	}
	for _, imp := range All() {
		if imp.Name() == name {
			return imp, nil
		}
	}
	return nil, fmt.Errorf("unknown importer: %s", name)
}
