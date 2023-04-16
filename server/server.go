package server

import (
	"encoding/gob"
	"fmt"
	"net"
	"strconv"
	"sync"

	"yatzcli/game"
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
type Message struct {
	Type          MessageType
	Players       []*game.Player
	currentPlayer string
	Dice          []game.Dice
	Category      game.ScoreCategory
}

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
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer listener.Close()

	fmt.Println("Listening for clients on port", Port)

	for len(s.encoders) < MaxPlayers {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			return
		}

		encoder := gob.NewEncoder(conn)
		decoder := gob.NewDecoder(conn)
		defer conn.Close()

		s.encoders = append(s.encoders, encoder)
		player := game.NewPlayer("Player " + strconv.Itoa(len(s.encoders)))
		s.players = append(s.players, player)

		go s.handleConnection(encoder, decoder, player)
	}
}

func (s *Server) handleConnection(encoder *gob.Encoder, decoder *gob.Decoder, player *game.Player) {
	for {
		message := &Message{}
		err := decoder.Decode(message)
		if err != nil {
			fmt.Println("Error decoding message:", err.Error())
			break
		}

		switch message.Type {
		case GameJoined:
			s.joinGame(player, encoder)
		case GameLeft:
			s.leaveGame(player, encoder)
		case GameStart:
			s.startGame(player, encoder)
		// case RollDice:
		// 	s.rollDice(player, encoder)
		// case TurnPlayed:
		// 	s.playTurn(player, message.Dice, message.Category, encoder)
		case UpdateGameState:
			s.updateGameState(player, encoder)
		case GameOver:
			s.gameOver(player, encoder)
		default:
			fmt.Println("Unknown message type:", message.Type)
		}
	}
}
