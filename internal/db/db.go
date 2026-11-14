// Package db manages the SQLite database for projx.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps a *sql.DB with projx-specific helpers.
type DB struct {
	*sql.DB
	path string
}

// Open opens (or creates) the projx database at the default location.
func Open() (*DB, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dir, "projects.db")
	// Set pragmas via DSN so they apply to every pooled connection before any
	// query runs; busy_timeout must be in effect from the start, otherwise
	// concurrent processes (launcher + open) fail with SQLITE_BUSY.
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db := &DB{DB: sqlDB, path: path}
	if err := db.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// Path returns the database file path.
func (d *DB) Path() string { return d.path }

// dataDir returns the projx data directory.
// Resolution order: PROJX_DATA_DIR > PROJX_HOME/data > XDG_DATA_HOME/projx > ~/.local/share/projx
func dataDir() (string, error) {
	if d := os.Getenv("PROJX_DATA_DIR"); d != "" {
		return d, nil
	}
	if home := os.Getenv("PROJX_HOME"); home != "" {
		return filepath.Join(home, "data"), nil
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "projx"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".local", "share", "projx"), nil
}

// ConfigDir returns the projx config directory.
// Resolution order: PROJX_CONFIG_DIR > PROJX_HOME/config > XDG_CONFIG_HOME/projx > ~/.config/projx
func ConfigDir() (string, error) {
	if d := os.Getenv("PROJX_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if home := os.Getenv("PROJX_HOME"); home != "" {
		return filepath.Join(home, "config"), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "projx"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".config", "projx"), nil
}

// CacheDir returns the projx cache directory (for future remote metadata cache).
// Resolution order: PROJX_CACHE_DIR > PROJX_HOME/cache > XDG_CACHE_HOME/projx > ~/.cache/projx
func CacheDir() (string, error) {
	if d := os.Getenv("PROJX_CACHE_DIR"); d != "" {
		return d, nil
	}
	if home := os.Getenv("PROJX_HOME"); home != "" {
		return filepath.Join(home, "cache"), nil
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "projx"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".cache", "projx"), nil
}
