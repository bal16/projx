// Package gitinfo extracts git branch and dirty status for a project path.
package gitinfo

import (
	"os/exec"
	"strings"
)

// Info holds git metadata for a directory.
type Info struct {
	Branch     string
	Staged     int
	Modified   int
	Untracked  int
	Deleted    int
	Renamed    int
	Conflicted int
	Dirty      int // total number of changes (sum of breakdown buckets)
	IsRepo     bool
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

	// status breakdown
	out3, err3 := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err3 == nil {
		parsePorcelain(&info, string(out3))
	}
	return info
}

func parsePorcelain(info *Info, out string) {
	out = strings.TrimRight(out, "\n")
	if strings.TrimSpace(out) == "" {
		return
	}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		switch {
		case x == '?' && y == '?':
			info.Untracked++
			continue
		case x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D'):
			info.Conflicted++
			continue
		}
		switch x {
		case 'R', 'C':
			info.Renamed++
		case 'M', 'A', 'D':
			info.Staged++
		}
		switch y {
		case 'M':
			info.Modified++
		case 'D':
			info.Deleted++
		}
	}
	info.Dirty = info.Staged + info.Modified + info.Untracked + info.Deleted + info.Renamed + info.Conflicted
}

// IsGitRepo reports whether dir is inside a git repository.
func IsGitRepo(dir string) bool {
	_, err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Output()
	return err == nil
}
