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
	PlayerJoinedRoom
	PlayerLeft
	DiceRolled
	RerollDice
	TurnStarted
	TurnPlayed
	ChooseCategory
	UpdateScorecard
	RollDice
	UpdateGameState
	GameOver
	CreateRoom
	JoinRoom
	LeaveRoom
	ListRooms
	ListRoomsResponse
	WaitForPlayers
	RoomFull
)

type Message struct {
	Type          MessageType
	Players       []*game.Player
	Player        *game.Player
	CurrentPlayer string
	Dice          []game.Dice
	DiceRolls     int
	Category      game.ScoreCategory
	RoomID        string
	RoomList      []string
}
