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
	dir := initRepo(t)
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644)
	runGit(t, dir, "add", "a.txt")
	runGit(t, dir, "commit", "-q", "-m", "init")

	info := Inspect(dir)
	if !info.IsRepo {
		t.Fatal("expected repo")
	}
	if info.Branch == "" {
		t.Fatal("expected branch")
	}
	// dirty after modifying
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0o644)
	info2 := Inspect(dir)
	if info2.Dirty == 0 {
		t.Fatal("expected dirty count > 0")
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		if err := exec.Command("git", "-C", dir, "init", "-q").Run(); err != nil {
			t.Skip("git not available")
		}
	}
	runGit(t, dir, "config", "user.email", "t@t.com")
	runGit(t, dir, "config", "user.name", "t")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, out)
	}
}

func TestInspectBreakdown(t *testing.T) {
	dir := initRepo(t)
	// base commit
	_ = os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "del.txt"), []byte("bye"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "ren.txt"), []byte("rename me"), 0o644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")

	// staged new file
	_ = os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("s"), 0o644)
	runGit(t, dir, "add", "staged.txt")
	// unstaged modified
	_ = os.WriteFile(filepath.Join(dir, "base.txt"), []byte("changed"), 0o644)
	// unstaged deleted
	_ = os.Remove(filepath.Join(dir, "del.txt"))
	// untracked
	_ = os.WriteFile(filepath.Join(dir, "new.txt"), []byte("u"), 0o644)
	// renamed (staged)
	runGit(t, dir, "mv", "ren.txt", "renamed.txt")

	info := Inspect(dir)
	if !info.IsRepo {
		t.Fatal("expected repo")
	}
	if info.Staged < 1 {
		t.Fatalf("staged = %d, want >=1", info.Staged)
	}
	if info.Modified != 1 {
		t.Fatalf("modified = %d, want 1", info.Modified)
	}
	if info.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", info.Deleted)
	}
	if info.Untracked != 1 {
		t.Fatalf("untracked = %d, want 1", info.Untracked)
	}
	if info.Renamed != 1 {
		t.Fatalf("renamed = %d, want 1", info.Renamed)
	}
	wantDirty := info.Staged + info.Modified + info.Untracked + info.Deleted + info.Renamed + info.Conflicted
	if info.Dirty != wantDirty {
		t.Fatalf("dirty = %d, want %d", info.Dirty, wantDirty)
	}
}

func TestInspectConflict(t *testing.T) {
	dir := initRepo(t)
	_ = os.WriteFile(filepath.Join(dir, "c.txt"), []byte("base\n"), 0o644)
	runGit(t, dir, "add", "c.txt")
	runGit(t, dir, "commit", "-q", "-m", "base")
	// create branch and conflicting change
	runGit(t, dir, "checkout", "-b", "feature")
	_ = os.WriteFile(filepath.Join(dir, "c.txt"), []byte("feature\n"), 0o644)
	runGit(t, dir, "commit", "-qam", "feature")
	runGit(t, dir, "checkout", "main")
	_ = os.WriteFile(filepath.Join(dir, "c.txt"), []byte("main\n"), 0o644)
	runGit(t, dir, "commit", "-qam", "main")
	cmd := exec.Command("git", "-C", dir, "merge", "feature")
	_ = cmd.Run() // expect conflict
	info := Inspect(dir)
	if info.Conflicted < 1 {
		t.Fatalf("conflicted = %d, want >=1", info.Conflicted)
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Fatal("expected false")
	}
}
