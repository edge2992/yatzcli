package main

import (
	"log"

	"yatzcli/client"
)

func main() {
	conn, err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
	ioHandler := &client.ConsoleIOHandler{}
	c := client.NewClient(conn, ioHandler)
	c.Run()
}
