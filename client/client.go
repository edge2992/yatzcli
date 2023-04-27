package client

import (
	"fmt"
	"log"
	"net"

	"yatzcli/game"
	"yatzcli/messages"
)

const (
	serverAddress = "localhost:8080"
)

type Client struct {
	connection messages.Connection
	Player     *game.Player
	ioHandler  IOHandler
	turnFlag   bool
}

func NewClient(conn messages.Connection, ioHandler IOHandler) *Client {
	return &Client{connection: conn, ioHandler: ioHandler, turnFlag: false}
}

func Connect() (messages.Connection, error) {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		return nil, err
	}

	gobConnection := messages.NewGobConnection(conn)
	return gobConnection, nil
}

func (c *Client) Run() {
	defer c.connection.Close()

	log.Println("Connected to server")

	joinMessage := messages.Message{
		Type: messages.GameJoined,
	}
	c.connection.Encode(&joinMessage)

	for {
		message := &messages.Message{}
		err := c.connection.Decode(message)
		if err != nil {
			fmt.Println("Error decoding message:", err.Error())
			break
		}

		switch message.Type {
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

func (c *Client) setReady() {
	readyMessage := messages.Message{
		Type: messages.PlayerReady,
	}
	c.connection.Encode(&readyMessage)
}

func (c *Client) handleUpdateScorecard(message *messages.Message) {
	players := make([]game.Player, 0)
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

func (c *Client) chooseCategory(player *game.Player, dice []game.Dice) {
	category := c.ioHandler.ChooseCategory(player, dice)
	message := messages.Message{
		Type:     messages.ChooseCategory,
		Player:   player,
		Category: category,
	}
	c.connection.Encode(&message)
	c.turnFlag = false
}
