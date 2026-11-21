package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bal16/projx/internal/model"
)

func TestEditProjectFlow(t *testing.T) {
	repo, p := setupTestRepo(t)

	m := Model{
		repo:   repo,
		width:  80,
		height: 24,
		mode:   "normal",
		edit:   NewEditForm(),
	}
	m.projects = []model.Project{*p}
	m.filtered = m.projects
	m.selected = 0

	// Enter edit mode
	up, _ := m.handleNormalMode("e")
	m = up.(Model)
	if !m.edit.Active() {
		t.Fatal("editing not entered")
	}
	if m.View() == "" || !strings.Contains(m.View(), "Editor:") {
		t.Fatal("edit form not rendered")
	}

	// Change fields: editor + tags
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

	got, err := repo.Get(p.ID)
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

	// Esc must discard unsaved changes
	m.filtered = []model.Project{*got}
	m.selected = 0
	up, _ = m.handleNormalMode("e")
	m = up.(Model)
	m.edit.inputs[1].SetValue("vim")
	canceled, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = canceled.(Model)
	got, _ = repo.Get(p.ID)
	if got.Editor != "code" {
		t.Errorf("esc saved editor: %q, want code", got.Editor)
	}
}
