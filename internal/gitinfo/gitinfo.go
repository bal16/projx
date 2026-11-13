// Package gitinfo extracts git branch and dirty status for a project path.
package gitinfo

import (
	"os/exec"
	"strings"
)

// Info holds git metadata for a directory.
type Info struct {
	Branch string
	Dirty  int // number of changed files
	IsRepo bool
}

// Inspect returns git info for the given directory.
func Inspect(dir string) Info {
	info := Info{}
	// branch
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return info
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		// detached; try symbolic
		out2, err2 := exec.Command("git", "-C", dir, "describe", "--tags", "--exact-match").Output()
		if err2 == nil {
			branch = strings.TrimSpace(string(out2))
		} else {
			branch = "detached"
		}
	}
	info.Branch = branch
	info.IsRepo = true

	// dirty count
	out3, err3 := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err3 == nil {
		lines := strings.Split(strings.TrimSpace(string(out3)), "\n")
		if len(lines) == 1 && lines[0] == "" {
			info.Dirty = 0
		} else {
			info.Dirty = len(lines)
		}
	}
	return info
}

// IsGitRepo reports whether dir is inside a git repository.
func IsGitRepo(dir string) bool {
	_, err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Output()
	return err == nil
}
