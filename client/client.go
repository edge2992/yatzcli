package client

import (
	"fmt"
	"log"
	"net"

	"yatzcli/game"
	"yatzcli/messages"
	"yatzcli/network"
)

const (
	serverAddress = "localhost:8080"
)

type Client struct {
	connection network.Connection
	Player     *game.PlayerInfo
	ioHandler  IOHandler
	turnFlag   bool
}

func NewClient(conn network.Connection, ioHandler IOHandler) *Client {
	return &Client{connection: conn, ioHandler: ioHandler, turnFlag: false}
}

func Connect() (network.Connection, error) {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		return nil, err
	}

	gobConnection := network.NewGobConnection(conn)

	return gobConnection, nil
}

func (c *Client) sendMessage(message *messages.Message) {
	err := c.connection.Encode(message)
	if err != nil {
		log.Println("Error sending message:", err.Error())
	}
}

func (c *Client) waitForServerJoin() error {
	message := &messages.Message{}
	err := c.connection.Decode(message)
	if err != nil {
		return err
	}
	return c.handleServerJoinMessage(message)
}

func (c *Client) handleServerJoinMessage(message *messages.Message) error {
	if message.Type != messages.ServerJoin {
		return fmt.Errorf("expected ServerJoin message, got %d", message.Type)
	}
	c.Player = message.Player
	return nil
}

func (c *Client) joinOrCreateRoom() {
	choice := c.ioHandler.askJoinOrCreateRoom()

	switch choice {
	case CreateRoom:
		roomName := c.ioHandler.askRoomName()
		c.CreateRoom(roomName)
	case JoinRoom:
		roomList := c.requestRoomList()
		selectedRoom := c.ioHandler.askRoomSelection(roomList)
		c.JoinRoom(selectedRoom)
	}
}

func (c *Client) handleMessages() {
	for {
		message := &messages.Message{}
		err := c.connection.Decode(message)
		if err != nil {
			fmt.Println("Error decoding message:", err.Error())
			break
		}
		c.processMessage(message)
	}
}

func (c *Client) processMessage(message *messages.Message) {
	switch message.Type {
	case messages.RoomCreated:
		c.handleCreateRoomMessage(message)
	case messages.RoomJoined:
		c.handleJoinRoomMessage(message)
	case messages.RoomLeft:
		c.handleLeaveRoomMessage(message)
	case messages.GameStarted:
		c.handleGameStartedMessage(message)
	case messages.UpdateScorecard:
		c.handleUpdateScorecard(message)
	case messages.TurnStarted:
		c.handleTurnStarted(message.RoomID)
	case messages.DiceRolled:
		c.handleDiceRolled(message)
	case messages.GameOver:
		c.handleGameOverMessage(message)
	default:
		fmt.Println("Unknown message type:", message.Type)
	}
}

func (c *Client) Run() {
	defer c.connection.Close()
	log.Println("Connected to server")

	if err := c.waitForServerJoin(); err != nil {
		fmt.Println("Error waiting for server join:", err.Error())
		return
	}

	c.joinOrCreateRoom()
	c.handleMessages()
}

func (c *Client) CreateRoom(roomName string) {
	message := messages.Message{
		Type:   messages.RequestCreateRoom,
		RoomID: roomName, // ignored by server
	}
	c.connection.Encode(&message)
}

func (c *Client) requestRoomList() []string {
	message := messages.Message{
		Type: messages.RequestRoomList,
	}
	c.sendMessage(&message)

	response := &messages.Message{}
	c.connection.Decode(response)

	return response.RoomList
}

func (c *Client) JoinRoom(roomID string) {
	message := messages.Message{
		Type:   messages.RequestJoinRoom,
		RoomID: roomID,
	}
	c.sendMessage(&message)
}

func (c *Client) handleCreateRoomMessage(message *messages.Message) {
	log.Println("Room created: ", message.RoomID)
}

func (c *Client) handleJoinRoomMessage(message *messages.Message) {
	log.Println("Room joined: ", message.RoomID, message.Player.Name)
}

func (c *Client) handleLeaveRoomMessage(message *messages.Message) {
	log.Println("Player left: ", message.Player.Name)
}

func (c *Client) handleGameStartedMessage(message *messages.Message) {
	log.Println("Game started")
	if c.Player.Name == message.Player.Name {
		message := &messages.Message{
			Type:   messages.TurnStarted,
			RoomID: message.RoomID,
		}
		c.sendMessage(message)
	}
}

func (c *Client) handleGameOverMessage(message *messages.Message) {
	log.Println("Game over")
	// TODO - display winner
}

func (c *Client) handleUpdateScorecard(message *messages.Message) {
	players := make([]game.PlayerInfo, 0)
	for _, player := range message.Players {
		players = append(players, *player)
	}
	c.ioHandler.DisplayCurrentScoreboard(players)
}

func (c *Client) handleTurnStarted(roomID string) {
	log.Println("It's your turn!")
	c.turnFlag = true
	message := messages.Message{
		Type:   messages.RequestRollDice,
		RoomID: roomID,
	}
	c.sendMessage(&message)
}

func (c *Client) handleDiceRolled(message *messages.Message) {
	c.ioHandler.DisplayDice(message.Dice)
	if c.turnFlag {
		if message.DiceRolls < game.MaxRolls {
			c.reRollDice(message.Dice, message.RoomID)
		} else {
			c.chooseCategory(message.Player, message.Dice, message.RoomID)
		}
	}
}

func (c *Client) reRollDice(dice []game.Dice, roomID string) {
	selectedIndices := c.ioHandler.GetPlayerHoldInput(dice)
	game.HoldDice(dice, selectedIndices)
	message := messages.Message{
		Type:   messages.RequestRerollDice,
		RoomID: roomID,
		Dice:   dice,
	}
	c.sendMessage(&message)
}

func (c *Client) chooseCategory(player *game.PlayerInfo, dice []game.Dice, roomID string) {
	category := c.ioHandler.ChooseCategory(player, dice)
	message := messages.Message{
		Type:     messages.RequestChooseCategory,
		RoomID:   roomID,
		Player:   player,
		Category: category,
	}
	c.sendMessage(&message)
	c.turnFlag = false
}
