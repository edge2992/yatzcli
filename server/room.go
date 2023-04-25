package server

import (
	"encoding/gob"
	"yatzcli/game"
)

type Room struct {
	ID              string
	Players         []*game.Player
	encoders        []*gob.Encoder
	dices           []game.Dice
	gameStarted     bool
	gameTurnNum     int
	currentPlayerId int
	diceRolls       int
}

func NewRoom(roomID string) *Room {
	return &Room{
		ID:       roomID,
		Players:  []*game.Player{},
		encoders: []*gob.Encoder{},
		dices:    game.CreateDices(),
	}
}

func (room *Room) AddPlayer(player *game.Player, encoder *gob.Encoder) error {
	room.Players = append(room.Players, player)
	room.encoders = append(room.encoders, encoder)
	return nil
}

// Move all the methods related to Room type here from the controller.go file
// For example:
// func (room *Room) startGame() {
//     // ...
// }
