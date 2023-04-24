package server

import (
	"bytes"
	"encoding/gob"
	"testing"
	"yatzcli/game"
	"yatzcli/messages"
)

func setupGameControllerTestEnvironment() (*GameController, *gob.Encoder, *gob.Decoder) {
	controller := NewGameController()
	buf := &bytes.Buffer{}
	encoder := gob.NewEncoder(buf)
	decoder := gob.NewDecoder(buf)
	return controller, encoder, decoder
}
func TestCreateRoom(t *testing.T) {
	controller, encoder, decoder := setupGameControllerTestEnvironment()
	player := game.NewPlayer("Player 1")

	controller.CreateRoom(player, encoder)

	message := &messages.Message{}
	err := decoder.Decode(message)
	if err != nil {
		t.Fatal("Error decoding message:", err.Error())
	}

	if message.Type != messages.CreateRoom {
		t.Errorf("Expected message type to be CreateRoom, got %v", message.Type)
	}

	if message.Player.Name != player.Name {
		t.Errorf("Expected player to be %v, got %v", player, message.Player)
	}

	if len(controller.rooms) != 1 {
		t.Errorf("Expected controller rooms to have 1 room, got %v", len(controller.rooms))
	}
}

func TestJoinRoom(t *testing.T) {
	controller, encoder, decoder := setupGameControllerTestEnvironment()
	player1 := game.NewPlayer("Player 1")
	player2 := game.NewPlayer("Player 2")
	encoder2 := gob.NewEncoder(&bytes.Buffer{})

	controller.CreateRoom(player1, encoder)

	message := &messages.Message{}
	err := decoder.Decode(message)
	if err != nil {
		t.Fatal("Error decoding message:", err.Error())
	}

	roomID := message.RoomID

	controller.JoinRoom(roomID, player2, encoder2)

	room, ok := controller.rooms[roomID]
	if !ok {
		t.Fatalf("Room not found: %v", roomID)
	}

	if len(room.Players) != 2 {
		t.Errorf("Expected room players to have 2 players, got %v", len(room.Players))
	}
}

// func TestStartGame(t *testing.T) {
// 	controller, encoder, decoder := setupGameControllerTestEnvironment()
// 	player1 := game.NewPlayer("Player 1")
// 	player2 := game.NewPlayer("Player 2")

// 	controller.CreateRoom(player1, encoder)
// 	roomID := ""
// 	for id := range controller.rooms {
// 		roomID = id
// 		break
// 	}

// 	controller.JoinRoom(roomID, player2, encoder)
// 	controller.StartGame(roomID)

// 	message := &messages.Message{}
// 	err := decoder.Decode(message)
// 	if err != nil {
// 		t.Fatal("Error decoding message:", err.Error())
// 	}

// 	if message.Type != messages.GameStarted {
// 		t.Errorf("Expected message type to be GameStarted, got %v", message.Type)
// 	}

// 	room, ok := controller.rooms[roomID]
// 	if !ok {
// 		t.Fatalf("Room not found: %v", roomID)
// 	}

// 	if !room.gameStarted {
// 		t.Error("Expected game to be started")
// 	}
// }

// func TestJoinGameController(t *testing.T) {
// 	controller, encoder, decoder := setupGameControllerTestEnvironment()
// 	player := game.NewPlayer("Player 1")

// 	controller.JoinGame(player, encoder)

// 	message := &messages.Message{}
// 	err := decoder.Decode(message)
// 	if err != nil {
// 		t.Fatal("Error decoding message:", err.Error())
// 	}

// 	if message.Type != messages.GameJoined {
// 		t.Errorf("Expected message type to be GameJoined, got %v", message.Type)
// 	}

// 	if message.Player.Name != player.Name {
// 		t.Errorf("Expected player to be %v, got %v", player, message.Player)
// 	}

// 	if len(controller.players) != 1 {
// 		t.Errorf("Expected controller players to have 1 player, got %v", len(controller.players))
// 	}
// }

// // func TestLeaveGameController(t *testing.T) {
// // 	controller, encoder, decoder := setupGameControllerTestEnvironment()
// // 	player := game.NewPlayer("Player 1")

