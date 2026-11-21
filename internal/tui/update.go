package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bal16/projx/internal/model"
)

// Update handles messages and input.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4
		m.edit.SetWidth(msg.Width)
		m.ensureVisible()
		return m, nil

	case projectsMsg:
		m.projects = msg.projects
		m.applyFilter()
		if m.status == "refreshed status" {
			m.status = fmt.Sprintf("refreshed status (%d projects)", len(m.projects))
		} else if m.lastOutput == "" {
			m.status = fmt.Sprintf("%d projects", len(m.projects))
		}
		return m, nil

	case errMsg:
		m.status = "error: " + msg.err.Error()
		return m, nil

	case commandResultMsg:
		if msg.err != nil {
			m.status = "cmd error: " + msg.err.Error()
			m.lastOutput = msg.err.Error() + "\n" + msg.output
		} else {
			if msg.cmd == "refresh" {
				m.status = "refreshed status"
				m.lastOutput = ""
			} else {
				m.status = "ran: " + msg.cmd
				m.lastOutput = msg.output
			}
		}
		return m, loadProjectsCmd(m.repo)

	case tea.KeyMsg:
		key := msg.String()

		if m.edit.Active() {
			cmd, status := m.edit.Update(msg, m.repo, m.projects)
			if status != "" {
				m.status = status
			}
			return m, cmd
		}

		// Command mode
		if m.mode == "command" {
			switch key {
			case "esc":
				m.mode = "normal"
				m.input.SetValue("")
				m.input.Blur()
				m.cmd.Reset()
				return m, nil
			case "enter":
				cmdStr := strings.TrimSpace(m.input.Value())
				m.input.SetValue("")
				m.mode = "normal"
				m.input.Blur()
				m.cmd.Reset()
				if cmdStr == "" {
					return m, nil
				}
				return m, m.runCommand(cmdStr)
			case "tab":
				m.cmd.CompleteNext(&m.input)
				return m, nil
			case "shift+tab":
				m.cmd.CompletePrev(&m.input)
				return m, nil
			}
			var c tea.Cmd
			m.input, c = m.input.Update(msg)
			m.cmd.UpdateSuggestions(m.input.Value())
			return m, c
		}

		// Open Editor mode: choose editor override
		if m.mode == "open_editor" {
			switch key {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.mode = "normal"
				m.input.SetValue("")
				m.input.Blur()
				return m, nil
			case "enter":
				if len(m.filtered) > 0 && m.selected < len(m.filtered) {
					p := m.filtered[m.selected]
					m.opened = true
					m.openPath = p.Path
					m.openNew = false
					m.openEditor = strings.TrimSpace(m.input.Value())
					_ = m.repo.TouchLastOpened(p.ID)
					return m, tea.Quit
				}
			case "tab":
				m.cycleEditorCompletion()
				return m, nil
			default:
				var c tea.Cmd
				m.input, c = m.input.Update(msg)
				return m, c
			}
			return m, nil
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
					m.ensureVisible()
				}
			case "down", "ctrl+j", "ctrl+n":
				if m.selected < len(m.filtered)-1 {
					m.selected++
					m.ensureVisible()
				}
			case "ctrl+r":
				return m, loadProjectsCmd(m.repo)
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
	if m.pendingKey == "g" {
		m.pendingKey = ""
		if key == "g" {
			m.lastOutput = ""
			m.showProfiles = false
			m.selected = 0
			m.ensureVisible()
			return m, nil
		}
	}
	switch key {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "j", "down", "ctrl+j", "ctrl+d", "pgdown":
		m.lastOutput = ""
		m.showProfiles = false
		if m.selected < len(m.filtered)-1 {
			if key == "pgdown" || key == "ctrl+d" {
				m.selected = min(m.selected+m.maxLines(), len(m.filtered)-1)
			} else {
				m.selected++
			}
			m.ensureVisible()
		}
	case "k", "up", "ctrl+k", "ctrl+u", "pgup":
		m.lastOutput = ""
		m.showProfiles = false
		if m.selected > 0 {
			if key == "pgup" || key == "ctrl+u" {
				m.selected = max(m.selected-m.maxLines(), 0)
			} else {
				m.selected--
			}
			m.ensureVisible()
		}
	case "g":
		m.pendingKey = "g"
	case "G":
		m.lastOutput = ""
		m.showProfiles = false
		if len(m.filtered) > 0 {
			m.selected = len(m.filtered) - 1
			m.ensureVisible()
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
		m.cmd.UpdateSuggestions("")
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
	case "E":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			m.mode = "open_editor"
			m.input.SetValue("")
			m.input.Focus()
			m.input.Placeholder = "open with editor (e.g. zed, code, nvim)..."
			m.lastOutput = ""
			m.showProfiles = false
		}
	case "d":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			p := m.filtered[m.selected]
			_ = m.repo.Remove(p.ID)
			return m, loadProjectsCmd(m.repo)
		}
	case "f":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			pp := m.filtered[m.selected]
			pp.Favorite = !pp.Favorite
			_ = m.repo.Update(&pp)
			return m, loadProjectsCmd(m.repo)
		}
	case "r":
		m.lastOutput = ""
		return m, loadProjectsCmd(m.repo)
	case "R":
		m.lastOutput = ""
		m.status = "refreshing status..."
		return m, m.runCommand("refresh")
	case "p":
		m.showProfiles = !m.showProfiles
		if m.showProfiles {
			return m, runProfilesListCmd()
		}
		m.lastOutput = ""
	case "e":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			m.edit.Open(m.filtered[m.selected])
		}
	case "?":
		m.lastOutput = "Normal mode keys:\n  j/k or arrows: navigate\n  gg/G: top/bottom\n  / or i: search mode\n  :: command mode\n  Enter/o: open   O: open new window   E: choose editor\n  e: edit project   d: delete project   f: toggle favorite\n  p: toggle profiles view   r: reload list   R: refresh status (git/GitHub)\n  ?: this help   q: quit"
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
		sortProjects(out, q)
		m.filtered = out
	}
	if m.selected >= len(m.filtered) {
		m.selected = 0
	}
	if len(m.filtered) == 0 {
		m.selected = 0
	}
	m.ensureVisible()
}

func sortProjects(out []model.Project, q string) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Favorite != out[j].Favorite {
			return out[i].Favorite
		}
		pi := strings.Index(strings.ToLower(out[i].Name), q)
		pj := strings.Index(strings.ToLower(out[j].Name), q)
		if pi != pj {
			if pi < 0 {
				return false
			}
			if pj < 0 {
				return true
			}
			return pi < pj
		}
		if !out[i].LastOpened.Equal(out[j].LastOpened) {
			return out[i].LastOpened.After(out[j].LastOpened)
		}
		return out[i].Name < out[j].Name
	})
}

var knownEditors = []string{"zed", "code", "codium", "cursor", "nvim", "vim", "helix"}

func (m *Model) cycleEditorCompletion() {
	val := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if val == "" {
		m.input.SetValue(knownEditors[0])
		m.input.CursorEnd()
		return
	}
	for i, ed := range knownEditors {
		if ed == val {
			next := (i + 1) % len(knownEditors)
			m.input.SetValue(knownEditors[next])
			m.input.CursorEnd()
			return
		}
	}
	for _, ed := range knownEditors {
		if strings.HasPrefix(ed, val) {
			m.input.SetValue(ed)
			m.input.CursorEnd()
			return
		}
	}
	m.input.SetValue(knownEditors[0])
	m.input.CursorEnd()
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
