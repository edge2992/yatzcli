package server

import (
	"encoding/gob"
	"log"
	"net"
	"strconv"

	"yatzcli/game"
)

const (
	MaxPlayers = 2
	Port       = ":8080"
)

type ConnectionHandler interface {
	HandleConnection(encoder *gob.Encoder, decoder *gob.Decoder, player *game.Player)
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

		encoder := gob.NewEncoder(conn)
		decoder := gob.NewDecoder(conn)

		player := game.NewPlayer("Player " + strconv.Itoa(s.handler.NumberOfConnetedPlayers()))

		go s.handler.HandleConnection(encoder, decoder, player)
	}
}
