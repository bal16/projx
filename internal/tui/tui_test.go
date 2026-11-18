package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/bal16/projx/internal/db"
	"github.com/bal16/projx/internal/model"
)

func TestViewHighlightsSelectedRow(t *testing.T) {
	// lipgloss drops colors when stdout is not a TTY (as under go test);
	// force a color profile so the highlight ANSI is actually rendered.
	lipgloss.SetColorProfile(termenv.ANSI256)
	m := Model{
		projects: []model.Project{
			{Name: "alpha", Path: "/tmp/alpha"},
			{Name: "beta", Path: "/tmp/beta"},
		},
		width: 80, height: 24, mode: "normal",
	}
	m.filtered = m.projects
	m.selected = 1

	var sel, plain string
	for _, l := range strings.Split(m.View(), "\n") {
		left, _, _ := strings.Cut(l, "|")
		if strings.Contains(left, "beta") {
			sel = left
		}
		if strings.Contains(left, "alpha") {
			plain = left
		}
	}
	if sel == "" {
		t.Fatal("selected row not found in view")
	}
	if !strings.Contains(sel, "> ") {
		t.Errorf("selected row missing pointer: %q", sel)
	}
	if !strings.Contains(sel, "\x1b[") {
		t.Errorf("selected row lost ANSI highlight: %q", sel)
	}
	if plain == "" {
		t.Fatal("unselected row not found in view")
	}
	if strings.Contains(plain, "> ") {
		t.Errorf("unselected row has pointer: %q", plain)
	}
}

func TestEditProjectFlow(t *testing.T) {
	t.Setenv("PROJX_DATA_DIR", t.TempDir())
	d, err := db.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close() }()
	repo := db.NewRepository(d)
	p := &model.Project{Name: "alpha", Path: "/tmp/alpha", Tags: []string{}}
	if err := repo.Add(p); err != nil {
		t.Fatal(err)
	}

	m := Model{
		repo:   repo,
		width:  80,
		height: 24,
		mode:   "normal",
		edit:   NewEditForm(),
	}
	m.projects, _ = repo.List("", "", false)
	m.filtered = m.projects
	m.selected = 0

	up, _ := m.handleNormalMode("e")
	m = up.(Model)
	if !m.edit.Active() {
		t.Fatal("editing not entered")
	}
	if m.View() == "" || !strings.Contains(m.View(), "Editor:") {
		t.Fatal("edit form not rendered")
	}

	// change fields: editor + tags
	m.edit.inputs[1].SetValue("code")
	m.edit.inputs[3].SetValue("web, api")
	saved, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = saved.(Model)
	if m.edit.Active() {
		t.Fatal("editing not closed after save")
	}

	got, err := repo.Get("/tmp/alpha")
	if err != nil {
		t.Fatal(err)
	}
	if got.Editor != "code" {
		t.Errorf("editor = %q, want code", got.Editor)
	}
	if strings.Join(got.Tags, ",") != "web,api" {
		t.Errorf("tags = %v", got.Tags)
	}
	if got.Name != "alpha" {
		t.Errorf("name changed unexpectedly: %q", got.Name)
	}

	// esc must not save
	m.filtered, _ = repo.List("", "", false)
	m.selected = 0
	up, _ = m.handleNormalMode("e")
	m = up.(Model)
	m.edit.inputs[1].SetValue("vim")
	canceled, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = canceled.(Model)
	got, _ = repo.Get("/tmp/alpha")
	if got.Editor != "code" {
		t.Errorf("esc saved editor: %q, want code", got.Editor)
	}
}

func TestViewShowsProjectStatusInfo(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	enabled := true
	prs, issues := 2, 5
	dir := t.TempDir()
	m := Model{
		projects: []model.Project{
			{
				Name: "web", Path: dir, GitBranch: "feat/auth",
				GitStaged: 1, GitModified: 3, GitUntracked: 2, GitDeleted: 0, GitRenamed: 0, GitConflicted: 0, GitDirty: 6,
				GHEnabled: &enabled, GHOpenPRs: &prs, GHOpenIssues: &issues,
			},
		},
		width: 120, height: 24, mode: "normal",
	}
	m.filtered = m.projects
	m.selected = 0
	out := m.View()
	if !strings.Contains(out, "feat/auth") || !strings.Contains(out, "M:3") || !strings.Contains(out, "?:2") || !strings.Contains(out, "S:1") {
		t.Fatalf("git breakdown missing in view: %q", out)
	}
	if !strings.Contains(out, "Modified:") {
		t.Fatalf("modified time missing in view: %q", out)
	}
	if !strings.Contains(out, "PR:") || !strings.Contains(out, "2") || !strings.Contains(out, "Issues:") || !strings.Contains(out, "5") {
		t.Fatalf("PR/issues missing in view: %q", out)
	}
}

func TestViewGitHubPlaceholderAndDisabled(t *testing.T) {
	enabled := true
	dir := t.TempDir()
	m := Model{
		projects: []model.Project{
			{Name: "a", Path: dir, GitBranch: "main", GHEnabled: &enabled},
			{Name: "b", Path: dir, GitBranch: "main"},
		},
		width: 120, height: 24, mode: "normal",
	}
	m.filtered = m.projects
	m.selected = 0
	out := m.View()
	if !strings.Contains(out, "PR:") || !strings.Contains(out, "-") || !strings.Contains(out, "Issues:") {
		t.Fatalf("expected PR/issues placeholder: %q", out)
	}
	m.selected = 1
	out = m.View()
	if strings.Contains(out, "PR:") || strings.Contains(out, "Issues:") {
		t.Fatalf("did not expect PR/issues for disabled project: %q", out)
	}
}
