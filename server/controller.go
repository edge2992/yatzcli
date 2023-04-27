package server

import (
	"yatzcli/game"
	"yatzcli/messages"
)

type Controller interface {
	HandleMessage(message *messages.Message, player *game.Player)
}
