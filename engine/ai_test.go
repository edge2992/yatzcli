package engine

import (
	"math/rand"
	"testing"
)

func TestAIPlayer_PlayTurn(t *testing.T) {
	g := NewGame([]string{"Human", "AI"}, rand.NewSource(42))
	ai := NewAIPlayer(g, "player-1")

	// Human plays first
	if err := g.Roll(); err != nil {
		t.Fatalf("human Roll() failed: %v", err)
	}
	if err := g.Score(Ones); err != nil {
		t.Fatalf("human Score() failed: %v", err)
	}
	if g.Players[g.Current].ID != "player-1" {
		t.Fatalf("expected AI's turn, got %s", g.Players[g.Current].ID)
	}

	// AI plays
	if err := ai.PlayTurn(); err != nil {
		t.Fatalf("AI PlayTurn() failed: %v", err)
	}

	// Should be back to human's turn
	if g.Players[g.Current].ID != "player-0" {
		t.Errorf("expected human's turn, got %s", g.Players[g.Current].ID)
	}
	if g.Round != 2 {
		t.Errorf("expected Round==2, got %d", g.Round)
	}

	// AI should have filled exactly one category
	filled := 0
	for _, c := range AllCategories {
		if g.Players[1].Scorecard.IsFilled(c) {
			filled++
		}
	}
	if filled != 1 {
		t.Errorf("expected AI to fill 1 category, got %d", filled)
	}
}

func TestAIPlayer_PlayTurn_NotMyTurn(t *testing.T) {
	g := NewGame([]string{"Human", "AI"}, rand.NewSource(42))
	ai := NewAIPlayer(g, "player-1")

	// It's human's turn (player-0), AI should fail
	err := ai.PlayTurn()
	if err == nil {
		t.Error("expected error when not AI's turn, got nil")
	}
}
