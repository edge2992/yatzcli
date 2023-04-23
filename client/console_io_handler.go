package client

import "yatzcli/game"

type ConsoleIOHandler struct{}

func (c *ConsoleIOHandler) GetPlayerHoldInput(dice []game.Dice) []int {
	return game.GetPlayerHoldInput(dice)
}

func (c *ConsoleIOHandler) ChooseCategory(player *game.Player, dice []game.Dice) game.ScoreCategory {
	return game.ChooseCategory(player, dice)
}

func (c *ConsoleIOHandler) DisplayCurrentScoreboard(players []game.Player) {
	game.DisplayCurrentScoreboard(players)
}

func (c *ConsoleIOHandler) DisplayDice(dice []game.Dice) {
	game.DisplayDice(dice)
}
