package db

import "fmt"

const schemaVersion = 2

const createSchema = `
CREATE TABLE IF NOT EXISTS projects (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    path        TEXT NOT NULL UNIQUE,
    editor      TEXT NOT NULL DEFAULT '',
    profile     TEXT NOT NULL DEFAULT '',
    tags        TEXT NOT NULL DEFAULT '[]',
    group_name  TEXT NOT NULL DEFAULT '',
    favorite    INTEGER NOT NULL DEFAULT 0,
    last_opened INTEGER NOT NULL DEFAULT 0,
    git_branch  TEXT NOT NULL DEFAULT '',
    git_dirty   INTEGER NOT NULL DEFAULT 0,
    remote      TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sets (
    name       TEXT PRIMARY KEY,
    paths      TEXT NOT NULL,   -- JSON array of paths
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER NOT NULL
);
`

func (d *DB) migrate() error {
	if _, err := d.Exec(createSchema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	var v int
	err := d.QueryRow("SELECT COALESCE((SELECT version FROM schema_version LIMIT 1), 0)").Scan(&v)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	// Run incremental migrations for older databases.
	if v < 2 {
		// Add profile column if missing (idempotent; ignore "duplicate column" errors).
		_, _ = d.Exec("ALTER TABLE projects ADD COLUMN profile TEXT NOT NULL DEFAULT ''")
	}
	if v == 0 {
		if _, err := d.Exec("INSERT INTO schema_version (version) VALUES (?)", schemaVersion); err != nil {
			return fmt.Errorf("set schema version: %w", err)
		}
	} else if v < schemaVersion {
		if _, err := d.Exec("UPDATE schema_version SET version = ?", schemaVersion); err != nil {
			return fmt.Errorf("update schema version: %w", err)
		}
	}
	return nil
}