// // 	controller.JoinGame(player, encoder)
// // 	controller.LeaveGame(player, encoder)

// // 	message := &messages.Message{}
// // 	err := decoder.Decode(message)
// // 	print(message)
// // 	if err != nil {
// // 		t.Fatal("Error decoding message:", err.Error())
// // 	}

// // 	if message.Type != messages.GameLeft {
// // 		t.Errorf("Expected message type to be GameLeft, got %v", message.Type)
// // 	}

// // 	if message.Player.Name != player.Name {
// // 		t.Errorf("Expected player to be %v, got %v", player, message.Player)
// // 	}

// // 	if len(controller.players) != 0 {
// // 		t.Errorf("Expected controller players to have 0 players, got %v", len(controller.players))
// // 	}
// // }

// func TestPlayerReadyController(t *testing.T) {
// 	controller, _, _ := setupGameControllerTestEnvironment()
// 	player1 := game.NewPlayer("Player 1")
// 	player2 := game.NewPlayer("Player 2")
// 	encoder1 := gob.NewEncoder(&bytes.Buffer{})
// 	encoder2 := gob.NewEncoder(&bytes.Buffer{})

// 	controller.JoinGame(player1, encoder1)
// 	controller.JoinGame(player2, encoder2)

// 	controller.PlayerReady(player1, encoder1)
// 	if controller.gameStarted {
// 		t.Error("Expected game not to start with only one player ready")
// 	}

// 	controller.PlayerReady(player2, encoder2)
// 	if !controller.gameStarted {
// 		t.Error("Expected game to start when two players are ready")
// 	}
// }

// func TestStartTurnController(t *testing.T) {
// 	controller, _, _ := setupGameControllerTestEnvironment()
// 	player1 := game.NewPlayer("Player 1")
// 	player2 := game.NewPlayer("Player 2")
// 	encoder1 := gob.NewEncoder(&bytes.Buffer{})
// 	encoder2 := gob.NewEncoder(&bytes.Buffer{})

// 	controller.JoinGame(player1, encoder1)
// 	controller.JoinGame(player2, encoder2)
// 	controller.PlayerReady(player1, encoder1)
// 	controller.PlayerReady(player2, encoder2)

// 	// TODO: Fix this test
// 	// when the number of players is 2, the game starts.
// 	// controller.StartTurn(player1, encoder1)

// 	if controller.gameTurnNum != 1 {
// 		t.Errorf("Expected gameTurnNum to be 1, got %v", controller.gameTurnNum)
// 	}
// }

// func TestRollDiceController(t *testing.T) {
// 	controller, _, _ := setupGameControllerTestEnvironment()
// 	player1 := game.NewPlayer("Player 1")
// 	encoder1 := gob.NewEncoder(&bytes.Buffer{})

// 	controller.JoinGame(player1, encoder1)

// 	controller.RollDice(player1, encoder1)
// 	if controller.diceRolls != 1 {
// 		t.Errorf("Expected diceRolls to be 1, got %v", controller.diceRolls)
// 	}
// }

// func TestRerollDiceController(t *testing.T) {
// 	controller, _, _ := setupGameControllerTestEnvironment()
// 	player1 := game.NewPlayer("Player 1")
// 	encoder1 := gob.NewEncoder(&bytes.Buffer{})

// 	controller.JoinGame(player1, encoder1)
// 	controller.RollDice(player1, encoder1)

// 	oldDices := make([]game.Dice, len(controller.dices))
// 	copy(oldDices, controller.dices)

// 	rerollDice := make([]game.Dice, len(controller.dices))
// 	copy(rerollDice, controller.dices)
// 	rerollDice[0].Held = true
// 	rerollDice[1].Held = true

// 	controller.RerollDice(player1, rerollDice, encoder1)
// 	if controller.dices[0].Value != oldDices[0].Value || controller.dices[1].Value != oldDices[1].Value {
// 		t.Error("Expected dice to be rerolled")
// 	}
// }

// func TestChooseScoreCategoryController(t *testing.T) {
// 	controller, _, _ := setupGameControllerTestEnvironment()
// 	player1 := game.NewPlayer("Player 1")
// 	player2 := game.NewPlayer("Player 2")
// 	encoder1 := gob.NewEncoder(&bytes.Buffer{})
// 	encoder2 := gob.NewEncoder(&bytes.Buffer{})

