package engine

import (
	"math/rand"
	"testing"
)

func newTestGame() *Game {
	return NewGame([]string{"Alice", "Bob"}, rand.NewSource(42))
}

func TestNewGame(t *testing.T) {
	g := newTestGame()
	if g.Phase != PhaseRolling {
		t.Errorf("expected PhaseRolling, got %d", g.Phase)
	}
	if len(g.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(g.Players))
	}
	if g.Round != 1 {
		t.Errorf("expected Round==1, got %d", g.Round)
	}
	if g.Current != 0 {
		t.Errorf("expected Current==0, got %d", g.Current)
	}
	if g.Players[0].Name != "Alice" {
		t.Errorf("expected Alice, got %s", g.Players[0].Name)
	}
	if g.Players[1].Name != "Bob" {
		t.Errorf("expected Bob, got %s", g.Players[1].Name)
	}
}

func TestGame_Roll(t *testing.T) {
	g := newTestGame()
	if err := g.Roll(); err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	if g.RollCount != 1 {
		t.Errorf("expected RollCount==1, got %d", g.RollCount)
	}
	for i, d := range g.Dice {
		if d < 1 || d > 6 {
			t.Errorf("dice[%d] out of range: %d", i, d)
		}
	}
}

func TestGame_Roll_NotInitialRoll(t *testing.T) {
	g := newTestGame()
	if err := g.Roll(); err != nil {
		t.Fatalf("first Roll() failed: %v", err)
	}
	err := g.Roll()
	if err == nil {
		t.Error("expected error on second Roll(), got nil")
	}
}

func TestGame_Hold(t *testing.T) {
	g := newTestGame()
	if err := g.Roll(); err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	held := []int{0, 2, 4}
	savedDice := g.Dice
	if err := g.Hold(held); err != nil {
		t.Fatalf("Hold() failed: %v", err)
	}
	if g.RollCount != 2 {
		t.Errorf("expected RollCount==2, got %d", g.RollCount)
	}
	for _, idx := range held {
		if g.Dice[idx] != savedDice[idx] {
			t.Errorf("held dice[%d] changed: %d -> %d", idx, savedDice[idx], g.Dice[idx])
		}
	}
}

func TestGame_Hold_BeforeRoll(t *testing.T) {
	g := newTestGame()
	err := g.Hold([]int{0, 1})
	if err == nil {
		t.Error("expected error on Hold() before Roll(), got nil")
	}
}

func TestGame_Hold_MaxRolls(t *testing.T) {
	g := newTestGame()
	if err := g.Roll(); err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	if err := g.Hold([]int{0}); err != nil {
		t.Fatalf("Hold() 1 failed: %v", err)
	}
	if err := g.Hold([]int{0}); err != nil {
		t.Fatalf("Hold() 2 failed: %v", err)
	}
	if g.RollCount != MaxRolls {
		t.Errorf("expected RollCount==%d, got %d", MaxRolls, g.RollCount)
	}
	if g.Phase != PhaseChoosing {
		t.Errorf("expected PhaseChoosing, got %d", g.Phase)
	}
	err := g.Hold([]int{0})
	if err == nil {
		t.Error("expected error on Hold() after max rolls, got nil")
	}
}

func TestGame_Roll_AfterMaxRolls(t *testing.T) {
	g := newTestGame()
	_ = g.Roll()
	_ = g.Hold([]int{0})
	_ = g.Hold([]int{0})
	err := g.Roll()
	if err == nil {
		t.Error("expected error on Roll() after PhaseChoosing, got nil")
	}
}

func TestGame_Score(t *testing.T) {
	g := newTestGame()
	if err := g.Roll(); err != nil {
		t.Fatalf("Roll() failed: %v", err)
	}
	if err := g.Score(Ones); err != nil {
		t.Fatalf("Score() failed: %v", err)
	}
	if g.Players[0].Scorecard.IsFilled(Ones) != true {
		t.Error("expected Ones to be filled for player 0")
	}
	if g.Current != 1 {
		t.Errorf("expected Current==1, got %d", g.Current)
	}
	if g.Phase != PhaseRolling {
		t.Errorf("expected PhaseRolling, got %d", g.Phase)
	}
	if g.RollCount != 0 {
		t.Errorf("expected RollCount==0, got %d", g.RollCount)
	}
}

func TestGame_Score_BeforeRoll(t *testing.T) {
	g := newTestGame()
	err := g.Score(Ones)
	if err == nil {
		t.Error("expected error on Score() before Roll(), got nil")
	}
}

func TestGame_Score_AlreadyFilled(t *testing.T) {
	g := newTestGame()
	_ = g.Roll()
	_ = g.Score(Ones) // fills Ones for player 0, advances to player 1

	// Player 1's turn
	_ = g.Roll()
	_ = g.Score(Ones) // fills Ones for player 1, advances to player 0 round 2

	// Player 0's turn again
	_ = g.Roll()
	err := g.Score(Ones) // already filled
	if err == nil {
		t.Error("expected error on Score() for already filled category, got nil")
	}
}

func TestGame_FullGame(t *testing.T) {
	g := NewGame([]string{"Solo"}, rand.NewSource(99))
	for i, cat := range AllCategories {
		if g.Phase == PhaseFinished {
			t.Fatalf("game finished early at round %d", i+1)
		}
		if err := g.Roll(); err != nil {
			t.Fatalf("Roll() round %d failed: %v", i+1, err)
		}
		if err := g.Score(cat); err != nil {
			t.Fatalf("Score(%s) round %d failed: %v", cat, i+1, err)
		}
	}
	if g.Phase != PhaseFinished {
		t.Errorf("expected PhaseFinished, got %d", g.Phase)
	}
}

func TestGame_GetState(t *testing.T) {
	g := newTestGame()
	_ = g.Roll()
	state := g.GetState()
	if state.CurrentPlayer != g.Players[0].ID {
		t.Errorf("expected current player %s, got %s", g.Players[0].ID, state.CurrentPlayer)
	}
	if state.CurrentPlayerIndex != 0 {
		t.Errorf("expected index 0, got %d", state.CurrentPlayerIndex)
	}
	if state.Round != 1 {
		t.Errorf("expected round 1, got %d", state.Round)
	}
	if state.RollCount != 1 {
		t.Errorf("expected RollCount 1, got %d", state.RollCount)
	}
	if state.Phase != PhaseRolling {
		t.Errorf("expected PhaseRolling, got %d", state.Phase)
	}
	if len(state.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(state.Players))
	}
	if state.Dice != g.Dice {
		t.Errorf("dice mismatch")
	}
	if len(state.AvailableCategories) != 13 {
		t.Errorf("expected 13 available categories, got %d", len(state.AvailableCategories))
	}
}

func TestGame_GetAvailableCategories(t *testing.T) {
	g := newTestGame()
	cats := g.GetAvailableCategories()
	if len(cats) != 13 {
		t.Errorf("expected 13 categories, got %d", len(cats))
	}
}
