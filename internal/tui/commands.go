package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bal16/projx/internal/db"
)

func loadProjectsCmd(repo *db.Repository) tea.Cmd {
	return func() tea.Msg {
		projects, err := repo.List("", "", false)
		if err != nil {
			return errMsg{err}
		}
		return projectsMsg{projects}
	}
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

func runProfilesListCmd() tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		c := exec.Command(exe, "profiles", "list", "--json")
		out, _ := c.CombinedOutput()
		return commandResultMsg{cmd: "profiles list", output: string(out)}
	}
}
