package engine

import (
	"math/rand"
	"testing"
)

func TestLocalClient_Roll(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	c := NewLocalClient(g, "player-0", nil)

	state, err := c.Roll()
	if err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	if state.RollCount != 1 {
		t.Errorf("expected RollCount==1, got %d", state.RollCount)
	}
	for i, d := range state.Dice {
		if d < 1 || d > 6 {
			t.Errorf("dice[%d] out of range: %d", i, d)
		}
	}
}

func TestLocalClient_Hold(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	c := NewLocalClient(g, "player-0", nil)

	state, err := c.Roll()
	if err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	savedDice := state.Dice

	state, err = c.Hold([]int{0, 2, 4})
	if err != nil {
		t.Fatalf("Hold() failed: %v", err)
	}
	if state.RollCount != 2 {
		t.Errorf("expected RollCount==2, got %d", state.RollCount)
	}
	for _, idx := range []int{0, 2, 4} {
		if state.Dice[idx] != savedDice[idx] {
			t.Errorf("held dice[%d] changed: %d -> %d", idx, savedDice[idx], state.Dice[idx])
		}
	}
}

func TestLocalClient_Score(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	c := NewLocalClient(g, "player-0", nil)

	if _, err := c.Roll(); err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	state, err := c.Score(Ones)
	if err != nil {
		t.Fatalf("Score() failed: %v", err)
	}
	if !state.Players[0].Scorecard.IsFilled(Ones) {
		t.Error("expected Ones to be filled")
	}
	if state.Round != 2 {
		t.Errorf("expected Round==2, got %d", state.Round)
	}
}

func TestLocalClient_ScoreTriggersAI(t *testing.T) {
	g := NewGame([]string{"Human", "AI"}, rand.NewSource(42))
	ai := NewAIPlayer(g, "player-1")
	c := NewLocalClient(g, "player-0", []*AIPlayer{ai})

	// Human rolls and scores
	if _, err := c.Roll(); err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	state, err := c.Score(Ones)
	if err != nil {
		t.Fatalf("Score() failed: %v", err)
	}

	// After scoring, AI should have played automatically
	if state.CurrentPlayer != "player-0" {
		t.Errorf("expected human's turn, got %s", state.CurrentPlayer)
	}
	if state.Round != 2 {
		t.Errorf("expected Round==2, got %d", state.Round)
	}

	// AI should have filled one category
	filled := 0
	for _, cat := range AllCategories {
		if state.Players[1].Scorecard.IsFilled(cat) {
			filled++
		}
	}
	if filled != 1 {
		t.Errorf("expected AI to fill 1 category, got %d", filled)
	}
}

func TestLocalClient_RunAITurns_BreaksForRemoteHuman(t *testing.T) {
	// Simulate P2P scenario: two human players, no AIs.
	// After player-0 scores, it becomes player-1's turn.
	// runAITurns must return immediately (not loop forever).
	g := NewGame([]string{"Host", "Guest"}, rand.NewSource(42))
	c := NewLocalClient(g, "player-0", nil)

	if _, err := c.Roll(); err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	state, err := c.Score(Ones)
	if err != nil {
		t.Fatalf("Score() failed: %v", err)
	}

	// Current player should be the guest (player-1), not stuck in a loop.
	if state.CurrentPlayer != "player-1" {
		t.Errorf("expected player-1's turn, got %s", state.CurrentPlayer)
	}
}

func TestLocalClient_GetState(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	c := NewLocalClient(g, "player-0", nil)

	state, err := c.GetState()
	if err != nil {
		t.Fatalf("GetState() failed: %v", err)
	}
	if state.CurrentPlayer != "player-0" {
		t.Errorf("expected player-0, got %s", state.CurrentPlayer)
	}
	if state.Phase != PhaseRolling {
		t.Errorf("expected PhaseRolling, got %d", state.Phase)
	}
	if state.Round != 1 {
		t.Errorf("expected Round==1, got %d", state.Round)
	}
}
