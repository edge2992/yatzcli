package main

import (
	"fmt"
)

func main() {
	addr := "localhost:8080"
	// conn, err := net.Dial("tcp", addr)
	// if err != nil {
	// 	log.Fatalf("Error connecting to %s: %v", addr, err)
	// }

	fmt.Printf("Connected to Yahtzee server at %s\n", addr)

	// gameClient := client.NewClient(conn)
	// gameClient.Run()
}

// package main

// import (
// 	"fmt"
// )

// func main() {
// 	// Initialize game state
// 	players := createPlayers()
// 	gameState := createGameState(players)

// 	playGame(players, gameState)
// 	displayFinalScores(players)
// }

// func playGame(players []Player, gameState map[string]*Player) {
// 	for round := 0; round < NumberOfRounds; round++ {
// 		for _, player := range players {
// 			fmt.Printf("\n%s's turn (round %d):\n", player.Name, round+1)
// 			displayCurrentScoreboard(players)
// 			playTurn(gameState[player.Name])
// 		}
// 	}
// }
