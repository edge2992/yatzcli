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
	GameStarted
	GameOver
	DiceRolled
	TurnStarted
	UpdateScorecard
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
