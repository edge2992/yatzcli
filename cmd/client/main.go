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
	c := client.NewClient(conn)
	c.Run()
}
