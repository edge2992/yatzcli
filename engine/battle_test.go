package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunBattle_TwoPlayers(t *testing.T) {
	turnCount := 0
	state, err := RunBattle(BattleConfig{
		Players: []BattlePlayer{
			{Name: "Greedy", Strategy: &GreedyStrategy{}},
			{Name: "Statistical", Strategy: &StatisticalStrategy{}},
		},
		Seed: 42,
		OnTurnDone: func(result AITurnResult) {
			turnCount++
		},
	})
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, PhaseFinished, state.Phase)
	assert.Equal(t, 26, turnCount, "13 rounds × 2 players = 26 turns")
}

func TestRunBattle_RequiresTwoPlayers(t *testing.T) {
	_, err := RunBattle(BattleConfig{
		Players: []BattlePlayer{
			{Name: "Solo", Strategy: &GreedyStrategy{}},
		},
	})
	assert.Error(t, err)
}

func TestRunBattle_ThreePlayers(t *testing.T) {
	turnCount := 0
	state, err := RunBattle(BattleConfig{
		Players: []BattlePlayer{
			{Name: "A", Strategy: &GreedyStrategy{}},
			{Name: "B", Strategy: &StatisticalStrategy{}},
			{Name: "C", Strategy: &GreedyStrategy{}},
		},
		Seed: 123,
		OnTurnDone: func(result AITurnResult) {
			turnCount++
		},
	})
	require.NoError(t, err)
	assert.Equal(t, PhaseFinished, state.Phase)
	assert.Equal(t, 39, turnCount, "13 rounds × 3 players = 39 turns")
}

func TestRunBattle_AllCategoriesFilled(t *testing.T) {
	state, err := RunBattle(BattleConfig{
		Players: []BattlePlayer{
			{Name: "A", Strategy: &GreedyStrategy{}},
			{Name: "B", Strategy: &StatisticalStrategy{}},
		},
		Seed: 42,
	})
	require.NoError(t, err)

	for _, p := range state.Players {
		for _, cat := range AllCategories {
			assert.True(t, p.Scorecard.IsFilled(cat),
				"player %s should have %s filled", p.Name, cat)
		}
	}
}
