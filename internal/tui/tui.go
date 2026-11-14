package tui

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/bal16/projx/internal/db"
	"github.com/bal16/projx/internal/model"
	"github.com/bal16/projx/internal/util"
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
	return func() tea.Msg {
		projects, err := m.repo.List("", "", false)
		if err != nil {
			return errMsg{err}
		}
		return projectsMsg{projects}
	}
}

var cmdSuggestionsBase = []string{
	"add", "config", "doctor", "export", "favorite", "group", "help",
	"import", "list", "open", "profiles", "refresh", "remove", "rename",
	"scan", "set", "tag", "tui", "untag", "version", "ws",
}

func (m *Model) updateCmdSuggestions() {
	val := m.input.Value()
	// complete first token only (simple, lazyvim-like command completion)
	if strings.Contains(val, " ") {
		m.cmdSugs = nil
		m.cmdSugSel = 0
		return
	}
	prefix := strings.ToLower(strings.TrimSpace(val))
	if prefix == "" {
		m.cmdSugs = append([]string(nil), cmdSuggestionsBase...)
		m.cmdSugSel = 0
		return
	}
	var out []string
	for _, c := range cmdSuggestionsBase {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	m.cmdSugs = out
	m.cmdSugSel = 0
}

type projectsMsg struct{ projects []model.Project }
type errMsg struct{ err error }
type commandResultMsg struct {
	cmd    string
	output string
	err    error
}

func (m Model) runCommand(cmd string) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return commandResultMsg{cmd: cmd, output: ""}
		}
		c := exec.Command(exe, parts...)
		out, err := c.CombinedOutput()
		return commandResultMsg{cmd: cmd, output: string(out), err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4
		for i := range m.editInputs {
			m.editInputs[i].Width = msg.Width - 12
		}
		return m, nil

	case projectsMsg:
		m.projects = msg.projects
		m.applyFilter()
		if m.lastOutput == "" {
			m.status = fmt.Sprintf("%d projects", len(m.projects))
		}
		return m, nil

	case errMsg:
		m.status = "error: " + msg.err.Error()
		return m, nil

	case commandResultMsg:
		m.lastOutput = msg.output
		if msg.err != nil {
			m.status = "cmd error: " + msg.err.Error()
			m.lastOutput = msg.err.Error() + "\n" + msg.output
		} else {
			m.status = "ran: " + msg.cmd
		}
		return m, func() tea.Msg {
			projects, err := m.repo.List("", "", false)
			if err != nil {
				return errMsg{err}
			}
			return projectsMsg{projects}
		}

	case tea.KeyMsg:
		key := msg.String()

		// Edit mode: form for the selected project.
		if m.editing {
			switch key {
			case "esc":
				m.editing = false
				return m, nil
			case "enter":
				var ed model.Project
				for _, p := range m.projects {
					if p.ID == m.editID {
						ed = p
					}
				}
				if ed.ID != "" {
					if v := strings.TrimSpace(m.editInputs[0].Value()); v != "" {
						ed.Name = v
					}
					ed.Editor = strings.TrimSpace(m.editInputs[1].Value())
					ed.Profile = strings.TrimSpace(m.editInputs[2].Value())
					ed.Tags = splitCSV(m.editInputs[3].Value())
					ed.Group = strings.TrimSpace(m.editInputs[4].Value())
					if err := m.repo.Update(&ed); err != nil {
						m.status = "error: " + err.Error()
						return m, nil
					}
					m.editing = false
					m.status = "updated " + ed.Name
					return m, func() tea.Msg {
						projects, err := m.repo.List("", "", false)
						if err != nil {
							return errMsg{err}
						}
						return projectsMsg{projects}
					}
				}
				m.editing = false
				return m, nil
			case "tab", "down", "ctrl+j", "ctrl+n":
				m.editIdx = (m.editIdx + 1) % len(m.editInputs)
			case "shift+tab", "up", "ctrl+k", "ctrl+p":
				m.editIdx = (m.editIdx + len(m.editInputs) - 1) % len(m.editInputs)
			default:
				var c tea.Cmd
				m.editInputs[m.editIdx], c = m.editInputs[m.editIdx].Update(msg)
				return m, c
			}
			for i := range m.editInputs {
				if i == m.editIdx {
					m.editInputs[i].Focus()
				} else {
					m.editInputs[i].Blur()
				}
			}
			return m, nil
		}

		// Command mode
		if m.mode == "command" {
			switch key {
			case "esc":
				m.mode = "normal"
				m.input.SetValue("")
				m.input.Blur()
				m.cmdSugs = nil
				return m, nil
			case "enter":
				cmd := strings.TrimSpace(m.input.Value())
				m.input.SetValue("")
				m.mode = "normal"
				m.input.Blur()
				m.cmdSugs = nil
				if cmd == "" {
					return m, nil
				}
				return m, m.runCommand(cmd)
			case "tab":
				if len(m.cmdSugs) > 0 {
					m.input.SetValue(m.cmdSugs[m.cmdSugSel])
					m.input.CursorEnd()
					m.cmdSugSel = (m.cmdSugSel + 1) % len(m.cmdSugs)
				}
				return m, nil
			case "shift+tab":
				if len(m.cmdSugs) > 0 {
					m.cmdSugSel = (m.cmdSugSel - 1 + len(m.cmdSugs)) % len(m.cmdSugs)
					m.input.SetValue(m.cmdSugs[m.cmdSugSel])
					m.input.CursorEnd()
				}
				return m, nil
			}
			var c tea.Cmd
			m.input, c = m.input.Update(msg)
			m.updateCmdSuggestions()
			return m, c
		}

		// Search mode (fuzzy): typing filters
		if m.mode == "search" {
			switch key {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.mode = "normal"
				m.input.Blur()
				m.pendingKey = ""
				return m, nil
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
					projects, _ := m.repo.List("", "", false)
					return projectsMsg{projects}
				}
			default:
				m.showProfiles = false
				m.lastOutput = ""
				var c tea.Cmd
				m.input, c = m.input.Update(msg)
				m.applyFilter()
				return m, c
			}
			return m, nil
		}

		return m.handleNormalMode(key)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) handleNormalMode(key string) (tea.Model, tea.Cmd) {
	// Normal mode (Vim-like)
	if m.pendingKey == "g" {
		m.pendingKey = ""
		if key == "g" {
			m.lastOutput = ""
			m.showProfiles = false
			m.selected = 0
			return m, nil
		}
	}
	switch key {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "j", "down":
		m.lastOutput = ""
		m.showProfiles = false
		if m.selected < len(m.filtered)-1 {
			m.selected++
		}
	case "k", "up":
		m.lastOutput = ""
		m.showProfiles = false
		if m.selected > 0 {
			m.selected--
		}
	case "g":
		m.pendingKey = "g"
	case "G":
		m.lastOutput = ""
		m.showProfiles = false
		if len(m.filtered) > 0 {
			m.selected = len(m.filtered) - 1
		}
	case "/":
		m.mode = "search"
		m.input.Focus()
		m.input.Placeholder = "search projects..."
		m.lastOutput = ""
		m.showProfiles = false
	case "i":
		m.mode = "search"
		m.input.Focus()
		m.input.Placeholder = "search projects..."
		m.lastOutput = ""
		m.showProfiles = false
	case ":":
		m.mode = "command"
		m.input.SetValue("")
		m.input.Focus()
		m.input.Placeholder = "command (scan, import, favorite, set, ws, config, doctor, refresh, help)"
		m.lastOutput = ""
		m.showProfiles = false
		m.updateCmdSuggestions()
	case "enter", "o":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			p := m.filtered[m.selected]
			m.opened = true
			m.openPath = p.Path
			m.openNew = false
			_ = m.repo.TouchLastOpened(p.ID)
			return m, tea.Quit
		}
	case "O":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			p := m.filtered[m.selected]
			m.opened = true
			m.openPath = p.Path
			m.openNew = true
			_ = m.repo.TouchLastOpened(p.ID)
			return m, tea.Quit
		}
	case "d":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			p := m.filtered[m.selected]
			_ = m.repo.Remove(p.ID)
			return m, func() tea.Msg {
				projects, _ := m.repo.List("", "", false)
				return projectsMsg{projects}
			}
		}
	case "f":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			pp := m.filtered[m.selected]
			pp.Favorite = !pp.Favorite
			_ = m.repo.Update(&pp)
			return m, func() tea.Msg {
				projects, _ := m.repo.List("", "", false)
				return projectsMsg{projects}
			}
		}
	case "r":
		return m, func() tea.Msg {
			projects, _ := m.repo.List("", "", false)
			return projectsMsg{projects}
		}
	case "p":
		m.showProfiles = !m.showProfiles
		if m.showProfiles {
			return m, func() tea.Msg {
				exe, _ := os.Executable()
				c := exec.Command(exe, "profiles", "list", "--json")
				out, _ := c.CombinedOutput()
				return commandResultMsg{cmd: "profiles list", output: string(out)}
			}
		}
		m.lastOutput = ""
	case "e":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			p := m.filtered[m.selected]
			m.editing = true
			m.editID = p.ID
			m.editIdx = 0
			if len(m.editInputs) == 0 {
				m.editInputs = make([]textinput.Model, 5)
				for i, label := range []string{"Name:", "Editor:", "Profile:", "Tags:", "Group:"} {
					m.editInputs[i] = textinput.New()
					m.editInputs[i].Prompt = label + " "
					m.editInputs[i].CharLimit = 128
				}
			}
			vals := []string{p.Name, p.Editor, p.Profile, strings.Join(p.Tags, ", "), p.Group}
			for i := range m.editInputs {
				m.editInputs[i].SetValue(vals[i])
				m.editInputs[i].CursorEnd()
			}
			for i := range m.editInputs {
				if i == 0 {
					m.editInputs[i].Focus()
				} else {
					m.editInputs[i].Blur()
				}
			}
		}
	case "?":
		m.lastOutput = "Normal mode keys:\n  j/k or arrows: navigate\n  gg/G: top/bottom\n  / or i: search mode\n  :: command mode\n  Enter/o: open   O: open new window\n  e: edit project   d: delete project   f: toggle favorite\n  p: toggle profiles view   r: refresh list   ?: this help\n  q: quit"
	}
	return m, nil
}

