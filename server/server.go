package server

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"

	"yatzcli/game"
	"yatzcli/messages"
)

const (
	MaxPlayers = 2
	Port       = ":8080"
)

type Server struct {
	players       []*game.Player
	gameStarted   bool
	currentPlayer int
	mutex         sync.Mutex
	encoders      []*gob.Encoder
}

// TODO who am i for client

func NewServer() *Server {
	return &Server{
		players:       make([]*game.Player, 0),
		gameStarted:   false,
		currentPlayer: 0,
		mutex:         sync.Mutex{},
		encoders:      make([]*gob.Encoder, 0),
	}
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

		if len(s.encoders) < MaxPlayers {
			s.encoders = append(s.encoders, encoder)
			player := game.NewPlayer("Player " + strconv.Itoa(len(s.encoders)))
			s.players = append(s.players, player)

			go s.handleConnection(encoder, decoder, player)
		} else {
			log.Println("Maximum number of players reached")
			conn.Close()
		}
	}
}

func (s *Server) handleConnection(encoder *gob.Encoder, decoder *gob.Decoder, player *game.Player) {
	log.Println("Handling connection for player", player.Name)
	for {
		message := &messages.Message{}
		err := decoder.Decode(message)
		if err != nil {
			log.Println("Error decoding message:", err.Error())
			break
		}
		log.Println("Received message:", message)

		switch message.Type {
		case messages.GameJoined:
			s.joinGame(player, encoder)
		case messages.GameLeft:
			s.leaveGame(player, encoder)
		case messages.GameStart:
			s.startGame(player, encoder)
		// case RollDice:
		// 	s.rollDice(player, encoder)
		// case TurnPlayed:
		// 	s.playTurn(player, message.Dice, message.Category, encoder)
		// case messages.UpdateGameState:
		// 	s.updateGameState(player, encoder)
		// case messages.GameOver:
		// 	s.gameOver(player, encoder)
		default:
			fmt.Println("Unknown message type:", message.Type)
		}
	}
}
