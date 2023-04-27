package server

import (
	"log"
	"net"
	"strconv"

	"yatzcli/game"
	"yatzcli/network"
)

const (
	MaxPlayers = 2
	Port       = ":8080"
)

type ConnectionHandler interface {
	HandleConnection(player *game.Player)
	NumberOfConnetedPlayers() int
}

type Server struct {
	handler ConnectionHandler
}

func NewServer(handler ConnectionHandler) *Server {
	server := &Server{
		handler: handler,
	}
	return server
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", Port)
	if err != nil {
		log.Printf("Error listening: %v", err.Error())
		return
	}
	defer listener.Close()

	log.Println("Listening for clients on port", Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			return
		}
		log.Println("Client connected")

		gobConn := network.NewGobConnection(conn)
		playerName := "Player " + strconv.Itoa(s.handler.NumberOfConnetedPlayers())
		player := game.NewPlayer(playerName, gobConn)

		go s.handler.HandleConnection(player)
	}
}
