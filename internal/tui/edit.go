package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bal16/projx/internal/db"
	"github.com/bal16/projx/internal/model"
)

// EditForm manages inline editing of a project.
type EditForm struct {
	active    bool
	idx       int
	projectID string
	inputs    []textinput.Model
}

// NewEditForm creates an initialized edit form.
func NewEditForm() EditForm {
	f := EditForm{inputs: make([]textinput.Model, 5)}
	for i, label := range []string{"Name:", "Editor:", "Profile:", "Tags:", "Group:"} {
		in := textinput.New()
		in.Prompt = label + " "
		in.CharLimit = 128
		f.inputs[i] = in
	}
	return f
}

// Active reports whether the form is open.
func (f EditForm) Active() bool { return f.active }

// SetWidth updates input widths for the current window size.
func (f *EditForm) SetWidth(width int) {
	for i := range f.inputs {
		f.inputs[i].Width = width - 12
	}
}

// Open populates and focuses the form for a project.
func (f *EditForm) Open(p model.Project) {
	f.active = true
	f.projectID = p.ID
	f.idx = 0
	vals := []string{p.Name, p.Editor, p.Profile, strings.Join(p.Tags, ", "), p.Group}
	for i := range f.inputs {
		f.inputs[i].SetValue(vals[i])
		f.inputs[i].CursorEnd()
		if i == 0 {
			f.inputs[i].Focus()
		} else {
			f.inputs[i].Blur()
		}
	}
}

// Close hides the form.
func (f *EditForm) Close() { f.active = false }

// Update handles a key message while the form is active.
// It returns an optional command (e.g. reload after save) and a status message.
func (f *EditForm) Update(msg tea.KeyMsg, repo *db.Repository, projects []model.Project) (tea.Cmd, string) {
	switch msg.String() {
	case "esc":
		f.active = false
		return nil, ""
	case "enter":
		return f.save(repo, projects)
	case "tab", "down", "ctrl+j", "ctrl+n":
		f.idx = (f.idx + 1) % len(f.inputs)
	case "shift+tab", "up", "ctrl+k", "ctrl+p":
		f.idx = (f.idx + len(f.inputs) - 1) % len(f.inputs)
	default:
		var c tea.Cmd
		f.inputs[f.idx], c = f.inputs[f.idx].Update(msg)
		return c, ""
	}
	for i := range f.inputs {
		if i == f.idx {
			f.inputs[i].Focus()
		} else {
			f.inputs[i].Blur()
		}
	}
	return nil, ""
}

func (f *EditForm) save(repo *db.Repository, projects []model.Project) (tea.Cmd, string) {
	var ed model.Project
	for _, p := range projects {
		if p.ID == f.projectID {
			ed = p
		}
	}
	if ed.ID == "" {
		f.active = false
		return nil, ""
	}
	if v := strings.TrimSpace(f.inputs[0].Value()); v != "" {
		ed.Name = v
	}
	ed.Editor = strings.TrimSpace(f.inputs[1].Value())
	ed.Profile = strings.TrimSpace(f.inputs[2].Value())
	ed.Tags = splitCSV(f.inputs[3].Value())
	ed.Group = strings.TrimSpace(f.inputs[4].Value())
	if err := repo.Update(&ed); err != nil {
		return nil, "error: " + err.Error()
	}
	f.active = false
	return loadProjectsCmd(repo), "updated " + ed.Name
}

// View renders the edit form.
func (f EditForm) View(status string, width int) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("projx") + "  " + dimStyle.Render("[edit] "+status) + "\n")
	b.WriteString(strings.Repeat("-", width) + "\n")
	for i, in := range f.inputs {
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
	b.WriteString(strings.Repeat("-", width) + "\n")
	b.WriteString(dimStyle.Render("Tab/arrows: field   Enter: save   Esc: cancel"))
	return b.String()
}
