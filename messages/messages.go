package messages

import "yatzcli/game"

type MessageType int

const (
	ServerJoin MessageType = iota
	WaitForPlayers
	RoomFull
	RoomCreated
	RoomJoined
	RoomLeft
	GamePlayerJoined
	GamePlayerLeft
	GameStart
	GameStarted
	GameOver
	PlayerLeft
	DiceRolled
	TurnStarted
	TurnPlayed
	UpdateScorecard
	UpdateGameState
	RequestRollDice
	RequestRerollDice
	RequestChooseCategory
	RequestCreateRoom
	RequestJoinRoom
	RequestLeaveRoom
	RequestRoomList
	RoomListResponse
)

type Message struct {
	Type          MessageType
	Players       []*game.PlayerInfo
	Player        *game.PlayerInfo
	CurrentPlayer string
	Dice          []game.Dice
	DiceRolls     int
	Category      game.ScoreCategory
	RoomID        string
	RoomList      []string
}
