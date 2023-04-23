package game

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

type Player struct {
	Name      string
	ScoreCard ScoreCard
}

func NewPlayer(name string) *Player {
	player := Player{
		Name:      name,
		ScoreCard: NewScoreCard(),
	}
	return &player
}

func CreatePlayers() []Player {
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

func CreateGameState(players []Player) map[string]*Player {
	gameState := make(map[string]*Player)
	for i := range players {
		gameState[players[i].Name] = &players[i]
	}
	return gameState
}

func CreateDices() []Dice {
	dices := make([]Dice, NumberOfDice)
	return dices
}
