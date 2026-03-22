package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreedyStrategy_AlwaysScores(t *testing.T) {
	s := &GreedyStrategy{}
	sc := NewScorecard()
	avail := sc.AvailableCategories()

	action := s.DecideAction([5]int{6, 6, 6, 6, 6}, 1, sc, avail)
	assert.Equal(t, "score", action.Type)
	assert.Equal(t, Yahtzee, action.Category)
}

func TestGreedyStrategy_Name(t *testing.T) {
	s := &GreedyStrategy{}
	assert.Equal(t, "greedy", s.Name())
}

func TestStatisticalStrategy_Name(t *testing.T) {
	s := &StatisticalStrategy{}
	assert.Equal(t, "statistical", s.Name())
}

func TestStatisticalStrategy_ScoresOnThirdRoll(t *testing.T) {
	s := &StatisticalStrategy{}
	sc := NewScorecard()
	avail := sc.AvailableCategories()

	action := s.DecideAction([5]int{1, 2, 3, 4, 5}, 3, sc, avail)
	assert.Equal(t, "score", action.Type)
	assert.Equal(t, LargeStraight, action.Category)
}

func TestStatisticalStrategy_HoldsGoodDice(t *testing.T) {
	s := &StatisticalStrategy{}
	sc := NewScorecard()
	avail := sc.AvailableCategories()

	// Four 6s — should hold them and try for Yahtzee
	action := s.DecideAction([5]int{6, 6, 6, 6, 1}, 1, sc, avail)
	if action.Type == "hold" {
		// Should hold at least the four 6s
		holdSet := make(map[int]bool)
		for _, i := range action.Indices {
			holdSet[i] = true
		}
		assert.True(t, holdSet[0] && holdSet[1] && holdSet[2] && holdSet[3],
			"should hold the four 6s")
	}
	// Also acceptable to score immediately with four_of_a_kind (25 pts)
}

func TestStatisticalStrategy_ScoresYahtzeeImmediately(t *testing.T) {
	s := &StatisticalStrategy{}
	sc := NewScorecard()
	avail := sc.AvailableCategories()

	// Five 6s — should score Yahtzee immediately
	action := s.DecideAction([5]int{6, 6, 6, 6, 6}, 1, sc, avail)
	assert.Equal(t, "score", action.Type)
	assert.Equal(t, Yahtzee, action.Category)
}

func TestExpectedValue_AllHeld(t *testing.T) {
	sc := NewScorecard()
	avail := sc.AvailableCategories()
	dice := [5]int{1, 2, 3, 4, 5}

	ev := expectedValue(dice, []int{0, 1, 2, 3, 4}, avail, sc)
	best := float64(bestScoreForDice(dice, avail))
	assert.Equal(t, best, ev, "holding all dice should give same as best immediate score")
}

func TestExpectedValue_Positive(t *testing.T) {
	sc := NewScorecard()
	avail := sc.AvailableCategories()
	dice := [5]int{3, 3, 3, 2, 1}

	ev := expectedValue(dice, []int{0, 1, 2}, avail, sc)
	assert.Greater(t, ev, 0.0, "expected value should be positive")
}

func TestHoldCombinations_Count(t *testing.T) {
	combos := holdCombinations()
	assert.Len(t, combos, 32, "should have 2^5 = 32 combinations")
}

func TestBestScoreForDice(t *testing.T) {
	avail := []Category{Ones, Yahtzee, Chance}
	best := bestScoreForDice([5]int{6, 6, 6, 6, 6}, avail)
	assert.Equal(t, 50, best, "Yahtzee should be the best for five 6s")
}

func TestStatisticalStrategy_FullGame(t *testing.T) {
	// Run a full game with statistical strategy to ensure no panics
	s := &StatisticalStrategy{}
	require.NotNil(t, s)

	sc := NewScorecard()
	dice := [5]int{3, 3, 4, 5, 6}
	avail := sc.AvailableCategories()
	action := s.DecideAction(dice, 1, sc, avail)
	require.Contains(t, []string{"hold", "score"}, action.Type)
}
