package server

import (
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

func (rc *RoomController) sendMessage(player *game.Player, message *messages.Message) {
	err := player.Connection.Encode(message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (rc *RoomController) CreateRoom(player *game.Player) {
	roomID := uuid.New().String()
	_, err := rc.roomManager.CreateRoom(roomID)
	if err != nil {
		log.Println("Error creating room:", err.Error())
		return
	}

	rc.roomManager.JoinRoom(roomID, player)

	message := messages.Message{
		Type:   messages.CreateRoom,
		Player: player,
		RoomID: roomID,
	}
	rc.sendMessage(player, &message)
}

func (rc *RoomController) JoinRoom(roomID string, player *game.Player) {

	rc.addPlayerToRoom(roomID, player)

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

func (rc *RoomController) addPlayerToRoom(roomID string, player *game.Player) {
	room, err := rc.roomManager.JoinRoom(roomID, player)
	if err != nil {
		log.Println("Error joining room:", err.Error())
		return
	}

	message := messages.Message{
		Type:   messages.JoinRoom,
		Player: player,
		RoomID: roomID,
	}
	rc.sendMessage(player, &message)
	rc.notifyPlayerJoinedRoomToOthers(room, player)
}

func (rc *RoomController) notifyPlayerJoinedRoomToOthers(room *Room, player *game.Player) {
	message := messages.Message{
		Type:   messages.PlayerJoinedRoom,
		Player: player,
		RoomID: room.ID,
	}
	for _, p := range room.Players {
		if p != player {
			rc.sendMessage(p, &message)
		}
	}
}

func (rc *RoomController) ListRooms(player *game.Player) {
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
	rc.sendMessage(player, &message)
}

func (rc *RoomController) LeaveRoom(room *Room, player *game.Player) {
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

func (rc *RoomController) HandleMessage(message *messages.Message, player *game.Player) {
	switch message.Type {
	case messages.CreateRoom:
		rc.CreateRoom(player)
	case messages.JoinRoom:
		rc.addPlayerToRoom(message.RoomID, player)
	case messages.ListRooms:
		rc.ListRooms(player)
		// case messages.LeaveRoom:
		// 	room := rc.rooms[message.RoomID]
		// 	rc.LeaveRoom(room, player, encoder)
	}

}
