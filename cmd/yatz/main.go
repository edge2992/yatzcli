package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edge2992/yatzcli/cli"
	"github.com/edge2992/yatzcli/engine"
	mcpserver "github.com/edge2992/yatzcli/mcp"
)

var rootCmd = &cobra.Command{
	Use:   "yatz",
	Short: "Yahtzee CLI game",
}

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Play a local game against AI",
	RunE: func(cmd *cobra.Command, args []string) error {
		opponents, _ := cmd.Flags().GetInt("opponents")
		playerName, _ := cmd.Flags().GetString("name")

		names := []string{playerName}
		for i := 0; i < opponents; i++ {
			names = append(names, fmt.Sprintf("AI_%d", i+1))
		}

		game := engine.NewGame(names, nil)
		var ais []*engine.AIPlayer
		for i := 1; i <= opponents; i++ {
			pid := fmt.Sprintf("player-%d", i)
			ais = append(ais, engine.NewAIPlayer(game, pid))
		}

		client := engine.NewLocalClient(game, "player-0", ais)
		return cli.RunGame(client, playerName)
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for LLM integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcpserver.Serve()
	},
}

func init() {
	playCmd.Flags().IntP("opponents", "o", 1, "Number of AI opponents (1-3)")
	playCmd.Flags().StringP("name", "n", "Player", "Your player name")
	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(mcpCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
