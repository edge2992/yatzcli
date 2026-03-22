package bot

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/edge2992/yatzcli/engine"
	"github.com/edge2992/yatzcli/p2p"
)

const maxRetries = 3

type Bot struct {
	client   *p2p.RemoteClient
	name     string
	strategy string
}

func New(addr, name, strategy string) (*Bot, error) {
	rc, err := p2p.NewRemoteClient(addr, name)
	if err != nil {
		return nil, fmt.Errorf("connect to server: %w", err)
	}
	return &Bot{
		client:   rc,
		name:     name,
		strategy: strategy,
	}, nil
}

func (b *Bot) Run() error {
	defer b.client.Close()

	state, err := b.client.GetState()
	if err != nil {
		return fmt.Errorf("get initial state: %w", err)
	}

	// If it's not our turn at the start, wait
	if state.CurrentPlayer != b.client.PlayerID() {
		fmt.Fprintf(os.Stderr, "Waiting for our turn...\n")
		s, isGameOver, err := b.client.WaitForTurn()
		if err != nil {
			return fmt.Errorf("wait for turn: %w", err)
		}
		if isGameOver {
			b.printFinalScore(s)
			return nil
		}
		state = s
	}

	for state.Phase != engine.PhaseFinished {
		state, err = b.playTurn(state)
		if err != nil {
			return err
		}
	}

	b.printFinalScore(state)
	return nil
}

func (b *Bot) playTurn(state *engine.GameState) (*engine.GameState, error) {
	round := state.Round
	var lastErr string

	for {
		resp, err := b.callClaudeWithRetry(state, lastErr)
		if err != nil {
			return nil, fmt.Errorf("claude call failed: %w", err)
		}

		switch resp.Action {
		case "roll":
			newState, err := b.client.Roll()
			if err != nil {
				lastErr = err.Error()
				fmt.Fprintf(os.Stderr, "[Round %d] Roll error: %s\n", round, lastErr)
				continue
			}
			fmt.Fprintf(os.Stdout, "[Round %d] Roll: %v\n", round, newState.Dice)
			state = newState
			lastErr = ""

		case "hold":
			newState, err := b.client.Hold(resp.Indices)
			if err != nil {
				lastErr = err.Error()
				fmt.Fprintf(os.Stderr, "[Round %d] Hold error: %s\n", round, lastErr)
				continue
			}
			fmt.Fprintf(os.Stdout, "[Round %d] Hold %v → %v\n", round, resp.Indices, newState.Dice)
			state = newState
			lastErr = ""

		case "score":
			cat := engine.Category(resp.Category)
			score := engine.CalcScore(cat, state.Dice)
			newState, err := b.client.Score(cat)
			if err != nil {
				lastErr = err.Error()
				fmt.Fprintf(os.Stderr, "[Round %d] Score error: %s\n", round, lastErr)
				continue
			}
			fmt.Fprintf(os.Stdout, "[Round %d] Score: %s (%d pts) — %q\n", round, cat, score, resp.Comment)

			// Send chat
			if resp.Comment != "" {
				_ = b.client.SendChat(b.client.PlayerID(), b.name, resp.Comment)
			}

			return newState, nil

		default:
			lastErr = fmt.Sprintf("unknown action %q", resp.Action)
			fmt.Fprintf(os.Stderr, "[Round %d] Unknown action: %s\n", round, resp.Action)
		}
	}
}

func (b *Bot) callClaudeWithRetry(state *engine.GameState, lastErr string) (*ClaudeResponse, error) {
	systemPrompt := BuildSystemPrompt(b.strategy)
	schemaJSON := ResponseSchemaJSON()

	for attempt := 0; attempt < maxRetries; attempt++ {
		var userPrompt string
		if lastErr != "" && attempt == 0 {
			userPrompt = BuildRetryPrompt(state, b.client.PlayerID(), lastErr)
		} else {
			userPrompt = BuildUserPrompt(state, b.client.PlayerID())
		}

		output, err := callClaude(systemPrompt, schemaJSON, userPrompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "claude command failed (attempt %d/%d): %v\n", attempt+1, maxRetries, err)
			continue
		}

		resp, err := ParseResponse(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse response failed (attempt %d/%d): %v\n", attempt+1, maxRetries, err)
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("claude failed after %d retries", maxRetries)
}

func callClaude(systemPrompt, schemaJSON, userPrompt string) ([]byte, error) {
	cmd := exec.Command("claude", "-p",
		"--output-format", "json",
		"--json-schema", schemaJSON,
		"--system-prompt", systemPrompt,
		userPrompt,
	)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("execute claude: %w", err)
	}
	return output, nil
}

func (b *Bot) printFinalScore(state *engine.GameState) {
	fmt.Fprintf(os.Stdout, "\n=== Game Over ===\n")
	for _, p := range state.Players {
		fmt.Fprintf(os.Stdout, "%s: %d pts\n", p.Name, p.Scorecard.Total())
	}
}
