package server

import (
	"testing"
	"yatzcli/game"
	"yatzcli/messages"
	"yatzcli/network"
)

func TestRoomController_CreateRoom(t *testing.T) {
	rm := NewRoomManager()
	rc := NewRoomController(rm)

	mockConn := network.NewMockConnection()
	player := game.NewPlayer("Player 1", mockConn)

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
