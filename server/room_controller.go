package server

import (
	"encoding/gob"
	"log"
	"yatzcli/game"
	"yatzcli/messages"

	"github.com/google/uuid"
)

type RoomController struct {
	roomManager *RoomManager
}

func NewRoomController(rm *RoomManager) *RoomController {
	return &RoomController{
		roomManager: rm,
	}
}

func (rc *RoomController) CreateRoom(player *game.Player, encoder *gob.Encoder) {
	roomID := uuid.New().String()
	_, err := rc.roomManager.CreateRoom(roomID)
	if err != nil {
		log.Println("Error creating room:", err.Error())
		return
	}

	rc.roomManager.JoinRoom(roomID, player, encoder)

	message := messages.Message{
		Type:   messages.CreateRoom,
		Player: player,
		RoomID: roomID,
	}
	encodeError := encoder.Encode(&message)
	if encodeError != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (rc *RoomController) JoinRoom(roomID string, player *game.Player, encoder *gob.Encoder) {

	rc.addPlayerToRoom(roomID, player, encoder)

	//TODO check if room is full, if so start game

	// room, ok := rc.rooms[roomID]
	// if !ok {
	// 	log.Println("Room not found:", roomID)
	// 	return
	// }

	// if len(room.Players) >= 2 {
	// 	rc.StartGame(roomID)
	// }
}

func (rc *RoomController) addPlayerToRoom(roomID string, player *game.Player, encoder *gob.Encoder) {
	room, err := rc.roomManager.JoinRoom(roomID, player, encoder)
	if err != nil {
		log.Println("Error joining room:", err.Error())
		return
	}

	message := messages.Message{
		Type:   messages.JoinRoom,
		Player: player,
		RoomID: roomID,
	}
	encodeErr := encoder.Encode(&message)
	if encodeErr != nil {
		log.Println("Error encoding message:", err.Error())
	}
	rc.notifyPlayerJoinedRoomToOthers(room, player, encoder)
}

func (rc *RoomController) notifyPlayerJoinedRoomToOthers(room *Room, player *game.Player, encoder *gob.Encoder) {
	message := messages.Message{
		Type:   messages.PlayerJoinedRoom,
		Player: player,
		RoomID: room.ID,
	}
	for i, p := range room.Players {
		if p != player {
			err := room.encoders[i].Encode(&message)
			if err != nil {
				log.Println("Error encoding message:", err.Error())
			}
		}
	}
}

func (rc *RoomController) ListRooms(player *game.Player, encoder *gob.Encoder) {
	rooms := rc.roomManager.ListRooms()

	roomList := make([]string, 0, len(rooms))
	for _, room := range rooms {
		roomList = append(roomList, room.ID)
	}

	message := messages.Message{
		Type:     messages.ListRoomsResponse,
		Player:   player,
		RoomList: roomList,
	}
	err := encoder.Encode(&message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (rc *RoomController) LeaveRoom(room *Room, player *game.Player, encoder *gob.Encoder) {
	// for i, p := range room.Players {
	// 	if p.Name == player.Name {
	// 		room.Players = append(room.Players[:i], room.Players[i+1:]...)
	// 		break
	// 	}
	// }

	// message := messages.Message{
	// 	Type:   messages.GameLeft,
	// 	Player: player,
	// }
	// err := encoder.Encode(&message)
	// if err != nil {
	// 	log.Println("Error encoding message:", err.Error())
	// }
}

func (rc *RoomController) HandleMessage(message *messages.Message, player *game.Player, encoder *gob.Encoder) {
	switch message.Type {
	case messages.CreateRoom:
		rc.CreateRoom(player, encoder)
	case messages.JoinRoom:
		rc.addPlayerToRoom(message.RoomID, player, encoder)
	case messages.ListRooms:
		rc.ListRooms(player, encoder)
		// case messages.LeaveRoom:
		// 	room := rc.rooms[message.RoomID]
		// 	rc.LeaveRoom(room, player, encoder)
	}

}
