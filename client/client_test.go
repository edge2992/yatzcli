package client

import (
	"reflect"
	"testing"
	"yatzcli/game"
	"yatzcli/messages"
	"yatzcli/network"
)

type mockIOHandler struct {
	displayedScoreboards [][]game.PlayerInfo
	displayedDice        [][]game.Dice
	getHoldInputCalls    [][]game.Dice
	chooseCategoryCalls  []struct {
		player *game.PlayerInfo
		dice   []game.Dice
	}
}

func (m *mockIOHandler) DisplayCurrentScoreboard(players []game.PlayerInfo) {
	m.displayedScoreboards = append(m.displayedScoreboards, players)
}

func (m *mockIOHandler) DisplayDice(dice []game.Dice) {
	m.displayedDice = append(m.displayedDice, dice)
}

func (m *mockIOHandler) GetPlayerHoldInput(dice []game.Dice) []int {
	m.getHoldInputCalls = append(m.getHoldInputCalls, dice)
	return []int{1, 3}
}

func (m *mockIOHandler) ChooseCategory(player *game.PlayerInfo, dice []game.Dice) game.ScoreCategory {
	m.chooseCategoryCalls = append(m.chooseCategoryCalls, struct {
		player *game.PlayerInfo
		dice   []game.Dice
	}{player, dice})
	return game.Ones
}

func (m *mockIOHandler) askJoinOrCreateRoom() ChoiceType {
	return JoinRoom
}

func (m *mockIOHandler) askRoomName() string {
	return ""
}

func (m *mockIOHandler) askRoomSelection([]string) string {
	return ""
}

func TestSetReady(t *testing.T) {
	conn := &network.MockConnection{}
	ioHandler := &mockIOHandler{}
	client := NewClient(conn, ioHandler)

	client.setReady()

	if len(conn.EncodedMessages) != 1 {
		t.Fatalf("Expected 1 encoded message, got %d", len(conn.EncodedMessages))
	}

	msg, ok := conn.EncodedMessages[0].(*messages.Message)
	if !ok {
		t.Fatal("Expected messages.Message type")
	}

	if msg.Type != messages.PlayerReady {
		t.Fatalf("Expected message type to be PlayerReady, got %d", msg.Type)
	}
}

func TestHandleUpdateScorecard(t *testing.T) {
	conn := &network.MockConnection{}
	ioHandler := &mockIOHandler{}
	client := NewClient(conn, ioHandler)

	player1 := &game.PlayerInfo{Name: "player1"}
	player2 := &game.PlayerInfo{Name: "player2"}

	msg := &messages.Message{
		Type:    messages.UpdateScorecard,
		Players: []*game.PlayerInfo{player1, player2},
	}

	client.handleUpdateScorecard(msg)

	if len(ioHandler.displayedScoreboards) != 1 {
		t.Fatalf("Expected 1 displayed scoreboard, got %d", len(ioHandler.displayedScoreboards))
	}

	scoreboard := ioHandler.displayedScoreboards[0]
	if len(scoreboard) != 2 {
		t.Fatalf("Expected scoreboard to have 2 players, got %d", len(scoreboard))
	}

	if scoreboard[0].Name != player1.Name || scoreboard[1].Name != player2.Name {
		t.Fatal("Displayed scoreboard has incorrect players")
	}
}

func TestHandleTurnStarted(t *testing.T) {
	conn := &network.MockConnection{}
	ioHandler := &mockIOHandler{}
	client := NewClient(conn, ioHandler)

	msg := &messages.Message{
		Type: messages.TurnStarted,
	}

	client.handleTurnStarted(msg)

	if !client.turnFlag {
		t.Fatal("Expected turnFlag to be set to true")
	}

	if len(conn.EncodedMessages) != 1 {
		t.Fatalf("Expected 1 encoded message, got %d", len(conn.EncodedMessages))
	}

	msg, ok := conn.EncodedMessages[0].(*messages.Message)
	if !ok {
		t.Fatal("Expected messages.Message type")
	}

	if msg.Type != messages.DiceRolled {
		t.Fatalf("Expected message type to be DiceRolled, got %d", msg.Type)
	}
}

