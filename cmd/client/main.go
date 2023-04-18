package main

import (
	"yatzcli/client"
)

func main() {
	client := client.NewClient()
	client.Connect()
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
