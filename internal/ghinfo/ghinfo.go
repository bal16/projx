// Package ghinfo detects GitHub repositories and fetches cached open PR/issue counts.
package ghinfo

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Counts holds open PR and issue counts for a GitHub repository.
type Counts struct {
	OpenPRs    int
	OpenIssues int
	FetchedAt  time.Time
}

var (
	httpsRe = regexp.MustCompile(`(?i)^https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)
	sshRe   = regexp.MustCompile(`(?i)^(?:ssh://)?git@github\.com[:/]([^/]+)/([^/]+?)(?:\.git)?/?$`)
)

// ParseGitHubRemote extracts "owner/repo" from a GitHub remote URL.
func ParseGitHubRemote(url string) (string, bool) {
	url = strings.TrimSpace(url)
	if m := httpsRe.FindStringSubmatch(url); m != nil {
		return m[1] + "/" + m[2], true
	}
	if m := sshRe.FindStringSubmatch(url); m != nil {
		return m[1] + "/" + m[2], true
	}
	return "", false
}

// DetectRepo returns the GitHub "owner/repo" for the repository at path, if any.
func DetectRepo(path string) (string, bool) {
	for _, remote := range []string{"origin"} {
		out, err := exec.Command("git", "-C", path, "remote", "get-url", remote).Output()
		if err != nil {
			continue
		}
		if repo, ok := ParseGitHubRemote(strings.TrimSpace(string(out))); ok {
			return repo, true
		}
	}
	// fallback: first remote
	out, err := exec.Command("git", "-C", path, "remote").Output()
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name := strings.TrimSpace(line)
		if name == "" || name == "origin" {
			continue
		}
		u, err := exec.Command("git", "-C", path, "remote", "get-url", name).Output()
		if err != nil {
			continue
		}
		if repo, ok := ParseGitHubRemote(strings.TrimSpace(string(u))); ok {
			return repo, true
		}
	}
	return "", false
}

// FetchOpenCounts fetches open PR and issue counts for the GitHub repo at path using the gh CLI.
func FetchOpenCounts(ctx context.Context, path string) (Counts, error) {
	repo, ok := DetectRepo(path)
	if !ok {
		return Counts{}, fmt.Errorf("not a GitHub repository")
	}
	prs, err := ghCount(ctx, path, "pr", repo)
	if err != nil {
		return Counts{}, err
	}
	issues, err := ghCount(ctx, path, "issue", repo)
	if err != nil {
		return Counts{}, err
	}
	return Counts{OpenPRs: prs, OpenIssues: issues, FetchedAt: time.Now()}, nil
}

func ghCount(ctx context.Context, path, kind, repo string) (int, error) {
	cmd := exec.CommandContext(ctx, "gh", kind, "list", "-R", repo, "--state", "open", "--limit", "1000", "--json", "number", "--jq", "length")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse %s count %q: %w", kind, s, err)
	}
	return n, nil
}
