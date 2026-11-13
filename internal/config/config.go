// Package config handles projx configuration (scan roots, default editor, etc.).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bal16/projx/internal/db"
)

// Config is the in-memory projx configuration.
type Config struct {
	ScanRoots      []string          `json:"scan_roots"`
	ScanMaxDepth   int               `json:"scan_max_depth"`
	ScanMarkers    []string          `json:"scan_markers"`
	DefaultEditor  string            `json:"default_editor"`
	EditorCommands map[string]string `json:"editor_commands"` // editor name -> binary
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ScanRoots:     []string{},
		ScanMaxDepth:  2,
		ScanMarkers:   []string{".git", "Cargo.toml", "package.json", "go.mod", "pyproject.toml", "pom.xml", "build.gradle"},
		DefaultEditor: "zed",
		EditorCommands: map[string]string{
			"zed":  "zed",
			"code": "code",
			"nvim": "nvim",
			"vim":  "vim",
		},
	}
}

// Load reads config from disk, falling back to defaults + DB settings.
func Load(repo *db.Repository) (*Config, error) {
	cfg := DefaultConfig()

	// Try file first.
	dir, err := db.ConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// Override with DB settings if present.
	if repo != nil {
		if v, _ := repo.GetSetting("scan_roots"); v != "" {
			var roots []string
			if json.Unmarshal([]byte(v), &roots) == nil {
				cfg.ScanRoots = roots
			}
		}
		if v, _ := repo.GetSetting("default_editor"); v != "" {
			cfg.DefaultEditor = v
		}
		if v, _ := repo.GetSetting("scan_max_depth"); v != "" {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
				cfg.ScanMaxDepth = n
			}
		}
	}
	if cfg.ScanMaxDepth <= 0 {
		cfg.ScanMaxDepth = 2
	}
	return cfg, nil
}

// Save persists config to disk and DB.
func Save(cfg *Config, repo *db.Repository) error {
	dir, err := db.ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644); err != nil {
		return err
	}
	if repo != nil {
		if b, _ := json.Marshal(cfg.ScanRoots); true {
			_ = repo.SetSetting("scan_roots", string(b))
		}
		_ = repo.SetSetting("default_editor", cfg.DefaultEditor)
		_ = repo.SetSetting("scan_max_depth", fmt.Sprintf("%d", cfg.ScanMaxDepth))
	}
	return nil
}
