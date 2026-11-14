package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bal16/projx/internal/db"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultEditor != "zed" {
		t.Fatalf("default editor: %s", cfg.DefaultEditor)
	}
	if cfg.ScanMaxDepth != 2 {
		t.Fatalf("max depth: %d", cfg.ScanMaxDepth)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	d, err := db.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	repo := db.NewRepository(d)
	cfg := DefaultConfig()
	cfg.DefaultEditor = "code"
	cfg.ScanRoots = []string{"/tmp/a", "/tmp/b"}
	if err := Save(cfg, repo); err != nil {
		t.Fatal(err)
	}
	cfg2, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.DefaultEditor != "code" {
		t.Fatalf("editor: %s", cfg2.DefaultEditor)
	}
	if len(cfg2.ScanRoots) != 2 {
		t.Fatalf("roots: %v", cfg2.ScanRoots)
	}
	// file exists
	if _, err := os.Stat(filepath.Join(dir, "projx", "config.json")); err != nil {
		t.Fatal(err)
	}
}
