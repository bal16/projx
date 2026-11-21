package tui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4"))
	favStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	gitCleanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	gitDirtyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	gitConflictStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	ghStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)
