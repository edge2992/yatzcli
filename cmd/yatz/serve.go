package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/spf13/cobra"

	"github.com/edge2992/yatzcli/p2p"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a headless game server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		players, _ := cmd.Flags().GetInt("players")

		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		defer ln.Close()

		fmt.Printf("Game server listening on port %d, waiting for %d players...\n", port, players)
		return p2p.RunServer(ln, players, rand.NewSource(time.Now().UnixNano()))
	},
}
