package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/bal16/projx/internal/model"
)

// ZedImporter reads Zed's recent workspaces.
type ZedImporter struct{}

func (ZedImporter) Name() string { return "zed" }

func (ZedImporter) Available() bool {
	_, err := os.Stat(zedDBPath())
	return err == nil
}

func (ZedImporter) Import() ([]model.Project, error) {
	path := zedDBPath()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open zed db: %w", err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`SELECT paths, remote_connection_id, timestamp FROM workspaces ORDER BY timestamp DESC`)
	if err != nil {
		return nil, fmt.Errorf("query workspaces: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []model.Project
	for rows.Next() {
		var paths string
		var remoteID sql.NullInt64
		var ts string
		if err := rows.Scan(&paths, &remoteID, &ts); err != nil {
			continue
		}
		paths = strings.TrimSpace(paths)
		if paths == "" {
			continue
		}

		// Resolve remote host if applicable.
		remote := ""
		if remoteID.Valid {
			var host, user string
			var port sql.NullInt64
			_ = db.QueryRow("SELECT host, user, port FROM remote_connections WHERE id = ?", remoteID.Int64).Scan(&host, &user, &port)
			if host != "" {
				if user != "" {
					remote = user + "@" + host
				} else {
					remote = host
				}
				if port.Valid && port.Int64 != 0 && port.Int64 != 22 {
					remote += fmt.Sprintf(":%d", port.Int64)
				}
			}
		}

		parsed, _ := time.Parse("2006-01-02 15:04:05", ts)
		name := filepath.Base(paths)
		if remote != "" {
			name += " (remote)"
			paths = "ssh://" + remote + paths
		}

		out = append(out, model.Project{
			Name:       name,
			Path:       paths,
			Editor:     "zed",
			Tags:       []string{},
			Remote:     remote,
			LastOpened: parsed,
		})
	}
	return out, rows.Err()
}

func zedDBPath() string {
	// Allow override via env for testing.
	if v := os.Getenv("PROJX_ZED_DB"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "share", "zed", "db", "0-stable", "db.sqlite"),
		filepath.Join(home, ".local", "share", "zed", "db", "0-preview", "db.sqlite"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return candidates[0]
}
