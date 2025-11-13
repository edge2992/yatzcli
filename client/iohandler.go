package client

import "yatzcli/game"

type ChoiceType int

const (
	JoinRoom ChoiceType = iota
	CreateRoom
)

type IOHandler interface {
	GetPlayerHoldInput([]game.Dice) []int
	ChooseCategory(*game.PlayerInfo, []game.Dice) game.ScoreCategory
	DisplayCurrentScoreboard([]game.PlayerInfo)
	DisplayDice([]game.Dice)
	DisplayGameOver([]game.PlayerInfo)
	askJoinOrCreateRoom() ChoiceType
	askRoomName() string
	askRoomSelection([]string) string
}
