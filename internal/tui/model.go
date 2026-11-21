package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bal16/projx/internal/db"
	"github.com/bal16/projx/internal/model"
)

// Model is the Bubble Tea model for the project picker.
type Model struct {
	repo         *db.Repository
	projects     []model.Project
	filtered     []model.Project
	input        textinput.Model
	mode         string // "search", "normal", or "command"
	pendingKey   string
	selected     int
	scrollOffset int
	width        int
	height       int
	status       string
	lastOutput   string
	showProfiles bool
	quitting     bool
	opened       bool
	openPath     string
	openNew      bool
	openEditor   string
	edit         EditForm
	cmd          CommandPalette
}

type projectsMsg struct{ projects []model.Project }
type errMsg struct{ err error }
type commandResultMsg struct {
	cmd    string
	output string
	err    error
}

// New creates a new TUI model.
func New(repo *db.Repository) Model {
	ti := textinput.New()
	ti.Placeholder = "search projects..."
	ti.CharLimit = 64
	ti.Width = 40

	return Model{
		repo:   repo,
		input:  ti,
		mode:   "search",
		status: "loading...",
		edit:   NewEditForm(),
	}
}

// Init loads projects.
func (m Model) Init() tea.Cmd {
	return loadProjectsCmd(m.repo)
}

// OpenResult holds the user's selection when exiting the TUI.
type OpenResult struct {
	Opened    bool
	Path      string
	NewWindow bool
	Editor    string
}

// Opened reports whether the user selected a project to open.
func (m Model) Opened() bool { return m.opened }

// OpenPath returns the path selected for opening.
func (m Model) OpenPath() string { return m.openPath }

// OpenNew reports whether the selection should open in a new window.
func (m Model) OpenNew() bool { return m.openNew }

// OpenEditor reports any editor override specified when opening.
func (m Model) OpenEditor() string { return m.openEditor }

// Run starts the TUI and returns the selected project and options.
func Run(repo *db.Repository) (OpenResult, error) {
	m := New(repo)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return OpenResult{}, err
	}
	fm, ok := final.(Model)
	if !ok {
		return OpenResult{}, fmt.Errorf("unexpected model type")
	}
	return OpenResult{
		Opened:    fm.Opened(),
		Path:      fm.OpenPath(),
		NewWindow: fm.OpenNew(),
		Editor:    fm.OpenEditor(),
	}, nil
}

func (m Model) maxLines() int {
	maxLines := m.height - 6
	if maxLines < 5 {
		maxLines = 5
	}
	return maxLines
}

func (m *Model) ensureVisible() {
	maxLines := m.maxLines()
	if len(m.filtered) == 0 {
		m.selected = 0
		m.scrollOffset = 0
		return
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	}
	if m.selected >= m.scrollOffset+maxLines {
		m.scrollOffset = m.selected - maxLines + 1
	}
	if maxOffset := len(m.filtered) - maxLines; maxOffset >= 0 && m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}
