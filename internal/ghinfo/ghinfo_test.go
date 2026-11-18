package ghinfo

import "testing"

func TestParseGitHubRemote(t *testing.T) {
	tests := []struct {
		url  string
		want string
		ok   bool
	}{
		{"https://github.com/owner/repo.git", "owner/repo", true},
		{"https://github.com/owner/repo", "owner/repo", true},
		{"git@github.com:owner/repo.git", "owner/repo", true},
		{"ssh://git@github.com/owner/repo.git", "owner/repo", true},
		{"ssh://git@github.com/owner/repo", "owner/repo", true},
		{"https://gitlab.com/owner/repo.git", "", false},
		{"", "", false},
		{"not-a-url", "", false},
	}
	for _, tt := range tests {
		got, ok := ParseGitHubRemote(tt.url)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("ParseGitHubRemote(%q) = (%q,%v), want (%q,%v)", tt.url, got, ok, tt.want, tt.ok)
		}
	}
}
