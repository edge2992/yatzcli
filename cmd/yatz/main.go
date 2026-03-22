package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edge2992/yatzcli/cli"
	"github.com/edge2992/yatzcli/engine"
	"github.com/edge2992/yatzcli/match"
	mcpserver "github.com/edge2992/yatzcli/mcp"
	"github.com/edge2992/yatzcli/p2p"
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

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Host a P2P game",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		name, _ := cmd.Flags().GetString("name")
		return p2p.RunHost(port, name)
	},
}

var joinCmd = &cobra.Command{
	Use:   "join [address]",
	Short: "Join a P2P game",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		return p2p.RunGuest(args[0], name)
	},
}

var matchCmd = &cobra.Command{
	Use:   "match",
	Short: "Find opponent via matchmaking server",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		serverURL, _ := cmd.Flags().GetString("server")

		port, err := match.GetFreePort()
		if err != nil {
			return fmt.Errorf("failed to get free port: %w", err)
		}

		fmt.Printf("Searching for opponent...\n")
		result, err := match.FindMatch(serverURL, name, port)
		if err != nil {
			return fmt.Errorf("matchmaking failed: %w", err)
		}

		fmt.Printf("Matched with %s!\n", result.OpponentName)

		if result.IsHost {
			return p2p.RunHost(port, name)
		}
		return p2p.RunGuest(result.OpponentAddr, name)
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

	hostCmd.Flags().IntP("port", "p", 9876, "Port to listen on")
	hostCmd.Flags().StringP("name", "n", "Host", "Your player name")
	rootCmd.AddCommand(hostCmd)

	joinCmd.Flags().StringP("name", "n", "Guest", "Your player name")
	rootCmd.AddCommand(joinCmd)

	matchCmd.Flags().StringP("name", "n", "Player", "Your player name")
	matchCmd.Flags().String("server", "", "Matchmaking server WebSocket URL")
	rootCmd.AddCommand(matchCmd)

	rootCmd.AddCommand(mcpCmd)

	serveCmd.Flags().IntP("port", "p", 9876, "Port to listen on")
	serveCmd.Flags().Int("players", 2, "Number of players")
	rootCmd.AddCommand(serveCmd)

	botCmd.Flags().String("addr", "localhost:9876", "Game server address")
	botCmd.Flags().StringP("name", "n", "Claude", "Bot player name")
	botCmd.Flags().String("strategy", "", "Path to strategy file (uses built-in if empty)")
	botCmd.Flags().StringP("model", "m", "claude-haiku-4-5-20251001", "Claude model to use (e.g. claude-haiku-4-5-20251001, claude-sonnet-4-6)")
	rootCmd.AddCommand(botCmd)

	rootCmd.AddCommand(battleCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
