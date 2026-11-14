package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

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
