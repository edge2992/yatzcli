package server

import (
	"encoding/gob"
	"fmt"
	"log"
	"sync"
	"yatzcli/game"
	"yatzcli/messages"

	"github.com/google/uuid"
)

type Room struct {
	ID              string
	Players         []*game.Player
	encoders        []*gob.Encoder
	dices           []game.Dice
	gameStarted     bool
	gameTurnNum     int
	currentPlayerId int
	diceRolls       int
}

type GameController struct {
	rooms     map[string]*Room
	mutex     sync.Mutex
	playerNum int
}

func NewGameController() *GameController {
	return &GameController{
		rooms:     make(map[string]*Room),
		playerNum: 0,
	}
}

func (gc *GameController) BroadcastMessageToRoom(roomID string, message *messages.Message) {
	room, ok := gc.rooms[roomID]
	if !ok {
		log.Println("Room not found:", roomID)
		return
	}

	for _, encoder := range room.encoders {
		err := encoder.Encode(message)
		if err != nil {
			fmt.Println("Error encoding message:", err.Error())
		}
	}
}

func (gc *GameController) StartGame(roomID string) {
	room, ok := gc.rooms[roomID]
	if !ok {
		log.Println("Room not found:", roomID)
		return
	}

	if room.gameStarted {
		log.Println("Game already started")
		return
	}
	room.gameStarted = true
	room.currentPlayerId = 0

	message := messages.Message{
		Type: messages.GameStarted,
	}
	gc.BroadcastMessageToRoom(roomID, &message)

	gc.StartTurn(room, room.Players[room.currentPlayerId], room.encoders[room.currentPlayerId])
}

