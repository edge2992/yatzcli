package main

import (
	"yatzcli/server"
)

func main() {
	gc := server.NewGameController()
	server := server.NewServer(gc)
	server.Start()
}
