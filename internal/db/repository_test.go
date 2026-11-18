package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bal16/projx/internal/model"
)

func tempDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	// override data dir via env
	t.Setenv("XDG_DATA_HOME", dir)
	d, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestAddAndGet(t *testing.T) {
	d := tempDB(t)
	repo := NewRepository(d)
	p := &model.Project{Name: "foo", Path: "/tmp/foo", Editor: "zed", Tags: []string{"a"}}
	if err := repo.Add(p); err != nil {
		t.Fatal(err)
	}
	if p.ID == "" {
		t.Fatal("id not set")
	}
	got, err := repo.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "foo" || got.Path != "/tmp/foo" {
		t.Fatalf("mismatch: %+v", got)
	}
	if !got.HasTag("a") {
		t.Fatal("tag missing")
	}
}

func TestDuplicatePath(t *testing.T) {
	d := tempDB(t)
	repo := NewRepository(d)
	p1 := &model.Project{Name: "a", Path: "/tmp/x"}
	p2 := &model.Project{Name: "b", Path: "/tmp/x"}
	if err := repo.Add(p1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Add(p2); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestTagFilterExact(t *testing.T) {
	d := tempDB(t)
	repo := NewRepository(d)
	p1 := &model.Project{Name: "a", Path: "/tmp/a", Tags: []string{"Work"}}
	p2 := &model.Project{Name: "b", Path: "/tmp/b", Tags: []string{"Workshop"}}
	_ = repo.Add(p1)
	_ = repo.Add(p2)
	got, err := repo.List("Work", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "a" {
		t.Fatalf("expected only 'a', got %+v", got)
	}
}

func TestRemove(t *testing.T) {
	d := tempDB(t)
	repo := NewRepository(d)
	p := &model.Project{Name: "x", Path: "/tmp/x"}
	_ = repo.Add(p)
	if err := repo.Remove(p.Name); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(p.ID); err == nil {
		t.Fatal("expected not found")
	}
}

func TestSetAndListSets(t *testing.T) {
	d := tempDB(t)
	repo := NewRepository(d)
	if err := repo.AddSet("s1", []string{"/a", "/b"}); err != nil {
		t.Fatal(err)
	}
	sets, err := repo.ListSets()
	if err != nil || len(sets) != 1 {
		t.Fatalf("sets: %+v %v", sets, err)
	}
	got, err := repo.GetSet("s1")
	if err != nil || len(got.Paths) != 2 {
		t.Fatalf("getset: %+v %v", got, err)
	}
	if err := repo.RemoveSet("s1"); err != nil {
		t.Fatal(err)
	}
}

func TestConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "projx") {
		t.Fatalf("got %s", got)
	}
}

func TestPROJX_HOMEIsolation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROJX_HOME", dir)
	d, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close() }()
	if d.Path() != filepath.Join(dir, "data", "projects.db") {
		t.Fatalf("db path: %s", d.Path())
	}
	cfgDir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if cfgDir != filepath.Join(dir, "config") {
		t.Fatalf("config dir: %s", cfgDir)
	}
	cacheDir, err := CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if cacheDir != filepath.Join(dir, "cache") {
		t.Fatalf("cache dir: %s", cacheDir)
	}
}

func TestMigrateV3(t *testing.T) {
	d := tempDB(t)
	var v int
	if err := d.QueryRow("SELECT version FROM schema_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 3 {
		t.Fatalf("schema version = %d, want 3", v)
	}
	rows, err := d.Query("PRAGMA table_info(projects)")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		cols[name] = true
	}
	for _, want := range []string{"git_staged", "git_modified", "git_untracked", "git_deleted", "git_renamed", "git_conflicted", "gh_enabled", "gh_open_prs", "gh_open_issues", "gh_fetched_at"} {
		if !cols[want] {
			t.Fatalf("missing column %s", want)
		}
	}
}

func TestGitBreakdownAndGHRoundTrip(t *testing.T) {
	d := tempDB(t)
	repo := NewRepository(d)
	p := &model.Project{Name: "app", Path: "/tmp/app-rt", Tags: []string{}}
	if err := repo.Add(p); err != nil {
		t.Fatal(err)
	}
	enabled := true
	prs, issues := 2, 5
	fetched := time.Unix(1700000000, 0)
	p.GitBranch = "main"
	p.GitStaged = 1
	p.GitModified = 2
	p.GitUntracked = 3
	p.GitDeleted = 4
	p.GitRenamed = 1
	p.GitConflicted = 0
	p.GitDirty = 11
	p.GHEnabled = &enabled
	p.GHOpenPRs = &prs
	p.GHOpenIssues = &issues
	p.GHFetchedAt = &fetched
	if err := repo.Update(p); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.GitStaged != 1 || got.GitModified != 2 || got.GitUntracked != 3 || got.GitDeleted != 4 || got.GitRenamed != 1 || got.GitConflicted != 0 || got.GitDirty != 11 {
		t.Fatalf("git breakdown mismatch: %+v", got)
	}
	if got.GHEnabled == nil || !*got.GHEnabled {
		t.Fatalf("gh_enabled = %v", got.GHEnabled)
	}
	if got.GHOpenPRs == nil || *got.GHOpenPRs != 2 || got.GHOpenIssues == nil || *got.GHOpenIssues != 5 {
		t.Fatalf("gh counts = %v %v", got.GHOpenPRs, got.GHOpenIssues)
	}
	if got.GHFetchedAt == nil || got.GHFetchedAt.Unix() != fetched.Unix() {
		t.Fatalf("gh_fetched_at = %v", got.GHFetchedAt)
	}

	// nullable/unset states
	p2 := &model.Project{Name: "lib", Path: "/tmp/lib-rt", Tags: []string{}}
	if err := repo.Add(p2); err != nil {
		t.Fatal(err)
	}
	p2.GHEnabled = nil
	p2.GHOpenPRs = nil
	p2.GHOpenIssues = nil
	p2.GHFetchedAt = nil
	if err := repo.Update(p2); err != nil {
		t.Fatal(err)
	}
	got2, err := repo.Get(p2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.GHEnabled != nil || got2.GHOpenPRs != nil || got2.GHOpenIssues != nil || got2.GHFetchedAt != nil {
		t.Fatalf("expected nil gh fields, got %+v", got2)
	}
}

func TestUpdateGitHubStatus(t *testing.T) {
	d := tempDB(t)
	repo := NewRepository(d)
	p := &model.Project{Name: "gh", Path: "/tmp/gh-rt", Tags: []string{}}
	if err := repo.Add(p); err != nil {
		t.Fatal(err)
	}
	fetched := time.Unix(1700001000, 0)
	if err := repo.UpdateGitHubStatus(p.ID, 3, 7, fetched); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.GHOpenPRs == nil || *got.GHOpenPRs != 3 || got.GHOpenIssues == nil || *got.GHOpenIssues != 7 {
		t.Fatalf("counts = %v %v", got.GHOpenPRs, got.GHOpenIssues)
	}
	if got.GHFetchedAt == nil || got.GHFetchedAt.Unix() != fetched.Unix() {
		t.Fatalf("fetched_at = %v", got.GHFetchedAt)
	}
}
