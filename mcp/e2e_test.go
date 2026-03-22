package mcp

import (
	"strings"
	"testing"
)

// TestE2E_FullGameThroughMCP plays a complete 13-round Yahtzee game
// through the MCP server tools: new_game → (roll_dice → score) × 13.
// Verifies game progresses through all rounds and ends with "Game Over".
func TestE2E_FullGameThroughMCP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	c := setupClient(t)

	// Start a new game with 1 AI opponent
	result := callTool(t, c, "new_game", map[string]interface{}{
		"opponents": 1.0,
	})
	text := getText(t, result)
	if result.IsError {
		t.Fatalf("new_game error: %s", text)
	}
	if !contains(text, "Round: 1/13") {
		t.Fatalf("expected Round: 1/13, got:\n%s", text)
	}

	for round := 1; round <= 13; round++ {
		// Roll dice
		result = callTool(t, c, "roll_dice", nil)
		text = getText(t, result)
		if result.IsError {
			t.Fatalf("round %d: roll_dice error: %s", round, text)
		}
		if !contains(text, "Dice:") {
			t.Fatalf("round %d: expected dice display, got:\n%s", round, text)
		}

		// Get state to find available categories
		stateResult := callTool(t, c, "get_state", nil)
		stateText := getText(t, stateResult)
		if stateResult.IsError {
			t.Fatalf("round %d: get_state error: %s", round, stateText)
		}

		// Extract available categories
		category := extractFirstCategory(t, stateText)

		// Score with the first available category
		result = callTool(t, c, "score", map[string]interface{}{
			"category": category,
		})
		text = getText(t, result)
		if result.IsError {
			t.Fatalf("round %d: score(%s) error: %s", round, category, text)
		}
		if !contains(text, "points") {
			t.Fatalf("round %d: expected 'points' in score output, got:\n%s", round, text)
		}

		if round == 13 {
			if !contains(text, "Game Over") {
				t.Fatalf("expected 'Game Over' after round 13, got:\n%s", text)
			}
			if !contains(text, "Final Scores") {
				t.Fatalf("expected 'Final Scores' after game over, got:\n%s", text)
			}
		}
	}

	// Verify scorecard shows all categories filled
	scResult := callTool(t, c, "get_scorecard", map[string]interface{}{
		"player_id": "player-0",
	})
	scText := getText(t, scResult)
	if scResult.IsError {
		t.Fatalf("get_scorecard error: %s", scText)
	}
	// No dash entries should remain (all filled)
	lines := strings.Split(scText, "\n")
	for _, line := range lines {
		// Skip header/separator/total lines
		if strings.Contains(line, "---") || strings.Contains(line, "===") ||
			strings.Contains(line, "Total") || strings.Contains(line, "Category") ||
			strings.Contains(line, "Bonus") || strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasSuffix(strings.TrimSpace(line), "-") {
			t.Errorf("expected all categories filled, but found unfilled: %s", line)
		}
	}
}

// TestE2E_FullGameWithHoldThroughMCP plays a complete game using
// roll → hold → score pattern to verify the hold mechanic works across rounds.
func TestE2E_FullGameWithHoldThroughMCP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	c := setupClient(t)

	result := callTool(t, c, "new_game", map[string]interface{}{
		"opponents": 1.0,
	})
	text := getText(t, result)
	if result.IsError {
		t.Fatalf("new_game error: %s", text)
	}

	for round := 1; round <= 13; round++ {
		// Roll dice
		result = callTool(t, c, "roll_dice", nil)
		text = getText(t, result)
		if result.IsError {
			t.Fatalf("round %d: roll_dice error: %s", round, text)
		}

		// Hold first two dice and reroll the rest
		result = callTool(t, c, "hold_dice", map[string]interface{}{
			"indices": []interface{}{0.0, 1.0},
		})
		text = getText(t, result)
		if result.IsError {
			t.Fatalf("round %d: hold_dice error: %s", round, text)
		}
		if !contains(text, "Roll Count: 2/3") {
			t.Fatalf("round %d: expected Roll Count: 2/3, got:\n%s", round, text)
		}

		// Get available categories and score
		stateResult := callTool(t, c, "get_state", nil)
		stateText := getText(t, stateResult)
		category := extractFirstCategory(t, stateText)

		result = callTool(t, c, "score", map[string]interface{}{
			"category": category,
		})
		text = getText(t, result)
		if result.IsError {
			t.Fatalf("round %d: score(%s) error: %s", round, category, text)
		}
	}

	// Verify game ended
	stateResult := callTool(t, c, "get_state", nil)
	stateText := getText(t, stateResult)
	if !contains(stateText, "Finished") {
		t.Fatalf("expected Finished phase after 13 rounds, got:\n%s", stateText)
	}
}

// extractFirstCategory parses get_state output to find the first available category.
func extractFirstCategory(t *testing.T, stateText string) string {
	t.Helper()
	const prefix = "Available Categories: "
	idx := strings.Index(stateText, prefix)
	if idx == -1 {
		t.Fatalf("no available categories found in:\n%s", stateText)
	}
	rest := stateText[idx+len(prefix):]
	// Categories are comma-separated on one line
	lineEnd := strings.Index(rest, "\n")
	if lineEnd != -1 {
		rest = rest[:lineEnd]
	}
	parts := strings.SplitN(rest, ",", 2)
	category := strings.TrimSpace(parts[0])
	if category == "" {
		t.Fatalf("empty category from:\n%s", stateText)
	}
	return category
}
