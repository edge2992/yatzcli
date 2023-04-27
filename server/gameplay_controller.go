package server

import (
	"fmt"
	"log"
	"yatzcli/game"
	"yatzcli/messages"
)

type GamePlayController struct {
	roomManager *RoomManager
}

func NewGamePlayController(rm *RoomManager) *GamePlayController {
	return &GamePlayController{
		roomManager: rm,
	}
}

func (gpc *GamePlayController) BroadcastMessageToRoom(room *Room, message *messages.Message) {
	for _, player := range room.Players {
		err := player.Connection.Encode(message)
		if err != nil {
			fmt.Println("Error encoding message:", err.Error())
		}
	}
}

func (gpc *GamePlayController) StartGame(roomID string) {
	room, err := gpc.roomManager.GetRoom(roomID)
	if err != nil {
		log.Println("Error getting room:", err.Error())
		return
	}
	if room.gameStarted {
		log.Println("Game already started")
		return
	}
	room.gameStarted = true
	room.currentPlayerId = 0

	message := messages.Message{
		Type: messages.GameStarted,
	}
	gpc.BroadcastMessageToRoom(room, &message)

	gpc.StartTurn(roomID, room.Players[room.currentPlayerId])
}

func (gpc *GamePlayController) StartTurn(roomID string, player *game.Player) {
	room, err := gpc.roomManager.GetRoom(roomID)
	if err != nil {
		log.Println("Error getting room:", err.Error())
		return
	}
	gpc.UpdateScoreCard(roomID)
	if room.gameTurnNum == game.NumberOfRounds*len(room.Players) {
		gpc.GameOver(roomID)
		return
	}
	if room.Players[room.currentPlayerId] != player {
		return
	}
	room.gameTurnNum += 1

	for i := 0; i < game.NumberOfDice; i++ {
		room.dices[i].Held = false
	}
	room.diceRolls = 0

	message := messages.Message{
		Type:   messages.TurnStarted,
		Player: player,
	}
	errEncode := player.Connection.Encode(&message)
	if errEncode != nil {
		log.Println("Error encoding message:", errEncode.Error())
	}
}

func (gpc *GamePlayController) GameOver(roomID string) {
	room, err := gpc.roomManager.GetRoom(roomID)
	if err != nil {
		log.Println("Error getting room:", err.Error())
		return
	}
	if !room.gameStarted {
		return
	}
	room.gameStarted = false

	message := messages.Message{
		Type:    messages.GameOver,
		Players: room.Players,
	}
	gpc.BroadcastMessageToRoom(room, &message)
}

func (gpc *GamePlayController) RollDice(roomID string, player *game.Player) {
	room, err := gpc.roomManager.GetRoom(roomID)
	if err != nil {
		log.Println("Error getting room:", err.Error())
		return
	}
	room.diceRolls += 1
	game.RollDice(room.dices)

	message := messages.Message{
		Type:      messages.DiceRolled,
		Player:    player,
		Dice:      room.dices,
		DiceRolls: room.diceRolls,
	}
	gpc.BroadcastMessageToRoom(room, &message)
}

func (gpc *GamePlayController) RerollDice(roomID string, player *game.Player, dice []game.Dice) {
	room, err := gpc.roomManager.GetRoom(roomID)
	if err != nil {
		log.Println("Error getting room:", err.Error())
		return
	}
	if room.diceRolls >= game.NumberOfDice {
		// TODO: Send error message
		log.Println("Cannot reroll dice more than", game.NumberOfDice, "times")
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

	game.HoldDice(room.dices, selectedIndices)
	gpc.RollDice(roomID, player)
}

func (gpc *GamePlayController) ChooseScoreCategory(roomID string, player *game.Player, category game.ScoreCategory) {
	room, err := gpc.roomManager.GetRoom(roomID)
	if err != nil {
		log.Println("Error getting room:", err.Error())
		return
	}
	if room.Players[room.currentPlayerId] != player {
		return
	}

	score := game.CalculateScore(room.dices, category)
	player.ScoreCard.Scores[category] = score
	player.ScoreCard.Filled[category] = true

	room.currentPlayerId = (room.currentPlayerId + 1) % len(room.Players)
	gpc.StartTurn(roomID, room.Players[room.currentPlayerId])
}

func (gpc *GamePlayController) UpdateScoreCard(roomID string) {
	room, err := gpc.roomManager.GetRoom(roomID)
	if err != nil {
		log.Println("Error getting room:", err.Error())
		return
	}
	message := messages.Message{
		Type:    messages.UpdateScorecard,
		Players: room.Players,
	}
	gpc.BroadcastMessageToRoom(room, &message)
}

func (gpc *GamePlayController) HandleMessage(message *messages.Message, player *game.Player) {
	switch message.Type {
	case messages.TurnStarted:
		gpc.StartTurn(message.RoomID, player)
	case messages.DiceRolled:
		gpc.RollDice(message.RoomID, player)
	case messages.RerollDice:
		gpc.RerollDice(message.RoomID, player, message.Dice)
	case messages.ChooseCategory:
		gpc.ChooseScoreCategory(message.RoomID, player, message.Category)
	default:
		log.Println("Unknown message type:", message.Type)
	}

}
