package tui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/bal16/projx/internal/db"
	"github.com/bal16/projx/internal/model"
)

func TestMain(m *testing.M) {
	// Force 256-color ANSI output so Lip Gloss renders highlights during tests.
	lipgloss.SetColorProfile(termenv.ANSI256)
	os.Exit(m.Run())
}

func setupTestRepo(t *testing.T) (*db.Repository, *model.Project) {
	t.Helper()
	t.Setenv("PROJX_DATA_DIR", t.TempDir())
	d, err := db.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	repo := db.NewRepository(d)
	p := &model.Project{Name: "alpha", Path: "/tmp/alpha", Tags: []string{}}
	if err := repo.Add(p); err != nil {
		t.Fatal(err)
	}
	return repo, p
}
