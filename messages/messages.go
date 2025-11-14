package messages

import "yatzcli/game"

type MessageType int

const (
	// Game lifecycle events
	GameStarted MessageType = iota
	GameOver

	// Player events
	PlayerLeft

	// Turn events
	TurnStarted
	DiceRolled

	// Actions
	RerollDice
	ChooseCategory
	UpdateScorecard

	// Room management
	CreateRoom
	JoinRoom
	LeaveRoom
	ListRooms
	ListRoomsResponse

	// System
	ServerJoin
	Error

	// Removed unused types (2025-11-14):
	// - GameJoined, GameLeft, GameStart: Unused game lifecycle events
	// - PlayerReady, PlayerJoined, PlayerJoinedRoom: Unused player events
	// - TurnPlayed: Unused turn event
	// - RollDice: Duplicate of RerollDice
	// - UpdateGameState: Unused game state update
	// - WaitForPlayers, RoomFull: Unused room states
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
	ErrorMessage  string
}
