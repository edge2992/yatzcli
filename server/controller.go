package server

import (
	"encoding/gob"
	"yatzcli/game"
	"yatzcli/messages"
)

type Controller interface {
	HandleMessage(message *messages.Message, player *game.Player, encoder *gob.Encoder)
}
