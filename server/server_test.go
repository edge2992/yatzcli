package server

import (
	"bytes"
	"encoding/gob"
	"testing"
	"yatzcli/game"
	"yatzcli/messages"
)

func setupTestEnvironment() (*Server, *gob.Encoder, *gob.Decoder) {
	server := NewServer()
	buf := &bytes.Buffer{}
	encoder := gob.NewEncoder(buf)
	decoder := gob.NewDecoder(buf)
	return server, encoder, decoder
}

func TestJoinGame(t *testing.T) {
	server, encoder, decoder := setupTestEnvironment()
	player := game.NewPlayer("Player 1")

	server.joinGame(player, encoder)

	message := &messages.Message{}
	err := decoder.Decode(message)
	if err != nil {
		t.Fatal("Error decoding message:", err.Error())
	}

	if message.Type != messages.GameJoined {
		t.Errorf("Expected message type to be GameJoined, got %v", message.Type)
	}

	if message.Player.Name != player.Name {
		t.Errorf("Expected player to be %v, got %v", player, message.Player)
	}

	if len(server.players) != 1 {
		t.Errorf("Expected server players to have 1 player, got %v", len(server.players))
	}
}

func TestPlayerReady(t *testing.T) {
	// Create a server and two players
	server, encoder, _ := setupTestEnvironment()

	player1 := game.NewPlayer("Player 1")
	player2 := game.NewPlayer("Player 2")

	// Add the players to the server
	server.players = append(server.players, player1, player2)
	server.encoders = append(server.encoders, encoder, encoder)

	// Call the playerReady function for both players
	server.playerReady(player1, encoder)
	server.playerReady(player2, encoder)

	// Check if the readyPlayers count is correct
	if server.readyPlayers != 2 {
		t.Errorf("Expected 2 ready players, got %d", server.readyPlayers)
	}

	// Check if the game has started
	if !server.gameStarted {
		t.Error("Expected the game to start, but it didn't")
	}

	// Check if the currentPlayer is set to 0 (first player)
	if server.currentPlayer != 0 {
		t.Errorf("Expected currentPlayer to be 0, got %d", server.currentPlayer)
	}
}

func TestRollDice(t *testing.T) {
	server, encoder, _ := setupTestEnvironment()

	player := game.NewPlayer("Player 1")

	server.players = append(server.players, player)
	server.encoders = append(server.encoders, encoder)

	server.gameStarted = true
	server.currentPlayer = 0

	server.rollDice(player, encoder)

	// Check if the dice have been rolled (their values should have changed)
	allZero := true
	for _, d := range server.dices {
		if d.Value != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Expected the dice to be rolled, but their values did not change")
	}

	// Check if the diceRolls count is correct
	if server.diceRolls != 1 {
		t.Errorf("Expected 1 dice roll, got %d", server.diceRolls)
	}
}

func TestRerollDice(t *testing.T) {
	server, encoder, _ := setupTestEnvironment()
	player := game.NewPlayer("Player 1")

	server.players = append(server.players, player)
	server.encoders = append(server.encoders, encoder)

	// Set the gameStarted flag and currentPlayer
	server.gameStarted = true
	server.currentPlayer = 0
	server.diceRolls = 0

	// Call the rollDice function to set initial dice values
	server.rollDice(player, encoder)

	// Store the initial dice values
	initialDiceValues := make([]int, len(server.dices))
	for i, d := range server.dices {
		initialDiceValues[i] = d.Value
	}

	// Create a dice slice with the same values as server.dices, and set some dice to be held
	dice := make([]game.Dice, len(server.dices))
	copy(dice, server.dices)
	dice[0].Held = true
	dice[2].Held = true

	// Call the rerollDice function
	server.rerollDice(player, dice, encoder)

	// Check if the held dice values remain unchanged, and other dice values have changed
	for i, d := range server.dices {
		if dice[i].Held && d.Value != initialDiceValues[i] {
			t.Errorf("Expected held dice %d to remain unchanged, but the value changed", i)
		}
		if !dice[i].Held && d.Value == initialDiceValues[i] {
			t.Errorf("Expected unheld dice %d to change, but the value remained the same", i)
		}
	}

	// Check if the diceRolls count is correct
	if server.diceRolls != 2 {
		t.Errorf("Expected 2 dice rolls, got %d", server.diceRolls)
	}
}

func TestChooseCategory(t *testing.T) {
	server, encoder, _ := setupTestEnvironment()
	player1 := game.NewPlayer("Player 1")
	player2 := game.NewPlayer("Player 2")

	server.players = append(server.players, player1, player2)
	server.encoders = append(server.encoders, encoder, encoder)

	// Set the gameStarted flag and currentPlayer
	server.gameStarted = true
	server.currentPlayer = 0

	// Set a specific dice configuration
	server.dices = []game.Dice{
		{Value: 3, Held: false},
		{Value: 3, Held: false},
		{Value: 5, Held: false},
		{Value: 5, Held: false},
		{Value: 5, Held: false},
	}

	// Choose a category
	category := game.FullHouse

	// Call the chooseCategory function
	server.chooseCategory(player1, category, encoder)

	// Check if the score for the chosen category is correct
	expectedScore := 25
	actualScore := player1.ScoreCard.Scores[category]
	if actualScore != expectedScore {
		t.Errorf("Expected score for Full House: %d, got: %d", expectedScore, actualScore)
	}

	// Check if the chosen category is marked as filled
	if !player1.ScoreCard.Filled[category] {
		t.Errorf("Expected category %s to be marked as filled, but it's not", category)
	}

	// Check if the currentPlayer has been updated
	if server.currentPlayer != 1 {
		t.Errorf("Expected currentPlayer to be updated to 1, got: %d", server.currentPlayer)
	}
}
