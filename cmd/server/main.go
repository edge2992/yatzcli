package main

import "yatzcli/server"

func main() {
	server := server.NewServer()
	server.Start()
}
