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
	readyPlayers  int
	dices         []game.Dice
	diceRolls     int
}

// TODO who am i for client

func NewServer() *Server {
	return &Server{
		players:       make([]*game.Player, 0),
		gameStarted:   false,
		currentPlayer: 0,
		mutex:         sync.Mutex{},
		encoders:      make([]*gob.Encoder, 0),
		dices:         make([]game.Dice, game.NumberOfDice),
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

		switch message.Type {
		case messages.GameJoined:
			s.joinGame(player, encoder)
		case messages.PlayerReady:
			s.playerReady(player, encoder)
		case messages.GameLeft:
			s.leaveGame(player, encoder)
		case messages.TurnStarted:
			s.startTurn(player, encoder)
		case messages.DiceRolled:
			s.rollDice(player, encoder)
		case messages.RerollDice:
			s.rerollDice(player, message.Dice, encoder)
		case messages.ChooseCategory:
			s.chooseCategory(player, message.Category, encoder)
		// case messages.UpdateScorecard:
		// 	s.updateScorecard(player, encoder)
		// case messages.GameOver:
		// 	s.gameOver(player, encoder)
		default:
			fmt.Println("Unknown message type:", message.Type)
		}
	}
}
