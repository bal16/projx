package tui

import (
	"fmt"
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
	saved, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = saved.(Model)
	if m.edit.Active() {
		t.Fatal("editing not closed after save")
	}
	if cmd == nil {
		t.Fatal("expected background refresh command after save")
	}
	msg := cmd()
	if pmsg, ok := msg.(projectsMsg); !ok || len(pmsg.projects) == 0 {
		t.Fatalf("expected projectsMsg from save cmd, got %#v", msg)
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

	// Clean repo preview
	m.projects[0].GitStaged = 0
	m.projects[0].GitModified = 0
	m.projects[0].GitUntracked = 0
	m.projects[0].GitDirty = 0
	m.filtered = m.projects
	cleanOut := m.View()
	if !strings.Contains(cleanOut, "feat/auth (clean)") {
		t.Fatalf("expected clean repo indicator in view: %q", cleanOut)
	}

	// Conflicted repo preview
	m.projects[0].GitConflicted = 1
	m.projects[0].GitDirty = 1
	m.filtered = m.projects
	conflictOut := m.View()
	if !strings.Contains(conflictOut, "U:1") {
		t.Fatalf("expected conflict indicator in view: %q", conflictOut)
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

func TestRefreshKeyPreservesPreview(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	dir := t.TempDir()
	m := Model{
		projects: []model.Project{
			{Name: "my-project", Path: dir, GitBranch: "main"},
		},
		width: 120, height: 24, mode: "normal",
	}
	m.filtered = m.projects
	m.selected = 0

	// Pressing R
	up, cmd := m.handleNormalMode("R")
	m = up.(Model)
	if cmd == nil {
		t.Fatal("expected command from R key")
	}
	if m.status != "refreshing status..." {
		t.Fatalf("unexpected status: %q", m.status)
	}

	// Simulating commandResultMsg for refresh
	up, cmd = m.Update(commandResultMsg{cmd: "refresh", output: "my-project  main\n"})
	m = up.(Model)
	if cmd == nil {
		t.Fatal("expected loadProjectsCmd after refresh")
	}
	if m.lastOutput != "" {
		t.Fatalf("expected lastOutput to be empty on refresh, got %q", m.lastOutput)
	}
	if m.status != "refreshed status" {
		t.Fatalf("expected status 'refreshed status', got %q", m.status)
	}

	// Verify preview still shows the project, not raw CLI output
	out := m.View()
	if !strings.Contains(out, "my-project") {
		t.Fatalf("expected preview to show project name: %q", out)
	}

	// Simulating projectsMsg arrives
	up, _ = m.Update(projectsMsg{projects: m.projects})
	m = up.(Model)
	if !strings.Contains(m.status, "refreshed status (1 projects)") {
		t.Fatalf("expected updated status with project count, got %q", m.status)
	}
}

func TestViewScrollsWhenSelectedIsOffscreen(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	var projs []model.Project
	for i := 0; i < 25; i++ {
		projs = append(projs, model.Project{
			Name: fmt.Sprintf("proj-%02d", i),
			Path: fmt.Sprintf("/path/proj-%02d", i),
		})
	}
	m := Model{
		projects: projs,
		filtered: projs,
		width:    80,
		height:   16, // maxLines = 10
		mode:     "normal",
	}

	// At start: selected = 0. proj-00 visible, proj-20 NOT visible.
	out0 := m.View()
	if !strings.Contains(out0, "proj-00") {
		t.Fatalf("expected proj-00 to be visible initially: %q", out0)
	}
	if strings.Contains(out0, "proj-20") {
		t.Fatalf("expected proj-20 to be off-screen initially: %q", out0)
	}

	// Set selected to 20: view must scroll so proj-20 is visible and proj-00 scrolls off
	m.selected = 20
	m.ensureVisible()
	out20 := m.View()
	if !strings.Contains(out20, "proj-20") {
		t.Fatalf("expected proj-20 to be visible after scrolling: %q", out20)
	}
	if strings.Contains(out20, "proj-00") {
		t.Fatalf("expected proj-00 to have scrolled out of viewport: %q", out20)
	}
	if !strings.Contains(out20, "> ") {
		t.Fatalf("expected selected pointer in viewport: %q", out20)
	}

	// Jump to bottom via G
	up, _ := m.handleNormalMode("G")
	m = up.(Model)
	outBottom := m.View()
	if !strings.Contains(outBottom, "proj-24") {
		t.Fatalf("expected proj-24 to be visible at bottom: %q", outBottom)
	}

	// Jump to top via gg
	up, _ = m.handleNormalMode("g")
	m = up.(Model)
	up, _ = m.handleNormalMode("g")
	m = up.(Model)
	outTop := m.View()
	if !strings.Contains(outTop, "proj-00") {
		t.Fatalf("expected proj-00 to be visible at top after gg: %q", outTop)
	}
}
