package server

import (
	"log"
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

func (rc *RoomController) sendMessage(player *Player, message *messages.Message) {
	err := player.Connection.Encode(message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (rc *RoomController) sendMessageToRoom(room *Room, message *messages.Message) {
	for _, player := range room.Players {
		rc.sendMessage(player, message)
	}
}

func (rc *RoomController) CreateRoom(player *Player) {
	roomID := uuid.New().String()
	_, err := rc.roomManager.CreateRoom(roomID)
	if err != nil {
		log.Println("Error creating room:", err.Error())
		return
	}

	rc.roomManager.JoinRoom(roomID, player)

	message := messages.Message{
		Type:   messages.CreateRoom,
		Player: player.PlayerInfo(),
		RoomID: roomID,
	}
	rc.sendMessage(player, &message)
}

func (rc *RoomController) JoinRoom(roomID string, player *Player) {

	rc.addPlayerToRoom(roomID, player)

	// check if room is full, if so start game
	room, ok := rc.roomManager.GetRoom(roomID)
	if ok != nil {
		log.Println("Error getting room:", ok.Error())
		return
	}

	if len(room.Players) >= 2 {
		rc.StartGame(room)
	} else {
		log.Printf("Room %s has %d players, waiting for more players to join\n", roomID, len(room.Players))
	}
}

func (rc *RoomController) StartGame(room *Room) {
	//TODO client should send TurnStarted message to server when recieve GameStarted message
	log.Println("Starting game in room:", room.ID)
	room.StartGame(true)
	currentPlayer := room.Players[room.currentPlayerId]
	message := messages.Message{
		Type:   messages.GameStarted,
		Player: currentPlayer.PlayerInfo(),
		RoomID: room.ID,
	}
	rc.sendMessageToRoom(room, &message)
}

func (rc *RoomController) addPlayerToRoom(roomID string, player *Player) {
	room, err := rc.roomManager.JoinRoom(roomID, player)
	if err != nil {
		log.Println("Error joining room:", err.Error())
		return
	}

	rc.notifyPlayerJoinedRoomToOthers(room, player)
}

func (rc *RoomController) notifyPlayerJoinedRoomToOthers(room *Room, player *Player) {
	message := messages.Message{
		Type:   messages.JoinRoom,
		Player: player.PlayerInfo(),
		RoomID: room.ID,
	}
	rc.sendMessageToRoom(room, &message)
}

func (rc *RoomController) ListRooms(player *Player) {
	rooms := rc.roomManager.ListRooms()

	roomList := make([]string, 0, len(rooms))
	for _, room := range rooms {
		roomList = append(roomList, room.ID)
	}

	message := messages.Message{
		Type:     messages.ListRoomsResponse,
		Player:   player.PlayerInfo(),
		RoomList: roomList,
	}
	rc.sendMessage(player, &message)
}

func (rc *RoomController) LeaveRoom(roomID string, player *Player) {
	err := rc.roomManager.LeaveRoom(roomID, player)
	if err != nil {
		log.Println("Error leaving room:", err.Error())
		return
	}

	message := messages.Message{
		Type:   messages.LeaveRoom,
		Player: player.PlayerInfo(),
		RoomID: roomID,
	}
	room := rc.roomManager.rooms[roomID]

	rc.sendMessage(player, &message)
	rc.sendMessageToRoom(room, &message)

	if len(room.Players) == 0 {
		rc.roomManager.DestroyRoom(roomID)
	}
}

func (rc *RoomController) HandleMessage(message *messages.Message, player *Player) {
	switch message.Type {
	case messages.CreateRoom:
		rc.CreateRoom(player)
	case messages.JoinRoom:
		rc.JoinRoom(message.RoomID, player)
	case messages.ListRooms:
		rc.ListRooms(player)
	case messages.LeaveRoom:
		rc.LeaveRoom(message.RoomID, player)
	}
}
