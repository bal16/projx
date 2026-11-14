package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScannerFindsGitRepo(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "myrepo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// noise dir should be skipped
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "foo", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sc := &Scanner{Markers: []string{".git"}, MaxDepth: 2}
	found, err := sc.Scan([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "myrepo" {
		t.Fatalf("found: %+v", found)
	}
}

func TestExpandHome(t *testing.T) {
	// basic: no panic
	got, err := expandHome("~/foo")
	if err != nil || got == "~/foo" {
		t.Fatalf("got %s %v", got, err)
	}
}
