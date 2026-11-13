// Package discovery scans directories for project markers.
package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scanner finds project directories by walking roots up to maxDepth.
type Scanner struct {
	Markers  []string
	MaxDepth int
}

// Found represents a discovered project directory.
type Found struct {
	Path string
	Name string
}

// Scan walks roots and returns discovered project directories.
func (s *Scanner) Scan(roots []string) ([]Found, error) {
	var out []Found
	seen := map[string]bool{}
	for _, root := range roots {
		expanded, err := expandHome(root)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(expanded); err != nil {
			continue // skip missing roots
		}
		if err := filepath.Walk(expanded, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip unreadable
			}
			if !info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(expanded, path)
			depth := 0
			if rel != "." {
				depth = strings.Count(rel, string(os.PathSeparator)) + 1
			}
			if depth > s.MaxDepth {
				return filepath.SkipDir
			}
			if path == expanded {
				return nil
			}
			// skip hidden dirs and common noise
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" || base == "target" || base == "dist" {
				return filepath.SkipDir
			}
			if s.hasMarker(path) {
				if !seen[path] {
					seen[path] = true
					out = append(out, Found{Path: path, Name: base})
				}
				// don't descend into a project (its subdirs are part of it)
				return filepath.SkipDir
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk %s: %w", expanded, err)
		}
	}
	return out, nil
}

func (s *Scanner) hasMarker(dir string) bool {
	for _, m := range s.Markers {
		if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
			return true
		}
	}
	return false
}

func expandHome(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	if p == "~" {
		return os.UserHomeDir()
	}
	return p, nil
}
