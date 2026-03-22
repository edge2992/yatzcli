package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edge2992/yatzcli/bot"
)

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Run an AI bot player powered by Claude",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		name, _ := cmd.Flags().GetString("name")
		strategyFile, _ := cmd.Flags().GetString("strategy")

		strategy := bot.DefaultStrategy
		if strategyFile != "" {
			data, err := os.ReadFile(strategyFile)
			if err != nil {
				return fmt.Errorf("read strategy: %w", err)
			}
			strategy = string(data)
		}

		b, err := bot.New(addr, name, strategy)
		if err != nil {
			return err
		}
		return b.Run()
	},
}
