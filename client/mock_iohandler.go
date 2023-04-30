package client

import "yatzcli/game"

type MockIOHandler struct {
	displayedScoreboards [][]game.PlayerInfo
	displayedDice        [][]game.Dice
	getHoldInputCalls    [][]game.Dice
	chooseCategoryCalls  []struct {
		player *game.PlayerInfo
		dice   []game.Dice
	}
	createOrJoin ChoiceType
}

func (m *MockIOHandler) DisplayCurrentScoreboard(players []game.PlayerInfo) {
	m.displayedScoreboards = append(m.displayedScoreboards, players)
}

func (m *MockIOHandler) DisplayDice(dice []game.Dice) {
	m.displayedDice = append(m.displayedDice, dice)
}

func (m *MockIOHandler) GetPlayerHoldInput(dice []game.Dice) []int {
	m.getHoldInputCalls = append(m.getHoldInputCalls, dice)
	return []int{1, 3}
}

func (m *MockIOHandler) ChooseCategory(player *game.PlayerInfo, dice []game.Dice) game.ScoreCategory {
	m.chooseCategoryCalls = append(m.chooseCategoryCalls, struct {
		player *game.PlayerInfo
		dice   []game.Dice
	}{player, dice})
	return game.Ones
}

func (m *MockIOHandler) askJoinOrCreateRoom() ChoiceType {
	return m.createOrJoin
}

func (m *MockIOHandler) askRoomName() string {
	return ""
}

func (m *MockIOHandler) askRoomSelection([]string) string {
	return ""
}
