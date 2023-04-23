package client

import "yatzcli/game"

type IOHandler interface {
	GetPlayerHoldInput([]game.Dice) []int
	ChooseCategory(*game.Player, []game.Dice) game.ScoreCategory
	DisplayCurrentScoreboard([]game.Player)
	DisplayDice([]game.Dice)
}
