package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// ---------- Constants and Types ----------
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
		prompt := &survey.Input{
			Message: fmt.Sprintf("Enter name for player %d:", i+1),
		}
		survey.AskOne(prompt, &name)
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
		if dice[i].Held {
			color.Set(color.FgGreen)

		} else {
			color.Set(color.FgRed)
		}
		fmt.Printf("%d ", dice[i].Value)
		color.Unset()
	}
	fmt.Println()
}

func chooseCategory(player *Player) ScoreCategory {
	availableCategories := []string{}
	for cat, score := range player.ScoreCard.Scores {
		if score == nil {
			availableCategories = append(availableCategories, string(cat))
		}
	}

	selectedCategory := ""
	prompt := &survey.Select{
		Message: "Choose a category:",
		Options: availableCategories,
	}
	survey.AskOne(prompt, &selectedCategory)
	return ScoreCategory(selectedCategory)
}

func getPlayerHoldInput() string {
	var holdInput string
	fmt.Print("Enter the dice you want to hold (e.g. 123): ")
	fmt.Scanln(&holdInput)
	return holdInput
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

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Player"}

	for _, category := range []ScoreCategory{Ones, Twos, Threes, Fours, Fives, Sixes, ThreeOfAKind, FourOfAKind, FullHouse, SmallStraight, LargeStraight, Yahtzee, Chance} {
		header = append(header, string(category))
	}
	header = append(header, "Total")
	table.SetHeader(header)

	for _, player := range players {
		row := []string{player.Name}
		for _, category := range []ScoreCategory{Ones, Twos, Threes, Fours, Fives, Sixes, ThreeOfAKind, FourOfAKind, FullHouse, SmallStraight, LargeStraight, Yahtzee, Chance} {
			score := player.ScoreCard.Scores[category]
			if score != nil {
				row = append(row, fmt.Sprintf("%d", *score))
			} else {
				row = append(row, "-")
			}
		}
		row = append(row, fmt.Sprintf("%d", calculateTotalScore(player.ScoreCard)))
		table.Append(row)
	}
	table.Render()
}
