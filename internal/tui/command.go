package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

var cmdSuggestionsBase = []string{
	"add", "config", "doctor", "export", "favorite", "group", "help",
	"import", "list", "open", "profiles", "refresh", "remove", "rename",
	"scan", "set", "tag", "tui", "untag", "version", "ws",
}

// CommandPalette manages command-mode suggestions and completion.
type CommandPalette struct {
	sugs []string
	sel  int
}

// Reset clears suggestions.
func (c *CommandPalette) Reset() {
	c.sugs = nil
	c.sel = 0
}

// UpdateSuggestions recomputes suggestions for the current input.
func (c *CommandPalette) UpdateSuggestions(val string) {
	if strings.Contains(val, " ") {
		c.sugs = nil
		c.sel = 0
		return
	}
	prefix := strings.ToLower(strings.TrimSpace(val))
	if prefix == "" {
		c.sugs = append([]string(nil), cmdSuggestionsBase...)
		c.sel = 0
		return
	}
	var out []string
	for _, cmd := range cmdSuggestionsBase {
		if strings.HasPrefix(cmd, prefix) {
			out = append(out, cmd)
		}
	}
	c.sugs = out
	c.sel = 0
}

// CompleteNext inserts the selected suggestion and advances selection.
func (c *CommandPalette) CompleteNext(input *textinput.Model) {
	if len(c.sugs) == 0 {
		return
	}
	input.SetValue(c.sugs[c.sel])
	input.CursorEnd()
	c.sel = (c.sel + 1) % len(c.sugs)
}

// CompletePrev moves selection back and inserts it.
func (c *CommandPalette) CompletePrev(input *textinput.Model) {
	if len(c.sugs) == 0 {
		return
	}
	c.sel = (c.sel - 1 + len(c.sugs)) % len(c.sugs)
	input.SetValue(c.sugs[c.sel])
	input.CursorEnd()
}

// PreviewLines renders the suggestions panel.
func (c CommandPalette) PreviewLines(currentInput string) []string {
	lines := []string{headerStyle.Render("Commands")}
	if len(c.sugs) == 0 {
		lines = append(lines, dimStyle.Render("  (no matches)"))
		return lines
	}
	hi := c.sel
	cur := strings.TrimSpace(currentInput)
	for i, s := range c.sugs {
		if s == cur {
			hi = i
			break
		}
	}
	for i, s := range c.sugs {
		if i == hi {
			lines = append(lines, selStyle.Render("> "+s))
		} else {
			lines = append(lines, "  "+s)
		}
	}
	return lines
}
