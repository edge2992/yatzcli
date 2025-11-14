package server

import (
	"math/rand"
	"sync"
	"yatzcli/game"
)

type Room struct {
	mu sync.RWMutex // 並行アクセス制御用

	ID              string
	Players         []*Player
	dices           []game.Dice
	gameStarted     bool
	gameTurnNum     int
	currentPlayerId int
	diceRolls       int
}

func NewRoom(roomID string) *Room {
	return &Room{
		ID:      roomID,
		Players: []*Player{},
		dices:   game.CreateDices(),
	}
}

func (room *Room) AddPlayer(player *Player) error {
	room.mu.Lock()
	defer room.mu.Unlock()

	room.Players = append(room.Players, player)
	return nil
}

func (room *Room) RemovePlayer(player *Player) error {
	room.mu.Lock()
	defer room.mu.Unlock()

	for i, p := range room.Players {
		if p == player {
			room.Players = append(room.Players[:i], room.Players[i+1:]...)
			return nil
		}
	}
	return nil
}

func (room *Room) StartGame(started_randomly bool) {
	room.mu.Lock()
	defer room.mu.Unlock()

	room.gameStarted = true

	room.currentPlayerId = 0
	if started_randomly {
		room.currentPlayerId = rand.Intn(len(room.Players))
	}
}

func (room *Room) GetCurrentPlayer() *Player {
	room.mu.RLock()
	defer room.mu.RUnlock()

	return room.Players[room.currentPlayerId]
}
