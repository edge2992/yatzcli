package engine

import (
	"fmt"
	"math/rand"
	"time"
)

// BattlePlayer represents a player in a battle.
type BattlePlayer struct {
	Name     string
	Strategy Strategy
}

// BattleConfig holds the configuration for a battle.
type BattleConfig struct {
	Players    []BattlePlayer
	Seed       int64
	OnTurnDone func(result AITurnResult)
}

// BattleResult holds the final results of a battle.
type BattleResult struct {
	Players []BattlePlayerResult
}

// BattlePlayerResult holds the final result for a single player.
type BattlePlayerResult struct {
	Name  string
	Score int
}

// RunBattle executes a full AI-vs-AI game and returns the final state.
func RunBattle(cfg BattleConfig) (*GameState, error) {
	if len(cfg.Players) < 2 {
		return nil, fmt.Errorf("battle requires at least 2 players, got %d", len(cfg.Players))
	}

	var src rand.Source
	if cfg.Seed != 0 {
		src = rand.NewSource(cfg.Seed)
	} else {
		src = rand.NewSource(time.Now().UnixNano())
	}

	names := make([]string, len(cfg.Players))
	for i, p := range cfg.Players {
		names[i] = p.Name
	}

	game := NewGame(names, src)

	ais := make([]*AIPlayer, len(cfg.Players))
	for i, p := range cfg.Players {
		pid := fmt.Sprintf("player-%d", i)
		ais[i] = NewAIPlayerWithStrategy(game, pid, p.Strategy)
	}

	for game.Phase != PhaseFinished {
		current := game.Current
		result, err := ais[current].PlayTurn()
		if err != nil {
			return nil, fmt.Errorf("player %s turn failed: %w", cfg.Players[current].Name, err)
		}
		if cfg.OnTurnDone != nil {
			cfg.OnTurnDone(result)
		}
	}

	state := game.GetState()
	return &state, nil
}
