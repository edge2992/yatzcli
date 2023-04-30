package client

import (
	"fmt"
	"os"
	"strings"
	"yatzcli/game"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type ConsoleIOHandler struct{}

func (c *ConsoleIOHandler) GetPlayerHoldInput(dice []game.Dice) []int {
	var selectedIndices []int

	diceOptions := make([]string, len(dice))
	diceChecked := []int{}
	for i, die := range dice {
		diceOptions[i] = fmt.Sprintf("%d", die.Value)
		if die.Held {
			diceChecked = append(diceChecked, i)
		}
	}
	prompt := &survey.MultiSelect{
		Message: "select the dice you want to hold (use space to select and tab to navigate):",
		Options: diceOptions,
		Default: diceChecked,
	}

	err := survey.AskOne(prompt, &selectedIndices)
	if err != nil {
		fmt.Print(err)
		return nil
	}
	return selectedIndices
}

func (c *ConsoleIOHandler) ChooseCategory(player *game.PlayerInfo, dice []game.Dice) game.ScoreCategory {
	availableCategories := []string{}
	for cat, filled := range player.ScoreCard.Filled {
		if !filled {
			availableCategories = append(availableCategories, string(cat))
		}
	}

	selectedCategory := ""
	prompt := &survey.Select{
		Message: "Choose a category:",
		Options: game.CategoryWithScore(dice, availableCategories),
	}
	survey.AskOne(prompt, &selectedCategory)
	return game.ScoreCategory(strings.Split(selectedCategory, "\t")[0])
}

func (c *ConsoleIOHandler) DisplayCurrentScoreboard(players []game.PlayerInfo) {
	fmt.Println("\nCurrent Scoreboard:")

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Player"}

	for _, category := range game.AllCategories {
		header = append(header, string(category))
	}
	header = append(header, "Total")
	table.SetHeader(header)

	for _, player := range players {
		row := []string{player.Name}
		for _, category := range game.AllCategories {
			score := player.ScoreCard.Scores[category]
			filled := player.ScoreCard.Filled[category]
			if filled {
				row = append(row, fmt.Sprintf("%d", score))
			} else {
				row = append(row, "-")
			}
		}
		row = append(row, fmt.Sprintf("%d", game.CalculateTotalScore(player.ScoreCard)))
		table.Append(row)
	}
	table.Render()
}

func (c *ConsoleIOHandler) DisplayDice(dice []game.Dice) {
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

func (c *ConsoleIOHandler) askJoinOrCreateRoom() ChoiceType {
	var joinOrCreate string
	prompt := &survey.Select{
		Message: "Do you want to join or create a room?",
		Options: []string{"Join", "Create"},
	}
	survey.AskOne(prompt, &joinOrCreate)

	if joinOrCreate == "Join" {
		return JoinRoom
	} else {
		return CreateRoom
	}
}

func (c *ConsoleIOHandler) askRoomName() string {
	var roomName string
	prompt := &survey.Input{
		Message: "Enter a name for the new room:",
	}
	survey.AskOne(prompt, &roomName)
	return roomName
}

func (c *ConsoleIOHandler) askRoomSelection(rooms []string) string {
	var selectedRoom string
	prompt := &survey.Select{
		Message: "Select a room to join:",
		Options: rooms,
	}
	survey.AskOne(prompt, &selectedRoom)
	return selectedRoom
}
