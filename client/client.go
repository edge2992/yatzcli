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

func (c *Client) Run() {
	defer c.connection.Close()

	log.Println("Connected to server")

	// TODO :: プレイヤーを作成するタイミングをコネクションを取った時点に変更する
	// 問題: second playerの挙動がセグフォになっている

	choice := c.ioHandler.askJoinOrCreateRoom()

	switch choice {
	case CreateRoom:
		roomName := c.ioHandler.askRoomName()
		c.sendCreateRoomMessage(roomName)
	case JoinRoom:
		roomList := c.requestRoomList()
		selectedRoom := c.ioHandler.askRoomSelection(roomList)
		c.sendJoinRoomMessage(selectedRoom)
	}

	for {
		message := &messages.Message{}
		err := c.connection.Decode(message)
		if err != nil {
			fmt.Println("Error decoding message:", err.Error())
			break
		}

		switch message.Type {

		case messages.CreateRoom:
			c.Player = message.Player
			log.Println("Room created: ", message.RoomID)
			// log.Println("Player created: ", c.Player.Name)
		case messages.JoinRoom:
			log.Println("Room joined: ", message.Player.Name)
		case messages.GameJoined:
			log.Println("Game joined by: ", message.Player.Name)
			c.Player = message.Player
			c.setReady()
		case messages.PlayerJoined:
			log.Println("Player joined: ", message.Player.Name)
		case messages.PlayerLeft:
			log.Println("Player left: ", message.Player.Name)
		case messages.GameStarted:
			log.Println("Game started")
			if c.Player.Name == message.Player.Name {
				message := &messages.Message{
					Type:   messages.TurnStarted,
					RoomID: message.RoomID,
				}
				c.connection.Encode(message)
			}
		case messages.UpdateScorecard:
			c.handleUpdateScorecard(message)
		case messages.TurnStarted:
			c.handleTurnStarted(message)
		case messages.DiceRolled:
			c.handleDiceRolled(message)
		case messages.GameOver:
			log.Println("Game over")
			// TODO - display winner
		default:
			fmt.Println("Unknown message type:", message.Type)
		}
	}
}

func (c *Client) sendCreateRoomMessage(roomName string) {
	message := messages.Message{
		Type:   messages.CreateRoom,
		RoomID: roomName, // ignored by server
	}
	c.connection.Encode(&message)
}

func (c *Client) requestRoomList() []string {
	message := messages.Message{
		Type: messages.ListRooms,
	}
	c.connection.Encode(&message)

	response := &messages.Message{}
	c.connection.Decode(response)

	return response.RoomList
}

func (c *Client) sendJoinRoomMessage(roomID string) {
	message := messages.Message{
		Type:   messages.JoinRoom,
		RoomID: roomID,
	}
	c.connection.Encode(&message)
}

func (c *Client) setReady() {
	readyMessage := messages.Message{
		Type: messages.PlayerReady,
	}
	c.connection.Encode(&readyMessage)
}

func (c *Client) handleUpdateScorecard(message *messages.Message) {
	players := make([]game.PlayerInfo, 0)
	for _, player := range message.Players {
		players = append(players, *player)
	}
	c.ioHandler.DisplayCurrentScoreboard(players)
}

func (c *Client) handleTurnStarted(message *messages.Message) {
	log.Println("It's your turn!")
	c.turnFlag = true
	hmessage := messages.Message{
		Type: messages.DiceRolled,
	}
	c.connection.Encode(&hmessage)
}

func (c *Client) handleDiceRolled(message *messages.Message) {
	c.ioHandler.DisplayDice(message.Dice)
	if c.turnFlag {
		if message.DiceRolls < game.MaxRolls {
			c.reRollDice(message.Dice)
		} else {
			c.chooseCategory(message.Player, message.Dice)
		}
	}
}

func (c *Client) reRollDice(dice []game.Dice) {
	selectedIndices := c.ioHandler.GetPlayerHoldInput(dice)
	game.HoldDice(dice, selectedIndices)
	message := messages.Message{
		Type: messages.RerollDice,
		Dice: dice,
	}
	c.connection.Encode(&message)
}

func (c *Client) chooseCategory(player *game.PlayerInfo, dice []game.Dice) {
	category := c.ioHandler.ChooseCategory(player, dice)
	message := messages.Message{
		Type:     messages.ChooseCategory,
		Player:   player,
		Category: category,
	}
	c.connection.Encode(&message)
	c.turnFlag = false
}
