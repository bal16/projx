//go:build tools

// tools.go pins tool dependencies for this project.
// Run `go mod tidy` to populate go.sum, then `go install` the ones you need.
package tools

import (
	_ "golang.org/x/tools/cmd/goimports"
)
