// Package util provides small shared helpers.
package util

import "path/filepath"

// BaseName returns the last path element of p.
func BaseName(p string) string {
	b := filepath.Base(p)
	if b == "." || b == "/" || b == "\\" {
		return p
	}
	return b
}
