package server

import (
	"yatzcli/game"
)

type Room struct {
	ID              string
	Players         []*game.Player
	dices           []game.Dice
	gameStarted     bool
	gameTurnNum     int
	currentPlayerId int
	diceRolls       int
}

func NewRoom(roomID string) *Room {
	return &Room{
		ID:      roomID,
		Players: []*game.Player{},
		dices:   game.CreateDices(),
	}
}

func (room *Room) AddPlayer(player *game.Player) error {
	room.Players = append(room.Players, player)
	return nil
}

// Move all the methods related to Room type here from the controller.go file
// For example:
// func (room *Room) startGame() {
//     // ...
// }
