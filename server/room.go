package server

import (
	"math/rand"
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

func (room *Room) RemovePlayer(player *game.Player) error {
	for i, p := range room.Players {
		if p == player {
			room.Players = append(room.Players[:i], room.Players[i+1:]...)
			return nil
		}
	}
	return nil
}

func (room *Room) StartGame(started_randomly bool) {
	room.gameStarted = true

	room.currentPlayerId = 0
	if started_randomly {
		room.currentPlayerId = rand.Intn(len(room.Players))
	}
}

func (room *Room) GetCurrentPlayer() *game.Player {
	return room.Players[room.currentPlayerId]
}
