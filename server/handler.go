package server

import (
	"log"
	"yatzcli/messages"
)

type Handler struct {
	connectedPlayers int
	controllers      []Controller
}

func NewHandler(controllers []Controller) *Handler {
	return &Handler{
		connectedPlayers: 0,
		controllers:      controllers,
	}
}

func (h *Handler) HandleConnection(player *Player) {
	defer func() {
		h.connectedPlayers--
	}()
	log.Println("Handling connection for player", player.Name)
	h.connectedPlayers++

	for {
		message := &messages.Message{}
		err := player.Connection.Decode(message)
		if err != nil {
			log.Println("Error decoding message:", err.Error())
			return
		}
		for _, controller := range h.controllers {
			controller.HandleMessage(message, player)
		}
	}
}

func (h *Handler) NumberOfConnetedPlayers() int {
	return h.connectedPlayers
}
