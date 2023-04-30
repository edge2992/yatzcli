package server

import (
	"fmt"
	"yatzcli/game"
	"yatzcli/network"

	"github.com/AlecAivazis/survey/v2"
)

type Player struct {
	Name       string
	ScoreCard  game.ScoreCard
	Connection network.Connection `gob:"-"`
}

func NewPlayer(name string, conn network.Connection) *Player {
	player := Player{
		Name:       name,
		ScoreCard:  game.NewScoreCard(),
		Connection: conn,
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
		players[i] = Player{Name: name, ScoreCard: game.NewScoreCard()}
	}
	return players
}

func (p *Player) PlayerInfo() *game.PlayerInfo {
	return &game.PlayerInfo{
		Name:      p.Name,
		ScoreCard: p.ScoreCard,
	}
}
