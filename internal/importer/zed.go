package importer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/bal16/projx/internal/model"
	"github.com/bal16/projx/internal/platform"
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

		// paths may contain multiple roots (newline or JSON array). Split.
		var roots []string
		trimmed := strings.TrimSpace(paths)
		if strings.HasPrefix(trimmed, "[") {
			var arr []string
			if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
				roots = arr
			}
		}
		if len(roots) == 0 {
			for _, line := range strings.Split(trimmed, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					roots = append(roots, line)
				}
			}
		}
		if len(roots) == 0 {
			roots = []string{trimmed}
		}

		for _, rp := range roots {
			rp = strings.TrimSpace(rp)
			if rp == "" {
				continue
			}
			name := filepath.Base(rp)
			pp := rp
			r := remote
			if r != "" {
				name += " (remote)"
				pp = "ssh://" + r + rp
			}
			out = append(out, model.Project{
				Name:       name,
				Path:       pp,
				Editor:     "zed",
				Tags:       []string{},
				Remote:     r,
				LastOpened: parsed,
			})
		}
	}
	return out, rows.Err()
}

func zedDBPath() string {
	if v := os.Getenv("PROJX_ZED_DB"); v != "" {
		return v
	}
	for _, ch := range []string{"stable", "preview"} {
		pp, err := platform.ZedDBPath(ch)
		if err != nil {
			continue
		}
		if _, err := os.Stat(pp); err == nil {
			return pp
		}
	}
	pp, _ := platform.ZedDBPath("stable")
	return pp
}
