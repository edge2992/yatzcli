package cli

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/edge2992/yatzcli/engine"
)

type GameOption func(*model)

func WithChatChannel(ch <-chan ChatEntry) GameOption {
	return func(m *model) {
		m.chatCh = ch
	}
}

func WithStateUpdateChannel(ch <-chan *engine.GameState) GameOption {
	return func(m *model) {
		m.stateUpdateCh = ch
	}
}

func WithInitialWaiting() GameOption {
	return func(m *model) {
		m.state = stateWaiting
	}
}

func RunGame(client engine.GameClient, playerName string, opts ...GameOption) error {
	m := newModel(client, playerName)
	for _, opt := range opts {
		opt(&m)
	}
	p := tea.NewProgram(m)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
