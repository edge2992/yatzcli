package server

import (
	"encoding/gob"
	"fmt"
	"log"

	"yatzcli/game"
	"yatzcli/messages"
)

func (s *Server) broadcastMessage(message *messages.Message) {
	// s.mutex.Lock()
	// defer s.mutex.Unlock()
	for _, encoder := range s.encoders {
		err := encoder.Encode(message)
		if err != nil {
			fmt.Println("Error encoding message:", err.Error())
		}
	}
}

func (s *Server) joinGame(player *game.Player, encoder *gob.Encoder) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// s.players = append(s.players, player)

	message := messages.Message{
		Type:   messages.GameJoined,
		Player: player,
	}
	err := encoder.Encode(&message)
	if err != nil {
		fmt.Println("Error encoding message:", err.Error())
	}

	s.playerjoined(player, encoder)
}

func (s *Server) playerjoined(player *game.Player, encoder *gob.Encoder) {
	message := messages.Message{
		Type:   messages.PlayerJoined,
		Player: player,
	}
	s.broadcastMessage(&message)

}

func (s *Server) playerReady(player *game.Player, encoder *gob.Encoder) {
	s.mutex.Lock()
	s.readyPlayers++
	s.mutex.Unlock()
	log.Println("Player ready:", player.Name)

	if s.readyPlayers >= 2 {
		s.startGame()
	}
}

func (s *Server) startGame() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.gameStarted {
		return
	}
	s.gameStarted = true
	s.currentPlayer = 0

	message := messages.Message{
		Type: messages.GameStarted,
	}
	s.broadcastMessage(&message)

	s.startTurn(s.players[s.currentPlayer], s.encoders[s.currentPlayer])
}

func (s *Server) leaveGame(player *game.Player, encoder *gob.Encoder) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i, p := range s.players {
		if p.Name == player.Name {
			s.players = append(s.players[:i], s.players[i+1:]...)
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

func (s *Server) startTurn(player *game.Player, encoder *gob.Encoder) {
	log.Println("The number of players is:", len(s.players))
	s.updateScoreCard(player, encoder)
	if s.players[s.currentPlayer] != player {
		return
	}

	for i := 0; i < game.NumberOfDice; i++ {
		s.dices[i].Held = false
	}
	s.diceRolls = 0

	message := messages.Message{
		Type:   messages.TurnStarted,
		Player: player,
	}
	err := encoder.Encode(&message)
	if err != nil {
		log.Println("Error encoding message:", err.Error())
	}
}

func (s *Server) rollDice(player *game.Player, encoder *gob.Encoder) {
	s.diceRolls += 1
	game.RollDice(s.dices)

	message := messages.Message{
		Type:      messages.DiceRolled,
		Player:    player,
		Dice:      s.dices,
		DiceRolls: s.diceRolls,
	}
	s.broadcastMessage(&message)
}

func (s *Server) rerollDice(player *game.Player, dice []game.Dice, encoder *gob.Encoder) {
	if s.diceRolls >= game.NumberOfDice {
		// TODO: Send error message
		return
	}
	selectedIndices := make([]int, 0)
	for i, d := range dice {
		if d.Held {
			selectedIndices = append(selectedIndices, i)
		}
	}

	game.HoldDice(s.dices, selectedIndices)
	s.rollDice(player, encoder)
}

func (s *Server) chooseCategory(player *game.Player, category game.ScoreCategory, encoder *gob.Encoder) {
	if s.players[s.currentPlayer] != player {
		return
	}

	score := game.CalculateScore(s.dices, category)
	player.ScoreCard.Scores[category] = score
	player.ScoreCard.Filled[category] = true

	s.currentPlayer = (s.currentPlayer + 1) % len(s.players)
	s.startTurn(s.players[s.currentPlayer], s.encoders[s.currentPlayer])
}

// func (s *Server) playTurn(player *game.Player, dice []game.Dice, category game.ScoreCategory, encoder *gob.Encoder) {
// 	// Update the game state and inform other clients about the turn
// 	s.mutex.Lock()
// 	defer s.mutex.Unlock()

// 	if !s.gameStarted || s.players[s.currentPlayer].Name != player.Name {
// 		return
// 	}

// 	dice := make([]game.Dice, NumberOfDice)
// 	game.rollDice(dice)

// 	message := Message{
// 		Type:  RollDice,
// 		Dice:  dice,
// 		Rolls: 1,
// 	}
// 	encoder.Encode(&message)
// }

// 	score := game.calculateScore(dice, category)

// 	s.players[s.currentPlayer].ScoreCard.Scores[category] = &score
// 	s.currentPlayer = (s.currentPlayer + 1) % len(s.players)

// 	message := Message{
// 		Type:          TurnPlayed,
// 		Players:       s.players,
// 		currentPlayer: s.players[s.currentPlayer].Name,
// 	}
// 	encoder.Encode(&message)
// }

func (s *Server) updateScoreCard(player *game.Player, encoder *gob.Encoder) {
	message := messages.Message{
		Type:          messages.UpdateScorecard,
		Players:       s.players,
		CurrentPlayer: s.players[s.currentPlayer].Name,
	}
	s.broadcastMessage(&message)
}

// func (s *Server) gameOver(player *game.Player, encoder *gob.Encoder) {
// 	s.mutex.Lock()
// 	defer s.mutex.Unlock()

// 	if !s.gameStarted {
// 		return
// 	}
// 	s.gameStarted = false

// 	message := messages.Message{
// 		Type:    messages.GameOver,
// 		Players: s.players,
// 	}
// 	encoder.Encode(&message)
// }
