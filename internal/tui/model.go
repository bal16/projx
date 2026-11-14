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
	width        int
	height       int
	status       string
	lastOutput   string
	showProfiles bool
	quitting     bool
	opened       bool
	openPath     string
	openNew      bool
	editing      bool
	editIdx      int
	editID       string
	editInputs   []textinput.Model
	cmdSugs      []string
	cmdSugSel    int
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

	m := Model{
		repo:       repo,
		input:      ti,
		mode:       "search",
		status:     "loading...",
		editInputs: make([]textinput.Model, 5),
	}
	for i, label := range []string{"Name:", "Editor:", "Profile:", "Tags:", "Group:"} {
		in := textinput.New()
		in.Prompt = label + " "
		in.CharLimit = 128
		m.editInputs[i] = in
	}
	return m
}

// Init loads projects.
func (m Model) Init() tea.Cmd {
	return loadProjectsCmd(m.repo)
}

// Opened reports whether the user selected a project to open.
func (m Model) Opened() bool { return m.opened }

// OpenPath returns the path selected for opening.
func (m Model) OpenPath() string { return m.openPath }

// OpenNew reports whether the selection should open in a new window.
func (m Model) OpenNew() bool { return m.openNew }

// Run starts the TUI and returns selected path + new-window flag.
func Run(repo *db.Repository) (string, bool, error) {
	m := New(repo)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", false, err
	}
	fm, ok := final.(Model)
	if !ok {
		return "", false, fmt.Errorf("unexpected model type")
	}
	if fm.Opened() {
		return fm.OpenPath(), fm.OpenNew(), nil
	}
	return "", false, nil
}
