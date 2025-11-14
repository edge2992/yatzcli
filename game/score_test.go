package game

import "testing"

// Helper function to create dice from values
func createDice(values []int) []Dice {
	dice := make([]Dice, len(values))
	for i, v := range values {
		dice[i] = Dice{Value: v}
	}
	return dice
}

func TestCalculateScore_Ones(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"Two ones", []int{1, 1, 2, 3, 4}, 2},
		{"No ones", []int{2, 3, 4, 5, 6}, 0},
		{"Five ones", []int{1, 1, 1, 1, 1}, 5},
		{"One one", []int{1, 2, 3, 4, 5}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, Ones)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_Sixes(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"Two sixes", []int{6, 6, 2, 3, 4}, 12},
		{"No sixes", []int{1, 2, 3, 4, 5}, 0},
		{"Five sixes", []int{6, 6, 6, 6, 6}, 30},
		{"One six", []int{1, 2, 3, 4, 6}, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, Sixes)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_ThreeOfAKind(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"Three threes", []int{3, 3, 3, 1, 2}, 12},
		{"No three of a kind", []int{1, 2, 3, 4, 5}, 0},
		{"Four ones", []int{1, 1, 1, 1, 5}, 9},
		{"Five sixes", []int{6, 6, 6, 6, 6}, 30},
		{"Three twos with high dice", []int{2, 2, 2, 5, 6}, 17},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, ThreeOfAKind)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_FourOfAKind(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"Four threes", []int{3, 3, 3, 3, 2}, 14},
		{"No four of a kind", []int{1, 2, 3, 4, 5}, 0},
		{"Three of a kind only", []int{1, 1, 1, 2, 3}, 0},
		{"Five ones", []int{1, 1, 1, 1, 1}, 5},
		{"Four sixes", []int{6, 6, 6, 6, 1}, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, FourOfAKind)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_FullHouse(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"Two twos and three threes", []int{2, 2, 3, 3, 3}, 25},
		{"Three ones and two sixes", []int{1, 1, 1, 6, 6}, 25},
		{"No full house", []int{1, 2, 3, 4, 5}, 0},
		{"Four of a kind", []int{1, 1, 1, 1, 2}, 0},
		{"Five of a kind", []int{5, 5, 5, 5, 5}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, FullHouse)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_SmallStraight(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"1-2-3-4", []int{1, 2, 3, 4, 6}, 30},
		{"2-3-4-5", []int{2, 3, 4, 5, 6}, 30},
		{"3-4-5-6", []int{3, 4, 5, 6, 1}, 30},
		{"1-2-3-4-5 (large straight)", []int{1, 2, 3, 4, 5}, 30},
		{"No straight", []int{1, 1, 3, 4, 6}, 0},
		{"1-3-4-5-6 (contains 3-4-5-6)", []int{1, 3, 4, 5, 6}, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, SmallStraight)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_LargeStraight(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"1-2-3-4-5", []int{1, 2, 3, 4, 5}, 40},
		{"2-3-4-5-6", []int{2, 3, 4, 5, 6}, 40},
		{"Small straight only", []int{1, 2, 3, 4, 6}, 0},
		{"No straight", []int{1, 1, 3, 4, 6}, 0},
		{"1-2-3-4-4 (duplicate)", []int{1, 2, 3, 4, 4}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, LargeStraight)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_Yahtzee(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"Five ones", []int{1, 1, 1, 1, 1}, 50},
		{"Five sixes", []int{6, 6, 6, 6, 6}, 50},
		{"Four of a kind", []int{3, 3, 3, 3, 2}, 0},
		{"No Yahtzee", []int{1, 2, 3, 4, 5}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, Yahtzee)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_Chance(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected int
	}{
		{"All ones", []int{1, 1, 1, 1, 1}, 5},
		{"All sixes", []int{6, 6, 6, 6, 6}, 30},
		{"Mixed dice", []int{1, 2, 3, 4, 5}, 15},
		{"High roll", []int{5, 6, 6, 6, 6}, 29},
		{"Low roll", []int{1, 1, 2, 2, 3}, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			score := CalculateScore(dice, Chance)
			if score != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCalculateScore_InvalidCategory(t *testing.T) {
	dice := createDice([]int{1, 2, 3, 4, 5})
	score := CalculateScore(dice, ScoreCategory("invalid_category"))
	if score != 0 {
		t.Errorf("Expected 0 for invalid category, got %d", score)
	}
}

func TestCalculateTotalScore(t *testing.T) {
	tests := []struct {
		name     string
		scoreCard ScoreCard
		expected int
	}{
		{
			"Empty scorecard",
			ScoreCard{Scores: make(map[ScoreCategory]int)},
			0,
		},
		{
			"Upper section only (no bonus)",
			ScoreCard{Scores: map[ScoreCategory]int{
				Ones:   3,
				Twos:   6,
				Threes: 9,
				Fours:  12,
				Fives:  15,
				Sixes:  17,
			}},
			62, // Below 63, no bonus
		},
		{
			"Upper section with bonus",
			ScoreCard{Scores: map[ScoreCategory]int{
				Ones:   4,
				Twos:   8,
				Threes: 12,
				Fours:  12,
				Fives:  15,
				Sixes:  18,
			}},
			104, // 69 + 35 bonus
		},
		{
			"Full scorecard with Yahtzee",
			ScoreCard{Scores: map[ScoreCategory]int{
				Ones:          5,
				Twos:          10,
				Threes:        15,
				Fours:         20,
				Fives:         25,
				Sixes:         30,
				ThreeOfAKind:  20,
				FourOfAKind:   25,
				FullHouse:     25,
				SmallStraight: 30,
				LargeStraight: 40,
				Yahtzee:       50,
				Chance:        30,
			}},
			360, // 105 (upper) + 35 (bonus) + 220 (lower)
		},
		{
			"Lower section only",
			ScoreCard{Scores: map[ScoreCategory]int{
				Yahtzee:       50,
				LargeStraight: 40,
				SmallStraight: 30,
				FullHouse:     25,
			}},
			145,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := CalculateTotalScore(tt.scoreCard)
			if total != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, total)
			}
		})
	}
}

func TestCountDiceValues(t *testing.T) {
	tests := []struct {
		name     string
		dice     []int
		expected []int
	}{
		{"All ones", []int{1, 1, 1, 1, 1}, []int{5, 0, 0, 0, 0, 0}},
		{"All different", []int{1, 2, 3, 4, 5}, []int{1, 1, 1, 1, 1, 0}},
		{"Two pairs", []int{2, 2, 5, 5, 6}, []int{0, 2, 0, 0, 2, 1}},
		{"Three of a kind", []int{3, 3, 3, 1, 6}, []int{1, 0, 3, 0, 0, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := createDice(tt.dice)
			counts := countDiceValues(dice)

			for i, expected := range tt.expected {
				if counts[i] != expected {
					t.Errorf("At index %d: expected %d, got %d", i, expected, counts[i])
				}
			}
		})
	}
}
