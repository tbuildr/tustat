package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the interactive TUI. Call it only with an interactive
// terminal on stdin and stdout.
func Run(cfg Config) error {
	p := tea.NewProgram(newModel(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
