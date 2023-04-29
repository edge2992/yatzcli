package server

import (
	"reflect"
	"testing"
	"yatzcli/game"
	"yatzcli/messages"
	"yatzcli/network"
)

func setupGameStartedEnvironment() (*Room, *GamePlayController) {
	rm := NewRoomManager()
	gpc := NewGamePlayController(rm)
	roomID := "mockRoom"
	_, player1 := createMockPlayer("player1")
	_, player2 := createMockPlayer("player2")
	rm.CreateRoom(roomID)
	rm.JoinRoom(roomID, player1)
	rm.JoinRoom(roomID, player2)
	rm.StartGame(roomID, false)

	room, _ := rm.GetRoom(roomID)
	return room, gpc
}

func TestGamePlayController_RollDice(t *testing.T) {
	room, gpc := setupGameStartedEnvironment()
	player := room.GetCurrentPlayer()
	gpc.RollDice(room.ID, player)

	if len(room.Players) != 2 {
		t.Error("Expected 2 players, got:", len(room.Players))
	}

	for _, player := range room.Players {
		mockConn := player.Connection.(*network.MockConnection)
		msg := mockConn.TopEncodedMessage().(*messages.Message)

		if msg.Type != messages.DiceRolled {
			t.Error("Expected diceRolled message, got:", msg.Type)
		}

		if len(msg.Dice) != game.NumberOfDice {
			t.Error("Expected", game.NumberOfDice, "dice, got:", len(msg.Dice))
		}
	}
}

func TestGamePlayController_RerollDice(t *testing.T) {
	room, gpc := setupGameStartedEnvironment()
	player := room.GetCurrentPlayer()

	// Roll the dice first
	gpc.RollDice(room.ID, player)

	// Select some dice to hold
	oldDices := make([]game.Dice, game.NumberOfDice)
	rerollDice := make([]game.Dice, game.NumberOfDice)
	copy(oldDices, (*room).dices)
	copy(rerollDice, (*room).dices)
	selectedIndices := []int{0, 1, 2}

	for _, i := range selectedIndices {
		rerollDice[i].Held = true
	}

	// Reroll the selected dice
	gpc.RerollDice(room.ID, player, rerollDice)

	for _, p := range room.Players {
		mockConn := p.Connection.(*network.MockConnection)
		msg := mockConn.TopEncodedMessage().(*messages.Message)

		if msg.Type != messages.DiceRolled {
			t.Error("Expected diceRolled message, got:", msg.Type)
		}

		if len(msg.Dice) != game.NumberOfDice {
			t.Errorf("Expected %d dice, got: %d", game.NumberOfDice, len(msg.Dice))
		}

		// Check that the held dice were not rerolled
		for _, i := range selectedIndices {
			if oldDices[i].Value != msg.Dice[i].Value {
				t.Errorf("Expected dice %d to be %d, got: %d", i, oldDices[i].Value, msg.Dice[i].Value)
			}
		}
	}
}

func TestGamePlayController_StartTurn(t *testing.T) {
	room, gpc := setupGameStartedEnvironment()
	player := room.GetCurrentPlayer()

	// Call StartTurn
	gpc.StartTurn(room.ID, player)

	// Check that the game turn number has been incremented
	if room.gameTurnNum != 1 {
		t.Errorf("Expected game turn number to be %d, got: %d", 1, room.gameTurnNum)
	}

	// Check that all dice have been reset
	for _, d := range (*room).dices {
		if d.Held {
			t.Error("Expected all dice to be reset, but some were held")
		}
	}

	// Check that the dice rolls count has been reset
	if room.diceRolls != 0 {
		t.Errorf("Expected dice rolls count to be %d, got: %d", 0, room.diceRolls)
	}

	// Check that the TurnStarted message has been sent to the player
	mockConn := player.Connection.(*network.MockConnection)
	msg := mockConn.TopEncodedMessage().(*messages.Message)

	if msg.Type != messages.TurnStarted {
		t.Error("Expected TurnStarted message, got:", msg.Type)
	}

	if msg.Player != player.PlayerInfo() {
		t.Error("Expected message player to be the current player")
	}
}

func TestGamePlayController_GameOver(t *testing.T) {
	// TODO
}

func TestGamePlayController_ChooseScoreCategory(t *testing.T) {
	room, gpc := setupGameStartedEnvironment()
	player := room.GetCurrentPlayer()
	player_id := room.currentPlayerId

	// Roll the dice first
	room.dices = []game.Dice{
		{Value: 3, Held: false},
		{Value: 3, Held: false},
		{Value: 5, Held: false},
		{Value: 5, Held: false},
		{Value: 5, Held: false},
	}
	category := game.FullHouse

	// Choose a score category
	gpc.ChooseScoreCategory(room.ID, player, category)

	// Check that the score has been calculated and recorded
	if player.ScoreCard.Scores[category] != 25 {
		t.Error("Expected score to be recorded for category:", category)
	}

	// Check that the category has been marked as filled
	if !player.ScoreCard.Filled[category] {
		t.Error("Expected category to be marked as filled:", category)
	}

	// Check that it's now the next player's turn
	nextPlayer := room.Players[(player_id+1)%len(room.Players)]
	if room.GetCurrentPlayer() != nextPlayer {
		t.Error("Expected next player to be:", nextPlayer.Name)
	}

	if room.GetCurrentPlayer() == player {
		t.Error("Expected next player to not be:", player.Name)
	}

	// Check that a TurnStarted message has been sent to the next player
	mockConn := nextPlayer.Connection.(*network.MockConnection)
	msg := mockConn.TopEncodedMessage().(*messages.Message)

	if msg.Type != messages.TurnStarted {
		t.Error("Expected TurnStarted message, got:", msg.Type)
	}

	if msg.Player != nextPlayer.PlayerInfo() {
		t.Error("Expected message player to be the next player")
	}

}

func TestGamePlayController_UpdateScoreCard(t *testing.T) {
	room, gpc := setupGameStartedEnvironment()

	// Roll the dice first
	player := room.GetCurrentPlayer()
	gpc.RollDice(room.ID, player)

	// Choose a score category
	category := game.Ones
	gpc.ChooseScoreCategory(room.ID, player, category)

	// Update the scorecard
	gpc.UpdateScoreCard(room.ID)

	// Check that an UpdateScorecard message has been sent to all players
	for _, p := range room.Players {
		mockConn := p.Connection.(*network.MockConnection)
		msg := mockConn.TopEncodedMessage().(*messages.Message)

		if msg.Type != messages.UpdateScorecard {
			t.Error("Expected UpdateScorecard message, got:", msg.Type)
		}

		if !reflect.DeepEqual(msg.Players, room.Players) {
			t.Error("Expected message players to be the same as room players")
		}
	}
}