func (m *Model) applyFilter() {
	if m.mode == "command" {
		return
	}
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
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Favorite != out[j].Favorite {
				return out[i].Favorite
			}
			// earlier match position in name wins
			pi := strings.Index(strings.ToLower(out[i].Name), q)
			pj := strings.Index(strings.ToLower(out[j].Name), q)
			if pi != pj {
				// unmatched (-1) sorts last
				if pi < 0 {
					return false
				}
				if pj < 0 {
					return true
				}
				return pi < pj
			}
			// then last-opened desc
			if !out[i].LastOpened.Equal(out[j].LastOpened) {
				return out[i].LastOpened.After(out[j].LastOpened)
			}
			return out[i].Name < out[j].Name
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
	if m.editing {
		var b strings.Builder
		b.WriteString(headerStyle.Render("projx") + "  " + dimStyle.Render("[edit] "+m.status) + "\n")
		b.WriteString(strings.Repeat("-", m.width) + "\n")
		for i, in := range m.editInputs {
			b.WriteString(in.View() + "\n")
			switch i {
			case 1:
				b.WriteString("  " + dimStyle.Render("empty = follow default_editor") + "\n")
			case 2:
				b.WriteString("  " + dimStyle.Render("VS Code profile (empty = none)") + "\n")
			case 3:
				b.WriteString("  " + dimStyle.Render("comma separated") + "\n")
			}
		}
		b.WriteString(strings.Repeat("-", m.width) + "\n")
		b.WriteString(dimStyle.Render("Tab/arrows: field   Enter: save   Esc: cancel"))
		return b.String()
	}
	selStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4"))
	favStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	gitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	var b strings.Builder
	modeLabel := m.mode
	b.WriteString(headerStyle.Render("projx") + "  " + dimStyle.Render("["+modeLabel+"] "+m.status) + "\n")
	if m.mode == "search" || m.mode == "command" {
		b.WriteString(m.input.View() + "\n")
	}
	b.WriteString(strings.Repeat("-", m.width) + "\n")

	listWidth := m.width * 2 / 3
	if avail := m.width - 40; m.width > 40 && listWidth > avail {
		listWidth = avail // keep the preview at least ~38 cols
	}
	if listWidth > 90 {
		listWidth = 90 // cap dead space on very wide screens
	}
	twoCol := m.width >= 60 && listWidth < m.width-2
	if !twoCol {
		listWidth = m.width
	}

	var listLines []string
	nameCol := 26
	// path column adapts to the list width; keep ~12 cols for the git tag
	pathCol := min(max(listWidth-4-nameCol-12, 16), 48)
	for i, p := range m.filtered {
		name := p.Name
		if name == "" {
			name = util.BaseName(p.Path)
		}
		git := ""
		if p.GitBranch != "" {
			git = " [" + p.GitBranch
			if p.GitDirty > 0 {
				git += " +" + fmt.Sprintf("%d", p.GitDirty)
			}
			git += "]"
		}
		ptr := "  "
		if i == m.selected {
			ptr = "> "
		}
		star := " "
		if p.Favorite {
			star = favStyle.Render("*")
		}
		line := fmt.Sprintf("%s%s %-*s %-*s%s", ptr, star, nameCol, truncate(name, nameCol), pathCol, truncate(shortPath(p.Path, m.homeDir()), pathCol), gitStyle.Render(git))
		if i == m.selected {
			line = selStyle.Render(padRight(line, listWidth))
		}
		listLines = append(listLines, line)
	}
	if len(listLines) == 0 {
		listLines = append(listLines, dimStyle.Render("  no projects"))
	}

	var previewLines []string
	switch {
	case m.mode == "command":
		previewLines = append(previewLines, headerStyle.Render("Commands"))
		if len(m.cmdSugs) == 0 {
			previewLines = append(previewLines, dimStyle.Render("  (no matches)"))
		} else {
			hi := m.cmdSugSel
			cur := strings.TrimSpace(m.input.Value())
			for i, s := range m.cmdSugs {
				if s == cur {
					hi = i
					break
				}
			}
			for i, s := range m.cmdSugs {
				if i == hi {
					previewLines = append(previewLines, selStyle.Render("> "+s))
				} else {
					previewLines = append(previewLines, "  "+s)
				}
			}
		}
	case m.lastOutput != "":
		previewLines = append(previewLines, strings.Split(strings.TrimSpace(m.lastOutput), "\n")...)
	case len(m.filtered) > 0 && m.selected < len(m.filtered):
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
	default:
		previewLines = append(previewLines, dimStyle.Render("no selection"))
	}
	if m.mode == "command" && !twoCol {
		listLines = previewLines
		previewLines = nil
	}

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
		if twoCol && i < len(previewLines) {
			right = truncate(previewLines[i], m.width-listWidth-3)
		}
		if !twoCol {
			b.WriteString(left + "\n")
			continue
		}
		b.WriteString(padRight(left, listWidth))
		b.WriteString(dimStyle.Render("|"))
		b.WriteString(" " + right + "\n")
	}
	b.WriteString(strings.Repeat("-", m.width) + "\n")
	switch m.mode {
	case "search":
		b.WriteString(dimStyle.Render("type to filter | Esc: normal mode | Enter: open | Ctrl+O: new window"))
	case "normal":
		b.WriteString(dimStyle.Render("j/k: nav  gg/G: top/bottom  /: search  :: cmd  Enter/o: open  O: new win  e: edit  d: del  f: fav  r: refresh  ?: help  q: quit"))
	case "command":
		b.WriteString(dimStyle.Render("Tab: complete  Enter: run  Esc: cancel"))
	}

	return b.String()
}

func (m Model) homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func (m Model) Opened() bool     { return m.opened }
func (m Model) OpenPath() string { return m.openPath }
func (m Model) OpenNew() bool    { return m.openNew }

func shortPath(p, home string) string {
	if home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func truncate(s string, n int) string {
	if ansi.StringWidth(s) <= n {
		return s
	}
	if n <= 1 {
		return ansi.Truncate(s, n, "")
	}
	return ansi.Truncate(s, n-1, "…")
}

func padRight(s string, n int) string {
	w := ansi.StringWidth(s)
	if w >= n {
		return ansi.Truncate(s, n, "")
	}
	return s + strings.Repeat(" ", n-w)
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// splitCSV parses a comma-separated input into a clean tag slice.
func splitCSV(s string) []string {
	out := []string{}
	for _, part := range strings.Split(s, ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
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
