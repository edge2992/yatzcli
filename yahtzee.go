package main

import (
	"fmt"
	"math/rand"
)

// ---------- Constants and Types ----------

type ScoreCategory string

const (
	Ones          ScoreCategory = "Ones"
	Twos                        = "Twos"
	Threes                      = "Threes"
	Fours                       = "Fours"
	Fives                       = "Fives"
	Sixes                       = "Sixes"
	ThreeOfAKind                = "ThreeOfAKind"
	FourOfAKind                 = "FourOfAKind"
	FullHouse                   = "FullHouse"
	SmallStraight               = "SmallStraight"
	LargeStraight               = "LargeStraight"
	Yahtzee                     = "Yahtzee"
	Chance                      = "Chance"
)

const (
	NumberOfDice   = 5
	MaxRolls       = 3
	NumberOfRounds = 13
)

type Player struct {
	Name      string
	ScoreCard ScoreCard
}

type Dice struct {
	Value int
	Held  bool
}

type ScoreCard struct {
	Scores map[ScoreCategory]*int
}

// ---------- Initialization Functions ----------
func NewScoreCard() ScoreCard {
	scoreCard := ScoreCard{
		Scores: map[ScoreCategory]*int{
			Ones:          nil,
			Twos:          nil,
			Threes:        nil,
			Fours:         nil,
			Fives:         nil,
			Sixes:         nil,
			ThreeOfAKind:  nil,
			FourOfAKind:   nil,
			FullHouse:     nil,
			SmallStraight: nil,
			LargeStraight: nil,
			Yahtzee:       nil,
			Chance:        nil,
		},
	}
	return scoreCard
}

func createPlayers() []Player {
	players := make([]Player, 2)
	for i := 0; i < 2; i++ {
		var name string
		fmt.Printf("Enter name for player %d: ", i+1)
		fmt.Scanln(&name)
		players[i] = Player{Name: name, ScoreCard: NewScoreCard()}
	}
	return players
}

func createGameState(players []Player) map[string]*Player {
	gameState := make(map[string]*Player)
	for i := range players {
		gameState[players[i].Name] = &players[i]
	}
	return gameState
}

// ---------- Gameplay Functions ----------
func playTurn(player *Player) {
	dice := make([]Dice, NumberOfDice)
	rollDice(dice)
	for rolls := 1; rolls < MaxRolls; rolls++ {
		displayDice(dice)
		holdInput := getPlayerHoldInput()
		holdDice(dice, holdInput)
		rollDice(dice)
	}
	displayDice(dice)

	category := chooseCategory(player)
	score := calculateScore(dice, category)
	player.ScoreCard.Scores[category] = &score
}

func rollDice(dice []Dice) {
	for i := range dice {
		if !dice[i].Held {
			dice[i].Value = rand.Intn(6) + 1
		}
	}
}

func holdDice(dice []Dice, holdInput string) {
	for _, ch := range holdInput {
		index := int(ch - '1')
		if index >= 0 && index < len(dice) {
			dice[index].Held = !dice[index].Held
		}
	}
}

func displayDice(dice []Dice) {
	fmt.Print("Dice: ")
	for i := range dice {
		fmt.Printf("%d ", dice[i].Value)
	}
	fmt.Println()
}

func chooseCategory(player *Player) ScoreCategory {
	var category string
	displayAvailableCategories(player)

	fmt.Print("Enter the category you want to choose: ")
	for {
		fmt.Scanln(&category)
		cat := ScoreCategory(category)
		if _, ok := player.ScoreCard.Scores[cat]; ok && player.ScoreCard.Scores[cat] == nil {
			return cat
		}
		fmt.Println("Invalid category or already scored. Please enter a valid category:")
	}
}

func displayAvailableCategories(player *Player) {
	fmt.Println("Available categories:")
	for cat, score := range player.ScoreCard.Scores {
		if score == nil {
			fmt.Printf("%s ", cat)
		}
	}
	fmt.Println()
}

// ---------- Scoring Functions ----------
func getPlayerHoldInput() string {
	var holdInput string
	fmt.Print("Enter the dice you want to hold (e.g. 123): ")
	fmt.Scanln(&holdInput)
	return holdInput
}

func calculateScore(dice []Dice, category ScoreCategory) int {
	diceCounts := countDiceValues(dice)

	switch category {
	case Ones:
		return diceCounts[0] * 1
	case Twos:
		return diceCounts[1] * 2
	case Threes:
		return diceCounts[2] * 3
	case Fours:
		return diceCounts[3] * 4
	case Fives:
		return diceCounts[4] * 5
	case Sixes:
		return diceCounts[5] * 6
	case ThreeOfAKind:
		for i, count := range diceCounts {
			if count >= 3 {
				return (i + 1) * 3
			}
		}
	case FourOfAKind:
		for i, count := range diceCounts {
			if count >= 4 {
				return (i + 1) * 4
			}
		}
	case FullHouse:
		hasThree := false
		hasTwo := false
		for _, count := range diceCounts {
			if count == 2 {
				hasTwo = true
			} else if count == 3 {
				hasThree = true
			}
		}
		if hasTwo && hasThree {
			return 25
		}
	case SmallStraight:
		if (diceCounts[0] > 0 && diceCounts[1] > 0 && diceCounts[2] > 0 && diceCounts[3] > 0) ||
			(diceCounts[1] > 0 && diceCounts[2] > 0 && diceCounts[3] > 0 && diceCounts[4] > 0) ||
			(diceCounts[2] > 0 && diceCounts[3] > 0 && diceCounts[4] > 0 && diceCounts[5] > 0) {
			return 30
		}
	case LargeStraight:
		if (diceCounts[0] == 1 && diceCounts[1] == 1 && diceCounts[2] == 1 && diceCounts[3] == 1 && diceCounts[4] == 1) ||
			(diceCounts[1] == 1 && diceCounts[2] == 1 && diceCounts[3] == 1 && diceCounts[4] == 1 && diceCounts[5] == 1) {
			return 40
		}
	case Yahtzee:
		for _, count := range diceCounts {
			if count == 5 {
				return 50
			}
		}
	case Chance:
		total := 0
		for i, count := range diceCounts {
			total += (i + 1) * count
		}
		return total
	}

	return 0
}

func countDiceValues(dice []Dice) []int {
	counts := make([]int, 6)
	for _, die := range dice {
		counts[die.Value-1]++
	}
	return counts
}

func calculateTotalScore(scoreCard ScoreCard) int {
	total := 0
	for _, score := range scoreCard.Scores {
		if score != nil {
			total += *score
		}
	}
	return total
}

// ---------- Display Functions ----------
func displayFinalScores(players []Player) {
	fmt.Println("\nFinal Scores:")
	for _, player := range players {
		fmt.Printf("%s: %d\n", player.Name, calculateTotalScore(player.ScoreCard))
	}
}

func displayCurrentScoreboard(players []Player) {
	fmt.Println("\nCurrent Scoreboard:")
	for _, player := range players {
		fmt.Printf("%s:\n", player.Name)
		for category, score := range player.ScoreCard.Scores {
			if score != nil {
				fmt.Printf("  %s: %d\n", category, *score)
			} else {
				fmt.Printf("  %s: -\n", category)
			}
		}
		fmt.Printf("  Total: %d\n\n", calculateTotalScore(player.ScoreCard))
	}
}
