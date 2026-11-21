package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bal16/projx/internal/model"
)

func TestRefreshKeyPreservesPreview(t *testing.T) {
	dir := t.TempDir()
	m := Model{
		projects: []model.Project{
			{Name: "my-project", Path: dir, GitBranch: "main"},
		},
		width: 120, height: 24, mode: "normal",
	}
	m.filtered = m.projects
	m.selected = 0

	// Pressing R triggers refresh
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

	// Verify preview still displays the project, not raw CLI output
	out := m.View()
	if !strings.Contains(out, "my-project") {
		t.Fatalf("expected preview to show project name: %q", out)
	}

	// Simulating projectsMsg arrival updates status
	up, _ = m.Update(projectsMsg{projects: m.projects})
	m = up.(Model)
	if !strings.Contains(m.status, "refreshed status (1 projects)") {
		t.Fatalf("expected updated status with project count, got %q", m.status)
	}
}

func TestQuickChooseEditorFlow(t *testing.T) {
	repo, p := setupTestRepo(t)

	m := New(repo)
	m.projects = []model.Project{*p}
	m.filtered = []model.Project{*p}
	m.width = 80
	m.height = 24
	m.mode = "normal"
	m.selected = 0

	// Press E in normal mode
	up, _ := m.handleNormalMode("E")
	m = up.(Model)
	if m.mode != "open_editor" {
		t.Fatalf("expected mode open_editor, got %q", m.mode)
	}

	// Press Tab to cycle completion: zed -> code
	up, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = up.(Model)
	if m.input.Value() != "zed" {
		t.Fatalf("expected completion 'zed', got %q", m.input.Value())
	}
	up, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = up.(Model)
	if m.input.Value() != "code" {
		t.Fatalf("expected completion 'code', got %q", m.input.Value())
	}

	// Esc cancels back to normal mode without opening
	up, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = up.(Model)
	if m.mode != "normal" || m.Opened() {
		t.Fatalf("expected canceled to normal mode, got mode=%q opened=%v", m.mode, m.Opened())
	}

	// Press E again, type custom editor, and Enter
	up, _ = m.handleNormalMode("E")
	m = up.(Model)
	m.input.SetValue("nvim")
	up, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = up.(Model)
	if !m.Opened() {
		t.Fatal("expected Opened() to be true")
	}
	if m.OpenPath() != p.Path {
		t.Fatalf("unexpected open path: %q, want %q", m.OpenPath(), p.Path)
	}
	if m.OpenEditor() != "nvim" {
		t.Fatalf("expected open editor 'nvim', got %q", m.OpenEditor())
	}
}
