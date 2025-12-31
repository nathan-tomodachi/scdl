package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(ctx context.Context, cfg Config) error {
	m := newModel(ctx, cfg)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	_, err := prog.Run()
	return err
}
