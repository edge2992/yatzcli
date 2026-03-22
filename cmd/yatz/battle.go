package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/edge2992/yatzcli/bot"
	"github.com/edge2992/yatzcli/cli"
	"github.com/edge2992/yatzcli/engine"
)

var battleCmd = &cobra.Command{
	Use:   "battle",
	Short: "Watch AI vs AI battle",
	Long:  `Run an AI vs AI battle. Specify players with --players "Name:strategy" format.`,
	RunE:  runBattle,
}

func init() {
	battleCmd.Flags().StringSlice("players", []string{"Greedy:greedy", "Statistical:statistical"}, `Players in "Name:strategy" format (greedy, statistical, llm:persona.md)`)
	battleCmd.Flags().Duration("speed", time.Second, "Turn display speed")
	battleCmd.Flags().Int64("seed", 0, "Random seed (0=random)")
	battleCmd.Flags().String("api-key", "", "Claude API key (or ANTHROPIC_API_KEY env)")
	battleCmd.Flags().String("model", "claude-haiku-4-5-20251001", "Claude model for LLM strategy")
	battleCmd.Flags().Int("rounds", 1, "Number of consecutive games")
	battleCmd.Flags().Bool("quiet", false, "No TUI, show results only")
}

func parseBattlePlayers(playerSpecs []string, apiKey string, model string) ([]engine.BattlePlayer, error) {
	var players []engine.BattlePlayer
	for _, spec := range playerSpecs {
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid player spec %q: expected Name:strategy", spec)
		}
		name := parts[0]
		stratSpec := parts[1]

		strategy, err := resolveStrategy(stratSpec, apiKey, model)
		if err != nil {
			return nil, fmt.Errorf("player %s: %w", name, err)
		}

		players = append(players, engine.BattlePlayer{
			Name:     name,
			Strategy: strategy,
		})
	}
	return players, nil
}

func resolveStrategy(spec string, apiKey string, model string) (engine.Strategy, error) {
	switch {
	case spec == "greedy":
		return &engine.GreedyStrategy{}, nil
	case spec == "statistical":
		return &engine.StatisticalStrategy{}, nil
	case spec == "llm":
		return bot.NewLLMStrategy(apiKey, model, nil), nil
	case strings.HasPrefix(spec, "llm:"):
		personaPath := strings.TrimPrefix(spec, "llm:")
		persona, err := bot.LoadPersona(personaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load persona %s: %w", personaPath, err)
		}
		return bot.NewLLMStrategy(apiKey, model, persona), nil
	default:
		return nil, fmt.Errorf("unknown strategy %q (available: greedy, statistical, llm, llm:<persona.md>)", spec)
	}
}

func runBattle(cmd *cobra.Command, args []string) error {
	playerSpecs, _ := cmd.Flags().GetStringSlice("players")
	speed, _ := cmd.Flags().GetDuration("speed")
	seed, _ := cmd.Flags().GetInt64("seed")
	rounds, _ := cmd.Flags().GetInt("rounds")
	quiet, _ := cmd.Flags().GetBool("quiet")
	apiKey, _ := cmd.Flags().GetString("api-key")
	model, _ := cmd.Flags().GetString("model")

	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	players, err := parseBattlePlayers(playerSpecs, apiKey, model)
	if err != nil {
		return err
	}

	if quiet {
		return runQuietBattle(players, seed, rounds)
	}

	if rounds > 1 {
		return runQuietBattle(players, seed, rounds)
	}

	return runTUIBattle(players, seed, speed)
}

func runQuietBattle(players []engine.BattlePlayer, seed int64, rounds int) error {
	type stats struct {
		wins     int
		total    int
		maxScore int
	}
	playerStats := make(map[string]*stats)
	for _, p := range players {
		playerStats[p.Name] = &stats{}
	}

	for r := 0; r < rounds; r++ {
		gameSeed := seed
		if seed != 0 {
			gameSeed = seed + int64(r)
		}

		state, err := engine.RunBattle(engine.BattleConfig{
			Players: players,
			Seed:    gameSeed,
		})
		if err != nil {
			return fmt.Errorf("game %d failed: %w", r+1, err)
		}

		// Find winner
		bestScore := -1
		winner := ""
		for _, p := range state.Players {
			score := p.Scorecard.Total()
			st := playerStats[p.Name]
			st.total += score
			if score > st.maxScore {
				st.maxScore = score
			}
			if score > bestScore {
				bestScore = score
				winner = p.Name
			}
		}
		playerStats[winner].wins++
	}

	// Sort players by wins descending
	type row struct {
		name     string
		wins     int
		avgScore float64
		maxScore int
	}
	var rows []row
	for _, p := range players {
		st := playerStats[p.Name]
		rows = append(rows, row{
			name:     p.Name,
			wins:     st.wins,
			avgScore: float64(st.total) / float64(rounds),
			maxScore: st.maxScore,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].wins > rows[j].wins })

	fmt.Fprintf(os.Stdout, "\n=== Battle Results (%d games) ===\n", rounds)
	fmt.Fprintf(os.Stdout, "%-16s %6s %11s %11s\n", "Player", "Wins", "Avg Score", "Max Score")
	for _, r := range rows {
		fmt.Fprintf(os.Stdout, "%-16s %6d %11.1f %11d\n", r.name, r.wins, r.avgScore, r.maxScore)
	}

	return nil
}

func runTUIBattle(players []engine.BattlePlayer, seed int64, speed time.Duration) error {
	resultCh := make(chan engine.AITurnResult, 64)

	cfg := engine.BattleConfig{
		Players: players,
		Seed:    seed,
		OnTurnDone: func(result engine.AITurnResult) {
			resultCh <- result
		},
	}

	// Run the battle in a background goroutine.
	// errCh is buffered (cap=1) so the goroutine never blocks after closing resultCh.
	errCh := make(chan error, 1)
	go func() {
		_, err := engine.RunBattle(cfg)
		close(resultCh)
		errCh <- err
	}()

	return cli.RunSpectator(resultCh, errCh, players, speed)
}
