package server

import (
	"yatzcli/messages"
)

type Controller interface {
	HandleMessage(message *messages.Message, player *Player)
}
