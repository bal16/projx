package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/bal16/projx/internal/model"
	"github.com/bal16/projx/internal/util"
)

// View renders the TUI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		m.width = 80
	}

	if m.edit.Active() {
		return m.edit.View(m.status, m.width)
	}

	var b strings.Builder
	modeLabel := m.mode
	b.WriteString(headerStyle.Render("projx") + "  " + dimStyle.Render("["+modeLabel+"] "+m.status) + "\n")
	if m.mode == "search" || m.mode == "command" || m.mode == "open_editor" {
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
	pathCol := min(max(listWidth-4-nameCol-12, 16), 48)
	for i, p := range m.filtered {
		name := p.Name
		if name == "" {
			name = util.BaseName(p.Path)
		}
		git := formatGitList(p)
		ptr := "  "
		if i == m.selected {
			ptr = "> "
		}
		star := " "
		if p.Favorite {
			star = favStyle.Render("*")
		}
		line := fmt.Sprintf(
			"%s%s %-*s %-*s%s",
			ptr, star,
			nameCol, truncate(name, nameCol),
			pathCol, truncate(shortPath(p.Path, m.homeDir()), pathCol),
			renderGitStatus(p, git),
		)
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
		previewLines = append(previewLines, m.cmd.PreviewLines(strings.TrimSpace(m.input.Value()))...)
	case m.lastOutput != "":
		previewLines = append(previewLines, strings.Split(strings.TrimSpace(m.lastOutput), "\n")...)
	case len(m.filtered) > 0 && m.selected < len(m.filtered):
		p := m.filtered[m.selected]
		previewLines = append(previewLines, headerStyle.Render("Preview"))
		previewLines = append(previewLines, "Name:     "+p.Name)
		previewLines = append(previewLines, "Path:     "+p.Path)
		previewLines = append(previewLines, "Editor:   "+orDefault(p.Editor, "default"))
		if p.Profile != "" {
			previewLines = append(previewLines, "Profile:  "+p.Profile)
		}
		if len(p.Tags) > 0 {
			previewLines = append(previewLines, "Tags:     "+strings.Join(p.Tags, ", "))
		}
		if p.Group != "" {
			previewLines = append(previewLines, "Group:    "+p.Group)
		}
		if line := formatGitPreview(p); line != "" {
			previewLines = append(previewLines, "Git:      "+renderGitStatus(p, line))
		}
		if line, ok := formatGitHub(p); ok {
			previewLines = append(previewLines, line)
		}
		if p.Remote != "" {
			previewLines = append(previewLines, "Remote:   "+p.Remote)
		}
		previewLines = append(previewLines, "Modified: "+formatModified(p.Path))
		if !p.LastOpened.IsZero() {
			previewLines = append(previewLines, "Opened:   "+p.LastOpened.Format("2006-01-02 15:04"))
		}
	default:
		previewLines = append(previewLines, dimStyle.Render("no selection"))
	}
	if m.mode == "command" && !twoCol {
		listLines = previewLines
		previewLines = nil
	}

	maxLines := m.maxLines()
	offset := m.scrollOffset
	if m.selected < offset {
		offset = m.selected
	}
	if m.selected >= offset+maxLines {
		offset = m.selected - maxLines + 1
	}
	if maxOffset := len(listLines) - maxLines; maxOffset >= 0 && offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	for i := 0; i < maxLines; i++ {
		left := ""
		listIdx := offset + i
		if listIdx < len(listLines) {
			left = listLines[listIdx]
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
		b.WriteString(dimStyle.Render("j/k: nav  gg/G: top/bottom  /: search  :: cmd  Enter/o: open  O: new win  E: editor  e: edit  d: del  f: fav  r: reload  R: refresh status  ?: help  q: quit"))
	case "command":
		b.WriteString(dimStyle.Render("Tab: complete  Enter: run  Esc: cancel"))
	case "open_editor":
		b.WriteString(dimStyle.Render("type editor name (Tab: cycle) | Enter: open | Esc: cancel"))
	}

	return b.String()
}

func formatGitList(p model.Project) string {
	if p.GitBranch == "" {
		return ""
	}
	parts := gitNonZeroParts(p)
	if len(parts) == 0 {
		return " [" + p.GitBranch + "]"
	}
	return " [" + p.GitBranch + " " + strings.Join(parts, " ") + "]"
}

func renderGitStatus(p model.Project, s string) string {
	if s == "" {
		return ""
	}
	if p.GitConflicted > 0 {
		return gitConflictStyle.Render(s)
	}
	if p.GitDirty > 0 {
		return gitDirtyStyle.Render(s)
	}
	return gitCleanStyle.Render(s)
}

func formatGitPreview(p model.Project) string {
	if p.GitBranch == "" {
		return ""
	}
	parts := gitNonZeroParts(p)
	if len(parts) == 0 {
		return p.GitBranch + " (clean)"
	}
	return p.GitBranch + "  " + strings.Join(parts, " ")
}

func gitNonZeroParts(p model.Project) []string {
	var parts []string
	if p.GitStaged > 0 {
		parts = append(parts, fmt.Sprintf("S:%d", p.GitStaged))
	}
	if p.GitModified > 0 {
		parts = append(parts, fmt.Sprintf("M:%d", p.GitModified))
	}
	if p.GitUntracked > 0 {
		parts = append(parts, fmt.Sprintf("?:%d", p.GitUntracked))
	}
	if p.GitDeleted > 0 {
		parts = append(parts, fmt.Sprintf("D:%d", p.GitDeleted))
	}
	if p.GitRenamed > 0 {
		parts = append(parts, fmt.Sprintf("R:%d", p.GitRenamed))
	}
	if p.GitConflicted > 0 {
		parts = append(parts, fmt.Sprintf("U:%d", p.GitConflicted))
	}
	return parts
}

func formatModified(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "-"
	}
	t := info.ModTime()
	if gitInfo, err := os.Stat(filepath.Join(path, ".git", "index")); err == nil {
		if gitInfo.ModTime().After(t) {
			t = gitInfo.ModTime()
		}
	}
	return t.Format("2006-01-02 15:04")
}

func formatGitHub(p model.Project) (string, bool) {
	if p.GHEnabled == nil || !*p.GHEnabled {
		return "", false
	}
	if p.GHOpenPRs == nil || p.GHOpenIssues == nil {
		return ghStyle.Render("PR:       -  Issues: -"), true
	}
	line := fmt.Sprintf("PR:       %d  Issues: %d", *p.GHOpenPRs, *p.GHOpenIssues)
	if p.GHFetchedAt != nil {
		line += "  (" + p.GHFetchedAt.Format("2006-01-02 15:04") + ")"
	}
	return ghStyle.Render(line), true
}

func (m Model) homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

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
