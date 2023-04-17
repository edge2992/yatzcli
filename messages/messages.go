package messages

import "yatzcli/game"

type MessageType int

const (
	GameJoined MessageType = iota
	GameLeft
	GameStart
	GameStarted
	PlayerReady
	PlayerJoined
	PlayerLeft
	TurnPlayed
	RollDice
	UpdateGameState
	GameOver
)

type Message struct {
	Type          MessageType
	Players       []*game.Player
	Player        *game.Player
	CurrentPlayer string
	Dice          []game.Dice
	Category      game.ScoreCategory
}