// 	controller.JoinGame(player1, encoder1)
// 	controller.JoinGame(player2, encoder2)
// 	controller.PlayerReady(player1, encoder1)
// 	controller.PlayerReady(player2, encoder2)

// 	controller.RollDice(player1, encoder1)
// 	controller.ChooseScoreCategory(player1, game.Ones, encoder1)

// 	if !player1.ScoreCard.Filled[game.Ones] {
// 		t.Error("Expected Ones category to be filled")
// 	}
// }

// // func TestUpdateScoreCardController(t *testing.T) {
// // 	controller, _, buf := setupGameControllerTestEnvironment()
// // 	player1 := game.NewPlayer("Player 1")
// // 	player2 := game.NewPlayer("Player 2")
// // 	encoder1 := gob.NewEncoder(&bytes.Buffer{})
// // 	encoder2 := gob.NewEncoder(&bytes.Buffer{})

// // 	controller.JoinGame(player1, encoder1)
// // 	controller.JoinGame(player2, encoder2)
// // 	controller.PlayerReady(player1, encoder1)
// // 	controller.PlayerReady(player2, encoder2)

// // 	controller.RollDice(player1, encoder1)
// // 	controller.ChooseScoreCategory(player1, game.Ones, encoder1)

// // 	controller.UpdateScoreCard()

// // 	message := &messages.Message{}
// // 	err := gob.NewDecoder(buf).Decode(message)
// // 	if err != nil {
// // 		t.Fatalf("Error decoding message: %v", err)
// // 	}

// // 	if message.Type != messages.UpdateScorecard {
// // 		t.Errorf("Expected message type to be UpdateScorecard, got %v", message.Type)
// // 	}
// // }

// func TestChooseScoreCategoryController_FullHose(t *testing.T) {
// 	controller, _, _ := setupGameControllerTestEnvironment()
// 	player1 := game.NewPlayer("Player 1")
// 	player2 := game.NewPlayer("Player 2")
// 	encoder1 := gob.NewEncoder(&bytes.Buffer{})
// 	encoder2 := gob.NewEncoder(&bytes.Buffer{})

// 	controller.JoinGame(player1, encoder1)
// 	controller.JoinGame(player2, encoder2)
// 	controller.PlayerReady(player1, encoder1)
// 	controller.PlayerReady(player2, encoder2)

// 	controller.dices = []game.Dice{
// 		{Value: 3, Held: false},
// 		{Value: 3, Held: false},
// 		{Value: 5, Held: false},
// 		{Value: 5, Held: false},
// 		{Value: 5, Held: false},
// 	}
// 	category := game.FullHouse

// 	controller.ChooseScoreCategory(player1, category, encoder1)

// 	expectedScore := 25
// 	actualScore := player1.ScoreCard.Scores[category]

// 	if actualScore != expectedScore {
// 		t.Errorf("Expected score to be %v, got %v", expectedScore, actualScore)
// 	}

// 	if !player1.ScoreCard.Filled[category] {
// 		t.Errorf("Expected category %v to be filled", category)
// 	}

// 	if controller.currentPlayer != 1 {
// 		t.Errorf("Expected current player to be 1, got %v", controller.currentPlayer)
// 	}
// }

// func TestGameOverController(t *testing.T) {
// 	controller, _, _ := setupGameControllerTestEnvironment()
// 	player1 := game.NewPlayer("Player 1")
// 	player2 := game.NewPlayer("Player 2")
// 	encoder1 := gob.NewEncoder(&bytes.Buffer{})
// 	encoder2 := gob.NewEncoder(&bytes.Buffer{})

// 	controller.JoinGame(player1, encoder1)
// 	controller.JoinGame(player2, encoder2)
// 	controller.PlayerReady(player1, encoder1)
// 	controller.PlayerReady(player2, encoder2)

// 	controller.gameTurnNum = game.NumberOfRounds * len(controller.players)
// 	controller.GameOver()

// 	if controller.gameStarted {
// 		t.Error("Expected game to be over")
// 	}
// }
