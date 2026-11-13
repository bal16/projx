package importer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/bal16/projx/internal/model"
)

// VSCodeImporter reads recent workspaces from a VS Code variant.
type VSCodeImporter struct {
	Variant string // code, codium, cursor, windsurf
}

func (v VSCodeImporter) Name() string { return v.Variant }

func (v VSCodeImporter) Available() bool {
	_, err := os.Stat(vscodeDBPath(v.Variant))
	return err == nil
}

func (v VSCodeImporter) Import() ([]model.Project, error) {
	path := vscodeDBPath(v.Variant)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open vscode db: %w", err)
	}
	defer db.Close()

	// Try known keys that may contain recently opened paths.
	keys := []string{
		"history.recentlyOpenedPathsList",
		"recentlyOpenedPathsList",
	}
	var raw string
	for _, k := range keys {
		err := db.QueryRow("SELECT value FROM ItemTable WHERE key = ?", k).Scan(&raw)
		if err == nil && raw != "" {
			break
		}
	}
	if raw == "" {
		return nil, nil // nothing to import; not an error
	}

	// Shape: {"entries":[{"folderUri":"file:///...","workspace":{"configPath":"..."},...}]}
	var payload struct {
		Entries []struct {
			FolderURI string `json:"folderUri"`
			FileURI   string `json:"fileUri"`
			Workspace struct {
				ConfigPath string `json:"configPath"`
			} `json:"workspace"`
			Label string `json:"label"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("parse recentlyOpened: %w", err)
	}

	editor := v.Variant
	if editor == "code" {
		editor = "vscode"
	}
	var out []model.Project
	for _, e := range payload.Entries {
		p := ""
		switch {
		case e.FolderURI != "":
			p = uriToPath(e.FolderURI)
		case e.Workspace.ConfigPath != "":
			p = uriToPath(e.Workspace.ConfigPath)
		case e.FileURI != "":
			p = uriToPath(e.FileURI)
		}
		if p == "" {
			continue
		}
		name := e.Label
		if name == "" {
			name = filepath.Base(p)
		}
		out = append(out, model.Project{
			Name:   name,
			Path:   p,
			Editor: editor,
			Tags:   []string{},
		})
	}
	return out, nil
}

func vscodeDBPath(variant string) string {
	if v := os.Getenv("PROJX_VSCODE_DB"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	dirs := map[string]string{
		"code":     filepath.Join(home, ".config", "Code", "User", "globalStorage", "state.vscdb"),
		"codium":   filepath.Join(home, ".config", "VSCodium", "User", "globalStorage", "state.vscdb"),
		"cursor":   filepath.Join(home, ".config", "Cursor", "User", "globalStorage", "state.vscdb"),
		"windsurf": filepath.Join(home, ".config", "Windsurf", "User", "globalStorage", "state.vscdb"),
	}
	if p, ok := dirs[variant]; ok {
		return p
	}
	return dirs["code"]
}

func uriToPath(u string) string {
	u = strings.TrimSpace(u)
	if !strings.HasPrefix(u, "file://") {
		return u
	}
	p := strings.TrimPrefix(u, "file://")
	// handle file:///path
	p = strings.TrimPrefix(p, "/")
	// On linux, path should start with /
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// unescape %20 etc. (minimal)
	return p
}
