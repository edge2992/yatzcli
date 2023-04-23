package server

import (
	"encoding/gob"
	"fmt"
	"log"
	"sync"
	"yatzcli/game"
	"yatzcli/messages"
)

type GameController struct {
	players       []*game.Player
	gameStarted   bool
	gameTurnNum   int
	currentPlayer int
	mutex         sync.Mutex
	encoders      []*gob.Encoder
	readyPlayers  int
	dices         []game.Dice
	diceRolls     int
}

func NewGameController() *GameController {
	return &GameController{
		players:       []*game.Player{},
		gameStarted:   false,
		gameTurnNum:   0,
		currentPlayer: 0,
		encoders:      []*gob.Encoder{},
		readyPlayers:  0,
		dices:         game.CreateDices(),
		diceRolls:     0,
	}
}

func (gc *GameController) BroadcastMessage(message *messages.Message) {
	// gc.mutex.Lock()
	// defer gc.mutex.Unlock()
	for _, encoder := range gc.encoders {
		err := encoder.Encode(message)
		if err != nil {
			fmt.Println("Error encoding message:", err.Error())
		}
	}
}

func (gc *GameController) JoinGame(player *game.Player, encoder *gob.Encoder) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.players = append(gc.players, player)
	gc.encoders = append(gc.encoders, encoder)

	message := messages.Message{
		Type:   messages.GameJoined,
		Player: player,
	}
	err := encoder.Encode(&message)
	if err != nil {
		fmt.Println("Error encoding message:", err.Error())
	}

	gc.PlayerJoined(player)
}

func (gc *GameController) PlayerJoined(player *game.Player) {
	message := messages.Message{
		Type:   messages.PlayerJoined,
		Player: player,
	}
	gc.BroadcastMessage(&message)
}

func (gc *GameController) PlayerReady(player *game.Player, encoder *gob.Encoder) {
	gc.mutex.Lock()
	gc.readyPlayers++
	gc.mutex.Unlock()

	// rough implementation of starting the game
	// when two players are ready
	if gc.readyPlayers >= 2 {
		gc.StartGame()
	}
}

func (gc *GameController) StartGame() {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if gc.gameStarted {
		return
	}
	gc.gameStarted = true
	gc.currentPlayer = 0

	message := messages.Message{
		Type: messages.GameStarted,
	}
	gc.BroadcastMessage(&message)

	gc.StartTurn(gc.players[gc.currentPlayer], gc.encoders[gc.currentPlayer])
}

func (gc *GameController) LeaveGame(player *game.Player, encoder *gob.Encoder) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	for i, p := range gc.players {
		if p.Name == player.Name {
			gc.players = append(gc.players[:i], gc.players[i+1:]...)
			break
		}
	}

	message := messages.Message{
		Type:   messages.GameLeft,
		Player: player,
	}
	err := encoder.Encode(&message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (gc *GameController) StartTurn(player *game.Player, encoder *gob.Encoder) {
	gc.UpdateScoreCard()
	if gc.gameTurnNum == game.NumberOfRounds*len(gc.players) {
		gc.GameOver()
		return
	}
	if gc.players[gc.currentPlayer] != player {
		return
	}
	gc.gameTurnNum += 1

	for i := 0; i < game.NumberOfDice; i++ {
		gc.dices[i].Held = false
	}
	gc.diceRolls = 0

	message := messages.Message{
		Type:   messages.TurnStarted,
		Player: player,
	}
	err := encoder.Encode(&message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (gc *GameController) RollDice(player *game.Player, encoder *gob.Encoder) {
	gc.diceRolls += 1
	game.RollDice(gc.dices)

	message := messages.Message{
		Type:      messages.DiceRolled,
		Player:    player,
		Dice:      gc.dices,
		DiceRolls: gc.diceRolls,
	}
	gc.BroadcastMessage(&message)
}

func (gc *GameController) RerollDice(player *game.Player, dice []game.Dice, encoder *gob.Encoder) {
	if gc.diceRolls >= game.NumberOfDice {
		// TODO: Send error message
		return
	}
	// rough implementation of rerolling dice
	// Don't trust the dice numbers returned from the client
	// trust server's dice numbers
	selectedIndices := make([]int, 0)
	for i, d := range dice {
		if d.Held {
			selectedIndices = append(selectedIndices, i)
		}
	}

	game.HoldDice(gc.dices, selectedIndices)
	gc.RollDice(player, encoder)
}

func (gc *GameController) ChooseScoreCategory(player *game.Player, category game.ScoreCategory, encoder *gob.Encoder) {
	if gc.players[gc.currentPlayer] != player {
		return
	}

	score := game.CalculateScore(gc.dices, category)
	player.ScoreCard.Scores[category] = score
	player.ScoreCard.Filled[category] = true

	gc.currentPlayer = (gc.currentPlayer + 1) % len(gc.players)
	gc.StartTurn(gc.players[gc.currentPlayer], gc.encoders[gc.currentPlayer])
}

func (gc *GameController) UpdateScoreCard() {
	message := messages.Message{
		Type:    messages.UpdateScorecard,
		Players: gc.players,
	}
	gc.BroadcastMessage(&message)
}

func (gc *GameController) GameOver() {
	if !gc.gameStarted {
		return
	}
	gc.gameStarted = false

	message := messages.Message{
		Type:    messages.GameOver,
		Players: gc.players,
	}
	gc.BroadcastMessage(&message)
}

func (gc *GameController) HandleMessage(message *messages.Message, player *game.Player, encoder *gob.Encoder) {
	switch message.Type {
	case messages.GameJoined:
		gc.JoinGame(player, encoder)
	case messages.PlayerReady:
		gc.PlayerReady(player, encoder)
	case messages.GameLeft:
		gc.LeaveGame(player, encoder)
	case messages.TurnStarted:
		gc.StartTurn(player, encoder)
	case messages.DiceRolled:
		gc.RollDice(player, encoder)
	case messages.RerollDice:
		gc.RerollDice(player, message.Dice, encoder)
	case messages.ChooseCategory:
		gc.ChooseScoreCategory(player, message.Category, encoder)
	default:
		log.Println("Unknown message type:", message.Type)
	}
}

func (gc *GameController) HandleConnection(encoder *gob.Encoder, decoder *gob.Decoder, player *game.Player) {
	log.Println("Handling connection for player", player.Name)

	for {
		message := &messages.Message{}
		err := decoder.Decode(message)
		if err != nil {
			log.Println("Error decoding message:", err.Error())
			return
		}
		gc.HandleMessage(message, player, encoder)
	}
}

func (gc *GameController) NumberOfConnetedPlayers() int {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	return len(gc.players)
}
