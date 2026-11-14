package gitinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInspectNonRepo(t *testing.T) {
	dir := t.TempDir()
	info := Inspect(dir)
	if info.IsRepo {
		t.Fatal("expected not a repo")
	}
	if info.Branch != "" {
		t.Fatalf("expected empty branch, got %s", info.Branch)
	}
}

func TestInspectRepo(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	// init git repo
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}
	// configure user for commit
	exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "t").Run()
	// create a file and commit
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644)
	exec.Command("git", "-C", dir, "add", "a.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-q", "-m", "init").Run()

	info := Inspect(dir)
	if !info.IsRepo {
		t.Fatal("expected repo")
	}
	if info.Branch == "" {
		t.Fatal("expected branch")
	}
	// dirty after modifying
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0o644)
	info2 := Inspect(dir)
	if info2.Dirty == 0 {
		t.Fatal("expected dirty count > 0")
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Fatal("expected false")
	}
}
