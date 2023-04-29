package main

import (
	"yatzcli/server"
)

func main() {
	// gob.Register(&messages.Message{})
	// gob.Register(&game.Player{})
	// gob.Register([]*game.Player{})
	// gob.Register(&network.GobConnection{})
	rm := server.NewRoomManager()
	rc := server.NewRoomController(rm)
	gpc := server.NewGamePlayController(rm)
	h := server.NewHandler([]server.Controller{rc, gpc})
	server := server.NewServer(h)
	server.Start()
}
