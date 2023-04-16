package server

import (
	"encoding/gob"

	"yatzcli/game"
	"yatzcli/messages"
)

func (s *Server) joinGame(player *game.Player, encoder *gob.Encoder) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.players = append(s.players, player)

	message := messages.Message{
		Type: messages.GameJoined,
		// Players: s.players,
	}
	encoder.Encode(&message)
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
		Type: messages.GameLeft,
		// Players: s.players,
	}
	encoder.Encode(&message)
}

func (s *Server) startGame(player *game.Player, encoder *gob.Encoder) {
	// You can send a message to all clients using the encoder.Encode() method
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.gameStarted {
		return
	}

	s.gameStarted = true
	s.currentPlayer = 0

	message := messages.Message{
		Type:          messages.GameStarted,
		CurrentPlayer: s.players[s.currentPlayer].Name,
	}
	encoder.Encode(&message)
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

// func (s *Server) updateGameState(player *game.Player, encoder *gob.Encoder) {
// 	s.mutex.Lock()
// 	defer s.mutex.Unlock()

// 	message := messages.Message{
// 		Type:          messages.UpdateGameState,
// 		Players:       s.players,
// 		CurrentPlayer: s.players[s.currentPlayer].Name,
// 	}
// 	encoder.Encode(&message)
// }

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
