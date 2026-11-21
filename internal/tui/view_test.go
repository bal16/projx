package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bal16/projx/internal/model"
)

func TestViewHighlightsSelectedRow(t *testing.T) {
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

func TestViewShowsProjectStatusInfo(t *testing.T) {
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
	for _, want := range []string{"feat/auth", "M:3", "?:2", "S:1", "Modified:", "PR:", "2", "Issues:", "5"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected view to contain %q, got: %q", want, out)
		}
	}

	// Clean repo preview
	m.projects[0].GitStaged = 0
	m.projects[0].GitModified = 0
	m.projects[0].GitUntracked = 0
	m.projects[0].GitDirty = 0
	m.filtered = m.projects
	if cleanOut := m.View(); !strings.Contains(cleanOut, "feat/auth (clean)") {
		t.Fatalf("expected clean repo indicator in view: %q", cleanOut)
	}

	// Conflicted repo preview
	m.projects[0].GitConflicted = 1
	m.projects[0].GitDirty = 1
	m.filtered = m.projects
	if conflictOut := m.View(); !strings.Contains(conflictOut, "U:1") {
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

func TestViewScrollsWhenSelectedIsOffscreen(t *testing.T) {
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
	if outBottom := m.View(); !strings.Contains(outBottom, "proj-24") {
		t.Fatalf("expected proj-24 to be visible at bottom: %q", outBottom)
	}

	// Jump to top via gg
	up, _ = m.handleNormalMode("g")
	m = up.(Model)
	up, _ = m.handleNormalMode("g")
	m = up.(Model)
	if outTop := m.View(); !strings.Contains(outTop, "proj-00") {
		t.Fatalf("expected proj-00 to be visible at top after gg: %q", outTop)
	}
}