func (gc *GameController) LeaveGame(room *Room, player *game.Player, encoder *gob.Encoder) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	for i, p := range room.Players {
		if p.Name == player.Name {
			room.Players = append(room.Players[:i], room.Players[i+1:]...)
			break
		}
	}

	message := messages.Message{
		Type:   messages.GameLeft,
		Player: player,
	}
	err := encoder.Encode(&message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (gc *GameController) StartTurn(room *Room, player *game.Player, encoder *gob.Encoder) {
	gc.UpdateScoreCard(room)
	if room.gameTurnNum == game.NumberOfRounds*len(room.Players) {
		gc.GameOver(room)
		return
	}
	if room.Players[room.currentPlayerId] != player {
		return
	}
	room.gameTurnNum += 1

	for i := 0; i < game.NumberOfDice; i++ {
		room.dices[i].Held = false
	}
	room.diceRolls = 0

	message := messages.Message{
		Type:   messages.TurnStarted,
		Player: player,
	}
	err := encoder.Encode(&message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (gc *GameController) RollDice(room *Room, player *game.Player, encoder *gob.Encoder) {
	room.diceRolls += 1
	game.RollDice(room.dices)

	message := messages.Message{
		Type:      messages.DiceRolled,
		Player:    player,
		Dice:      room.dices,
		DiceRolls: room.diceRolls,
	}
	gc.BroadcastMessageToRoom(room.ID, &message)
}

func (gc *GameController) RerollDice(room *Room, player *game.Player, dice []game.Dice, encoder *gob.Encoder) {
	if room.diceRolls >= game.NumberOfDice {
		// TODO: Send error message
		log.Println("Cannot reroll dice more than", game.NumberOfDice, "times")
		return
	}
	// rough implementation of rerolling dice
	// Don't trust the dice numbers returned from the client
	// trust server's dice numbers
	selectedIndices := make([]int, 0)
	for i, d := range dice {
		if d.Held {
			selectedIndices = append(selectedIndices, i)
		}
	}

	game.HoldDice(room.dices, selectedIndices)
	gc.RollDice(room, player, encoder)
}

func (gc *GameController) ChooseScoreCategory(room *Room, player *game.Player, category game.ScoreCategory, encoder *gob.Encoder) {
	if room.Players[room.currentPlayerId] != player {
		return
	}

	score := game.CalculateScore(room.dices, category)
	player.ScoreCard.Scores[category] = score
	player.ScoreCard.Filled[category] = true

	room.currentPlayerId = (room.currentPlayerId + 1) % len(room.Players)
	gc.StartTurn(room, room.Players[room.currentPlayerId], room.encoders[room.currentPlayerId])
}

func (gc *GameController) UpdateScoreCard(room *Room) {
	message := messages.Message{
		Type:    messages.UpdateScorecard,
		Players: room.Players,
	}
	gc.BroadcastMessageToRoom(room.ID, &message)
}

func (gc *GameController) GameOver(room *Room) {
	if !room.gameStarted {
		return
	}
	room.gameStarted = false

	message := messages.Message{
		Type:    messages.GameOver,
		Players: room.Players,
	}
	gc.BroadcastMessageToRoom(room.ID, &message)
}

func (gc *GameController) CreateRoom(player *game.Player, encoder *gob.Encoder) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	roomID := uuid.New().String()
	room := &Room{
		ID:       roomID,
		Players:  []*game.Player{player},
		encoders: []*gob.Encoder{encoder},
		dices:    game.CreateDices(),
	}
	gc.rooms[roomID] = room

	message := messages.Message{
		Type:   messages.CreateRoom,
		Player: player,
		RoomID: roomID,
	}
	err := encoder.Encode(&message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (gc *GameController) JoinRoom(roomID string, player *game.Player, encoder *gob.Encoder) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.addPlayerToRoom(roomID, player, encoder)

	room, ok := gc.rooms[roomID]
	if !ok {
		log.Println("Room not found:", roomID)
		return
	}

	if len(room.Players) >= 2 {
		gc.StartGame(roomID)
	}
}

func (gc *GameController) addPlayerToRoom(roomID string, player *game.Player, encoder *gob.Encoder) {

	room, ok := gc.rooms[roomID]
	if !ok {
		log.Println("Room not found:", roomID)
		return
	}

	room.Players = append(room.Players, player)
	room.encoders = append(room.encoders, encoder)

	message := messages.Message{
		Type:   messages.JoinRoom,
		Player: player,
		RoomID: roomID,
	}
	err := encoder.Encode(&message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
	gc.notifyPlayerJoinedRoomToOthers(roomID, player, encoder)
}

func (gc *GameController) notifyPlayerJoinedRoomToOthers(roomID string, player *game.Player, encoder *gob.Encoder) {
	room, ok := gc.rooms[roomID]
	if !ok {
		log.Println("Room not found:", roomID)
		return
	}

	message := messages.Message{
		Type:   messages.PlayerJoinedRoom,
		Player: player,
		RoomID: roomID,
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

func (gc *GameController) ListRooms(player *game.Player, encoder *gob.Encoder) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	roomList := make([]string, 0, len(gc.rooms))
	for roomID := range gc.rooms {
		roomList = append(roomList, roomID)
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

func (gc *GameController) HandleMessage(message *messages.Message, player *game.Player, encoder *gob.Encoder) {
	switch message.Type {
	case messages.CreateRoom:
		gc.CreateRoom(player, encoder)
	case messages.JoinRoom:
		gc.addPlayerToRoom(message.RoomID, player, encoder)
	case messages.ListRooms:
		gc.ListRooms(player, encoder)
	case messages.GameLeft:
		room := gc.rooms[message.RoomID]
		gc.LeaveGame(room, player, encoder)
	case messages.TurnStarted:
		room := gc.rooms[message.RoomID]
		gc.StartTurn(room, player, encoder)
	case messages.DiceRolled:
		room := gc.rooms[message.RoomID]
		gc.RollDice(room, player, encoder)
	case messages.RerollDice:
		room := gc.rooms[message.RoomID]
		gc.RerollDice(room, player, message.Dice, encoder)
	case messages.ChooseCategory:
		room := gc.rooms[message.RoomID]
		gc.ChooseScoreCategory(room, player, message.Category, encoder)
	default:
		log.Println("Unknown message type:", message.Type)
	}
}

func (gc *GameController) HandleConnection(encoder *gob.Encoder, decoder *gob.Decoder, player *game.Player) {
	log.Println("Handling connection for player", player.Name)

	for {
		message := &messages.Message{}
		err := decoder.Decode(message)
		if err != nil {
			log.Println("Error decoding message:", err.Error())
			return
		}
		gc.HandleMessage(message, player, encoder)
	}
}

func (gc *GameController) NumberOfConnetedPlayers() int {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()
	gc.playerNum += 1
	return gc.playerNum
}
