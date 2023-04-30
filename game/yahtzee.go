package game

import (
	"fmt"
	"math/rand"
	"strconv"
)

// ---------- Constants and Types ----------
const (
	NumberOfDice   = 5
	MaxRolls       = 3
	NumberOfRounds = 13
)

type ScoreCategory string

const (
	Ones          ScoreCategory = "Ones"
	Twos          ScoreCategory = "Twos"
	Threes        ScoreCategory = "Threes"
	Fours         ScoreCategory = "Fours"
	Fives         ScoreCategory = "Fives"
	Sixes         ScoreCategory = "Sixes"
	ThreeOfAKind  ScoreCategory = "ThreeOfAKind"
	FourOfAKind   ScoreCategory = "FourOfAKind"
	FullHouse     ScoreCategory = "FullHouse"
	SmallStraight ScoreCategory = "SmallStraight"
	LargeStraight ScoreCategory = "LargeStraight"
	Yahtzee       ScoreCategory = "Yahtzee"
	Chance        ScoreCategory = "Chance"
)

var AllCategories = []ScoreCategory{
	Ones, Twos, Threes, Fours, Fives, Sixes,
	ThreeOfAKind, FourOfAKind, FullHouse, SmallStraight, LargeStraight, Yahtzee, Chance,
}

type Dice struct {
	Value int
	Held  bool
}

type ScoreCard struct {
	Scores map[ScoreCategory]int
	Filled map[ScoreCategory]bool
}

// ---------- Initialization Functions ----------
func NewScoreCard() ScoreCard {
	scoreCard := ScoreCard{
		Scores: make(map[ScoreCategory]int),
		Filled: make(map[ScoreCategory]bool),
	}
	for _, category := range AllCategories {
		scoreCard.Scores[category] = 0
		scoreCard.Filled[category] = false
	}
	return scoreCard
}

// ---------- Gameplay Functions ----------

func RollDice(dice []Dice) {
	for i := range dice {
		if !dice[i].Held {
			dice[i].Value = rand.Intn(6) + 1
		}
	}
}

func HoldDice(dice []Dice, selectedIndices []int) {
	for _, index := range selectedIndices {
		if index >= 0 && index < len(dice) {
			dice[index].Held = true
		}
	}
}

func CategoryWithScore(dice []Dice, categories []string) []string {
	// Returns a list of categories with their score
	options := make([]string, len(categories))
	for i, cat := range categories {
		score := CalculateScore(dice, ScoreCategory(cat))
		options[i] = cat + "\t(" + strconv.Itoa(score) + ")"
	}
	return options
}

// ---------- Display Functions ----------
func DisplayFinalScores(players []PlayerInfo) {
	fmt.Println("\nFinal Scores:")
	for _, player := range players {
		fmt.Printf("%s: %d\n", player.Name, CalculateTotalScore(player.ScoreCard))
	}
}
