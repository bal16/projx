package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bal16/projx/internal/db"
	"github.com/bal16/projx/internal/ghinfo"
	"github.com/bal16/projx/internal/gitinfo"
	"github.com/bal16/projx/internal/model"
)

const ghFieldIndex = 5

var ghCycle = []string{"unset", "on", "off"}

// EditForm manages inline editing of a project.
type EditForm struct {
	active    bool
	idx       int
	projectID string
	inputs    []textinput.Model
}

// NewEditForm creates an initialized edit form.
func NewEditForm() EditForm {
	f := EditForm{inputs: make([]textinput.Model, 6)}
	for i, label := range []string{"Name:", "Editor:", "Profile:", "Tags:", "Group:", "GitHub:"} {
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
	vals := []string{p.Name, p.Editor, p.Profile, strings.Join(p.Tags, ", "), p.Group, ghEnabledString(p.GHEnabled)}
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
	case "left", "right", " ":
		if f.idx == ghFieldIndex {
			f.cycleGH(msg.String() == "left")
			for i := range f.inputs {
				if i == f.idx {
					f.inputs[i].Focus()
				} else {
					f.inputs[i].Blur()
				}
			}
			return nil, ""
		}
		var c tea.Cmd
		f.inputs[f.idx], c = f.inputs[f.idx].Update(msg)
		return c, ""
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

func (f *EditForm) cycleGH(back bool) {
	cur := strings.ToLower(strings.TrimSpace(f.inputs[ghFieldIndex].Value()))
	i := 0
	for idx, v := range ghCycle {
		if v == cur {
			i = idx
			break
		}
	}
	if back {
		i = (i + len(ghCycle) - 1) % len(ghCycle)
	} else {
		i = (i + 1) % len(ghCycle)
	}
	f.inputs[ghFieldIndex].SetValue(ghCycle[i])
	f.inputs[ghFieldIndex].CursorEnd()
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
	ed.GHEnabled = parseGHEnabled(f.inputs[ghFieldIndex].Value())
	if err := repo.Update(&ed); err != nil {
		return nil, "error: " + err.Error()
	}
	f.active = false
	return refreshProjectAndLoadCmd(repo, ed), "updated " + ed.Name
}

func refreshProjectAndLoadCmd(repo *db.Repository, p model.Project) tea.Cmd {
	return func() tea.Msg {
		if gi := gitinfo.Inspect(p.Path); gi.IsRepo {
			_ = repo.UpdateGitStatus(p.ID, gi)
		}
		if p.GHEnabled != nil && *p.GHEnabled {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if c, err := ghinfo.FetchOpenCounts(ctx, p.Path); err == nil {
				_ = repo.UpdateGitHubStatus(p.ID, c.OpenPRs, c.OpenIssues, c.FetchedAt)
			}
			cancel()
		}
		projects, err := repo.List("", "", false)
		if err != nil {
			return errMsg{err}
		}
		return projectsMsg{projects}
	}
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
		case ghFieldIndex:
			b.WriteString("  " + dimStyle.Render("space/←/→ to cycle: unset, on, off (fetch PR/issues via gh when on)") + "\n")
		}
	}
	b.WriteString(strings.Repeat("-", width) + "\n")
	b.WriteString(dimStyle.Render("Tab/arrows: field   Enter: save   Esc: cancel"))
	return b.String()
}

func ghEnabledString(v *bool) string {
	if v == nil {
		return "unset"
	}
	if *v {
		return "on"
	}
	return "off"
}

func parseGHEnabled(s string) *bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "true", "1", "yes":
		v := true
		return &v
	case "off", "false", "0", "no":
		v := false
		return &v
	default:
		return nil
	}
}
