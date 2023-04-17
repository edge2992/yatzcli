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
}

func NewClient() *Client {
	return &Client{}
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