func TestHandleDiceRolled(t *testing.T) {
	conn := &network.MockConnection{}
	ioHandler := &mockIOHandler{}
	client := NewClient(conn, ioHandler)

	dice := []game.Dice{
		{Value: 1, Held: false},
		{Value: 2, Held: false},
		{Value: 3, Held: false},
		{Value: 4, Held: false},
		{Value: 5, Held: false},
	}

	msg := &messages.Message{
		Type:      messages.DiceRolled,
		Dice:      dice,
		DiceRolls: 1,
	}

	client.turnFlag = true
	client.handleDiceRolled(msg)

	if len(ioHandler.displayedDice) != 1 {
		t.Fatalf("Expected 1 displayed dice, got %d", len(ioHandler.displayedDice))
	}

	displayedDice := ioHandler.displayedDice[0]
	if !reflect.DeepEqual(displayedDice, dice) {
		t.Fatal("Displayed dice do not match the input dice")
	}

	if len(conn.EncodedMessages) != 1 {
		t.Fatalf("Expected 1 encoded message, got %d", len(conn.EncodedMessages))
	}

	msg, ok := conn.EncodedMessages[0].(*messages.Message)
	if !ok {
		t.Fatal("Expected messages.Message type")
	}

	if msg.Type != messages.RerollDice {
		t.Fatalf("Expected message type to be RerollDice, got %d", msg.Type)
	}

	if len(ioHandler.getHoldInputCalls) != 1 {
		t.Fatalf("Expected 1 GetPlayerHoldInput call, got %d", len(ioHandler.getHoldInputCalls))
	}

	holdInputCall := ioHandler.getHoldInputCalls[0]
	if !reflect.DeepEqual(holdInputCall, dice) {
		t.Fatal("GetPlayerHoldInput call has incorrect input dice")
	}
}

func TestReRollDice(t *testing.T) {
	conn := &network.MockConnection{}
	ioHandler := &mockIOHandler{}
	client := NewClient(conn, ioHandler)

	dice := []game.Dice{
		{Value: 1, Held: false},
		{Value: 2, Held: false},
		{Value: 3, Held: false},
		{Value: 4, Held: false},
		{Value: 5, Held: false},
	}

	client.reRollDice(dice)

	if len(conn.EncodedMessages) != 1 {
		t.Fatalf("Expected 1 encoded message, got %d", len(conn.EncodedMessages))
	}

	msg, ok := conn.EncodedMessages[0].(*messages.Message)
	if !ok {
		t.Fatal("Expected messages.Message type")
	}

	if msg.Type != messages.RerollDice {
		t.Fatalf("Expected message type to be RerollDice, got %d", msg.Type)
	}

	if !reflect.DeepEqual(msg.Dice, dice) {
		t.Fatal("RerollDice message has incorrect dice")
	}
}

func TestChooseCategory(t *testing.T) {
	conn := &network.MockConnection{}
	ioHandler := &mockIOHandler{}
	client := NewClient(conn, ioHandler)

	player := &game.PlayerInfo{Name: "player1"}
	dice := []game.Dice{
		{Value: 1, Held: false},
		{Value: 2, Held: false},
		{Value: 3, Held: false},
		{Value: 4, Held: false},
		{Value: 5, Held: false},
	}

	client.chooseCategory(player, dice)

	if len(conn.EncodedMessages) != 1 {
		t.Fatalf("Expected 1 encoded message, got %d", len(conn.EncodedMessages))
	}

	msg, ok := conn.EncodedMessages[0].(*messages.Message)
	if !ok {
		t.Fatal("Expected messages.Message type")
	}

	if msg.Type != messages.ChooseCategory {
		t.Fatalf("Expected message type to be ChooseCategory, got %d", msg.Type)
	}

	if msg.Category != game.Ones {
		t.Fatalf("Expected message category to be Ones, got %s", msg.Category)
	}

	if !reflect.DeepEqual(msg.Player, player) {
		t.Fatal("ChooseCategory message has incorrect player")
	}

	if len(ioHandler.chooseCategoryCalls) != 1 {
		t.Fatalf("Expected 1 ChooseCategory call, got %d", len(ioHandler.chooseCategoryCalls))
	}

	chooseCategoryCall := ioHandler.chooseCategoryCalls[0]
	if !reflect.DeepEqual(chooseCategoryCall.player, player) {
		t.Fatal("ChooseCategory call has incorrect player")
	}

	if !reflect.DeepEqual(chooseCategoryCall.dice, dice) {
		t.Fatal("ChooseCategory call has incorrect dice")
	}

	if client.turnFlag {
		t.Fatal("Expected turnFlag to be set to false")
	}
}
