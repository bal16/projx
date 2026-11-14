package db

import (
	"path/filepath"
	"testing"

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
	t.Cleanup(func() { d.Close() })
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
