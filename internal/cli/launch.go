package cli

import (
	"os"
	"os/exec"

	"github.com/bal16/projx/internal/config"
)

// tuiEditors lists editors that need a terminal emulator to be visible when
// launched detached (without a TTY, e.g. from a desktop launcher).
var tuiEditors = map[string]bool{
	"nvim": true, "neovim": true, "vim": true, "vi": true,
	"helix": true, "hx": true, "kak": true,
	"nano": true, "micro": true, "emacs": true,
}

// terminalPreArgs maps known terminal emulators to the arguments required
// before the program they should run.
var terminalPreArgs = map[string][]string{
	"kitty":          {},
	"foot":           {},
	"wezterm":        {"start", "--"},
	"alacritty":      {"-e"},
	"ghostty":        {"-e"},
	"konsole":        {"-e"},
	"gnome-terminal": {"--"},
	"xterm":          {"-e"},
	"uxterm":         {"-e"},
	"st":             {"-e"},
}

// terminalCandidates is the auto-detect order for terminal emulators.
var terminalCandidates = []string{"kitty", "ghostty", "alacritty", "foot", "wezterm", "konsole", "gnome-terminal", "xterm"}

// terminalCommand resolves the terminal emulator to use: the configured one
// (Config.Terminal) if present, otherwise the first known one on PATH.
func terminalCommand(cfg *config.Config) (bin string, preArgs []string, ok bool) {
	if t := cfg.Terminal; t != "" {
		if _, err := exec.LookPath(t); err != nil {
			return "", nil, false
		}
		if pre, known := terminalPreArgs[t]; known {
			return t, pre, true
		}
		// Configured but unknown to us: assume it runs the program directly.
		return t, nil, true
	}
	for _, t := range terminalCandidates {
		if _, err := exec.LookPath(t); err == nil {
			return t, terminalPreArgs[t], true
		}
	}
	return "", nil, false
}

// startEditor launches bin with argv, wrapping TUI editors in a terminal
// emulator so they work when there is no controlling TTY.
func startEditor(cfg *config.Config, ed, bin string, argv []string) error {
	args := argv
	if tuiEditors[bin] || tuiEditors[ed] {
		if term, pre, ok := terminalCommand(cfg); ok {
			wrapped := append([]string{}, pre...)
			wrapped = append(wrapped, bin)
			wrapped = append(wrapped, argv...)
			bin, args = term, wrapped
		}
	}
	c := exec.Command(bin, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Start(); err != nil {
		return err
	}
	_ = c.Process.Release()
	return nil
}
