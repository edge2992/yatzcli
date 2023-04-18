package client

import (
	"encoding/gob"
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
	connection net.Conn
	Player     *game.Player
	turnFlag   bool
}

func NewClient() *Client {
	return &Client{turnFlag: false}
}

func (c *Client) Connect() {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Println("Error connecting:", err.Error())
		panic(err)
	}
	c.connection = conn
	defer conn.Close()

	log.Println("Connected to server")

	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)

	joinMessage := messages.Message{
		Type: messages.GameJoined,
	}
	encoder.Encode(&joinMessage)

	for {
		message := &messages.Message{}
		err := decoder.Decode(message)
		if err != nil {
			fmt.Println("Error decoding message:", err.Error())
			break
		}

		switch message.Type {
		case messages.GameJoined:
			log.Println("Game joined by: ", message.Player.Name)
			c.Player = message.Player
			c.setReady(encoder)
		case messages.PlayerJoined:
			log.Println("Player joined: ", message.Player.Name)
		case messages.PlayerLeft:
			log.Println("Player left: ", message.Player.Name)
		case messages.GameStarted:
			log.Println("Game started")
		case messages.UpdateScorecard:
			c.handleUpdateScorecard(message)
		case messages.TurnStarted:
			c.handleTurnStarted(message, encoder)
		case messages.DiceRolled:
			c.handleDiceRolled(message, encoder)
		case messages.GameOver:
			log.Println("Game over")
			// TODO - display winner
		default:
			fmt.Println("Unknown message type:", message.Type)
		}
	}
}

func (c *Client) setReady(encoder *gob.Encoder) {
	readyMessage := messages.Message{
		Type: messages.PlayerReady,
	}
	encoder.Encode(&readyMessage)
}

func (c *Client) handleUpdateScorecard(message *messages.Message) {
	players := make([]game.Player, 0)
	for _, player := range message.Players {
		players = append(players, *player)
	}
	game.DisplayCurrentScoreboard(players)
}

func (c *Client) handleTurnStarted(message *messages.Message, encoder *gob.Encoder) {
	log.Println("It's your turn!")
	c.turnFlag = true
	hmessage := messages.Message{
		Type: messages.DiceRolled,
	}
	encoder.Encode(&hmessage)
}

func (c *Client) handleDiceRolled(message *messages.Message, encoder *gob.Encoder) {
	game.DisplayDice(message.Dice)
	if c.turnFlag {
		if message.DiceRolls < game.MaxRolls {
			c.reRollDice(message.Dice, encoder)
		} else {
			c.chooseCategory(message.Player, message.Dice, encoder)
		}
	}
}

func (c *Client) reRollDice(dice []game.Dice, encoder *gob.Encoder) {
	selectedIndices := game.GetPlayerHoldInput(dice)
	game.HoldDice(dice, selectedIndices)
	message := messages.Message{
		Type: messages.RerollDice,
		Dice: dice,
	}
	encoder.Encode(&message)
}

func (c *Client) chooseCategory(player *game.Player, dice []game.Dice, encoder *gob.Encoder) {
	category := game.ChooseCategory(player, dice)
	message := messages.Message{
		Type:     messages.ChooseCategory,
		Category: category,
	}
	encoder.Encode(&message)
	c.turnFlag = false
}
