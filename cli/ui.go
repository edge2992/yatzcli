package cli

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/edge2992/yatzcli/engine"
)

func RunGame(client engine.GameClient, playerName string) error {
	m := newModel(client, playerName)
	p := tea.NewProgram(m)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
