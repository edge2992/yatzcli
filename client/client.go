package client

import (
	"fmt"
	"net"

	"yatzcli/game"
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
		fmt.Println("Error connecting:", err.Error())
		panic(err)
	}
	c.connection = conn
	defer conn.Close()

	fmt.Println("Connected to server")

	// TODO: Implement communication betwrrn server and client
}
