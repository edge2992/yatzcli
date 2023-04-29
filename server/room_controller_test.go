package server

import (
	"testing"
	"yatzcli/game"
	"yatzcli/messages"
	"yatzcli/network"
)

func createMockPlayer(name string) (*network.MockConnection, *game.Player) {
	mockConn := network.NewMockConnection()
	player := game.NewPlayer(name, mockConn)
	return mockConn, player
}

func TestRoomController_CreateRoom(t *testing.T) {
	rm := NewRoomManager()
	rc := NewRoomController(rm)

	mockConn, player := createMockPlayer("Player 1")

	rc.CreateRoom(player)

	if len(rm.rooms) != 1 {
		t.Error("Expected room to be created")
	}

	roomID := ""
	for id := range rm.rooms {
		roomID = id
		break
	}

	if roomID == "" {
		t.Error("Expected non-empty room ID")
	}

	room, _ := rm.GetRoom(roomID)
	if len(room.Players) != 1 {
		t.Error("Expected one player in room")
	}

	if room.Players[0] != player {
		t.Error("Expected player to be added to room")
	}

	if len(mockConn.EncodedMessages) != 1 {
		t.Error("Expected message to be sent to player")
	}

	msg := mockConn.EncodedMessages[0].(*messages.Message)
	if msg.Type != messages.CreateRoom {
		t.Error("Expected message to be CreateRoom")
	}

	if msg.Player != player {
		t.Error("Expected message to contain player")
	}

	if msg.RoomID != roomID {
		t.Error("Expected message to contain room ID")
	}

}

func TestRoomController_JoinRoom(t *testing.T) {
	rm := NewRoomManager()
	rc := NewRoomController(rm)

	_, player1 := createMockPlayer("Player 1")
	rc.CreateRoom(player1)

	_, player2 := createMockPlayer("Player 2")

	room := rm.ListRooms()[0]
	rc.JoinRoom(room.ID, player2)

	if len(room.Players) != 2 {
		t.Error("Expected two players in room, got", len(room.Players))
	}

	if room.Players[1] != player2 {
		t.Error("Expected player2 to be added to room")
	}
}

func TestRoomController_ListRooms(t *testing.T) {
	rm := NewRoomManager()
	rc := NewRoomController(rm)

	_, player1 := createMockPlayer("Player 1")
	rc.CreateRoom(player1)

	_, player2 := createMockPlayer("Player 2")
	rc.CreateRoom(player2)

	mockConn, player3 := createMockPlayer("Player 3")
	rc.ListRooms(player3)

	if len(mockConn.EncodedMessages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(mockConn.EncodedMessages))
	}

	message, ok := mockConn.EncodedMessages[0].(*messages.Message)
	if !ok {
		t.Fatalf("Expected message, got %T", mockConn.EncodedMessages[0])
	}

	if message.Type != messages.ListRoomsResponse {
		t.Errorf("Expected message type ListRoomsResponse, got %d", message.Type)
	}

	if len(message.RoomList) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(message.RoomList))
	}
}

func TestRoomController_LeaveRoom(t *testing.T) {
	rm := NewRoomManager()
	rc := NewRoomController(rm)

	mockConn, player1 := createMockPlayer("Player 1")
	rc.CreateRoom(player1)

	_, player2 := createMockPlayer("Player 2")

	room := rm.ListRooms()[0]
	rc.JoinRoom(room.ID, player2)

	if len(room.Players) != 2 {
		t.Errorf("Expected 2 players in room, got %d", len(room.Players))
	}

	rc.LeaveRoom(room.ID, player1)

	if len(room.Players) != 1 {
		t.Errorf("Expected 1 player to remain in the room, but found %d", len(room.Players))
	}

	if room.Players[0] == player1 {
		t.Error("Expected player1 to be removed from the room")
	}

	if room.Players[0] != player2 {
		t.Error("Expected player2 to still be in the room")
	}

	if len(mockConn.EncodedMessages) != 3 {
		t.Fatal("Expected 2 messages to be sent to player1, got", len(mockConn.EncodedMessages))
	}

	msg := mockConn.EncodedMessages[2].(*messages.Message)
	if msg.Type != messages.LeaveRoom {
		t.Error("Expected message to be LeaveRoom")
	}

	if msg.Player != player1 {
		t.Error("Expected message to contain player")
	}

	if msg.RoomID != room.ID {
		t.Error("Expected message to contain room ID")
	}
}
