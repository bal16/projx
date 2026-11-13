package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bal16/projx/internal/db"
	"github.com/bal16/projx/internal/model"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the Bubble Tea model for the project picker.
type Model struct {
	repo     *db.Repository
	projects []model.Project
	filtered []model.Project

	input    textinput.Model
	selected int
	width    int
	height   int

	status   string
	quitting bool

	// actions
	opened   bool
	openPath string
	openNew  bool
}

// New creates a new TUI model.
func New(repo *db.Repository) Model {
	ti := textinput.New()
	ti.Placeholder = "search projects..."
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 40

	m := Model{
		repo:   repo,
		input:  ti,
		status: "loading...",
	}
	return m
}

// Init loads projects.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		projects, err := m.repo.List("", "", false)
		if err != nil {
			return errMsg{err}
		}
		return projectsMsg{projects}
	}
}

type projectsMsg struct{ projects []model.Project }
type errMsg struct{ err error }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4
		return m, nil

	case projectsMsg:
		m.projects = msg.projects
		m.status = fmt.Sprintf("%d projects", len(m.projects))
		m.applyFilter()
		return m, nil

	case errMsg:
		m.status = "error: " + msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.selected < len(m.filtered) {
				p := m.filtered[m.selected]
				m.opened = true
				m.openPath = p.Path
				m.openNew = false
				_ = m.repo.TouchLastOpened(p.ID)
				return m, tea.Quit
			}
		case "ctrl+o":
			if len(m.filtered) > 0 && m.selected < len(m.filtered) {
				p := m.filtered[m.selected]
				m.opened = true
				m.openPath = p.Path
				m.openNew = true
				_ = m.repo.TouchLastOpened(p.ID)
				return m, tea.Quit
			}
		case "up", "ctrl+p", "ctrl+k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "ctrl+j":
			if m.selected < len(m.filtered)-1 {
				m.selected++
			}
		case "ctrl+r":
			return m, func() tea.Msg {
				// refresh all
				projs, _ := m.repo.List("", "", false)
				return projectsMsg{projs}
			}
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.applyFilter()
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		m.filtered = append([]model.Project(nil), m.projects...)
	} else {
		var out []model.Project
		for _, p := range m.projects {
			name := strings.ToLower(p.Name)
			path := strings.ToLower(p.Path)
			if strings.Contains(name, q) || strings.Contains(path, q) {
				out = append(out, p)
			}
		}
		// sort: favorites first, then by score (contains position)
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Favorite != out[j].Favorite {
				return out[i].Favorite
			}
			return false
		})
		m.filtered = out
	}
	if m.selected >= len(m.filtered) {
		m.selected = 0
	}
	if len(m.filtered) == 0 {
		m.selected = 0
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		m.width = 80
	}
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4"))
	favStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	gitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	var b strings.Builder
	b.WriteString(headerStyle.Render("projx") + "  " + dimStyle.Render(m.status) + "\n")
	b.WriteString(m.input.View() + "\n")
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	listWidth := m.width
	if m.width > 60 {
		listWidth = m.width * 2 / 3
	}
	// Build list lines
	var listLines []string
	for i, p := range m.filtered {
		name := p.Name
		if name == "" {
			name = baseName(p.Path)
		}
		git := ""
		if p.GitBranch != "" {
			git = " [" + p.GitBranch
			if p.GitDirty > 0 {
				git += " +" + fmt.Sprintf("%d", p.GitDirty)
			}
			git += "]"
		}
		marker := "  "
		if p.Favorite {
			marker = favStyle.Render("* ")
		}
		line := fmt.Sprintf("%s%-26s %-30s%s", marker, truncate(name, 26), truncate(shortPath(p.Path, m.homeDir()), 30), gitStyle.Render(git))
		if i == m.selected {
			line = selStyle.Render(padRight(line, listWidth))
		}
		listLines = append(listLines, line)
	}
	if len(listLines) == 0 {
		listLines = append(listLines, dimStyle.Render("  no projects"))
	}

	// Preview
	var previewLines []string
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		p := m.filtered[m.selected]
		previewLines = append(previewLines, headerStyle.Render("Preview"))
		previewLines = append(previewLines, "Name:   "+p.Name)
		previewLines = append(previewLines, "Path:   "+p.Path)
		previewLines = append(previewLines, "Editor: "+orDefault(p.Editor, "default"))
		if p.Profile != "" {
			previewLines = append(previewLines, "Profile: "+p.Profile)
		}
		if len(p.Tags) > 0 {
			previewLines = append(previewLines, "Tags:   "+strings.Join(p.Tags, ", "))
		}
		if p.Group != "" {
			previewLines = append(previewLines, "Group:  "+p.Group)
		}
		if p.GitBranch != "" {
			previewLines = append(previewLines, "Git:    "+p.GitBranch+fmt.Sprintf(" (%d dirty)", p.GitDirty))
		}
		if p.Remote != "" {
			previewLines = append(previewLines, "Remote: "+p.Remote)
		}
		if !p.LastOpened.IsZero() {
			previewLines = append(previewLines, "Opened: "+p.LastOpened.Format("2006-01-02 15:04"))
		}
	} else {
		previewLines = append(previewLines, dimStyle.Render("no selection"))
	}

	// Render side by side
	maxLines := m.height - 6
	if maxLines < 5 {
		maxLines = 5
	}
	for i := 0; i < maxLines; i++ {
		left := ""
		if i < len(listLines) {
			left = listLines[i]
		}
		right := ""
		if i < len(previewLines) {
			right = previewLines[i]
		}
		b.WriteString(padRight(stripAnsi(left), listWidth))
		b.WriteString("│")
		b.WriteString(" " + right + "\n")
	}
	b.WriteString(strings.Repeat("─", m.width) + "\n")
	b.WriteString(dimStyle.Render("Enter: open  Ctrl+O: new window  Ctrl+R: refresh  Esc: quit"))

	return b.String()
}

func (m Model) homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

// Opened returns whether a project was selected.
func (m Model) Opened() bool     { return m.opened }
func (m Model) OpenPath() string { return m.openPath }
func (m Model) OpenNew() bool    { return m.openNew }

// --- helpers ---

func baseName(p string) string {
	parts := strings.Split(strings.TrimRight(p, "/"), "/")
	if len(parts) == 0 {
		return p
	}
	return parts[len(parts)-1]
}

func shortPath(p, home string) string {
	if home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func padRight(s string, n int) string {
	// pad based on visible length (strip ansi)
	vis := visibleLen(s)
	if vis >= n {
		return s
	}
	return s + strings.Repeat(" ", n-vis)
}

func visibleLen(s string) int {
	// strip ansi escape sequences
	out := 0
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				inEsc = false
			}
			continue
		}
		out++
	}
	return out
}

func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

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
