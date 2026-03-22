package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func setupClient(t *testing.T) *client.Client {
	t.Helper()
	s := newServer()
	c, err := client.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("failed to create in-process client: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	ctx := context.Background()
	_, err = c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			ClientInfo: mcp.Implementation{
				Name:    "test-client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}
	return c
}

func callTool(t *testing.T, c *client.Client, name string, args map[string]interface{}) *mcp.CallToolResult {
	t.Helper()
	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(%s) error: %v", name, err)
	}
	return result
}

func getText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("empty result content")
	}
	tc, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("result content is not text")
	}
	return tc.Text
}

func TestNewGame(t *testing.T) {
	c := setupClient(t)

	result := callTool(t, c, "new_game", map[string]interface{}{
		"opponents": 2.0,
	})
	text := getText(t, result)

	if result.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	for _, want := range []string{"New game started", "2 AI opponent", "Round: 1/13", "Rolling"} {
		if !contains(text, want) {
			t.Errorf("expected %q in output, got:\n%s", want, text)
		}
	}
}

func TestNewGameDefault(t *testing.T) {
	c := setupClient(t)

	result := callTool(t, c, "new_game", nil)
	text := getText(t, result)

	if result.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if !contains(text, "1 AI opponent") {
		t.Errorf("expected default 1 opponent, got:\n%s", text)
	}
}

func TestRollDice(t *testing.T) {
	c := setupClient(t)

	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})

	result := callTool(t, c, "roll_dice", nil)
	text := getText(t, result)

	if result.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if !contains(text, "Dice:") {
		t.Errorf("expected dice display, got:\n%s", text)
	}
	if !contains(text, "Roll Count: 1/3") {
		t.Errorf("expected roll count 1/3, got:\n%s", text)
	}
}

func TestRollDiceBeforeNewGame(t *testing.T) {
	c := setupClient(t)

	result := callTool(t, c, "roll_dice", nil)
	text := getText(t, result)

	if !result.IsError {
		t.Error("expected error for roll before new_game")
	}
	if !contains(text, "No game in progress") {
		t.Errorf("expected 'No game in progress' error, got: %s", text)
	}
}

func TestHoldDice(t *testing.T) {
	c := setupClient(t)

	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})
	callTool(t, c, "roll_dice", nil)

	result := callTool(t, c, "hold_dice", map[string]interface{}{
		"indices": []interface{}{0.0, 2.0, 4.0},
	})
	text := getText(t, result)

	if result.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if !contains(text, "Held dice") {
		t.Errorf("expected hold confirmation, got:\n%s", text)
	}
	if !contains(text, "Roll Count: 2/3") {
		t.Errorf("expected roll count 2/3, got:\n%s", text)
	}
}

func TestScore(t *testing.T) {
	c := setupClient(t)

	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})
	callTool(t, c, "roll_dice", nil)

	result := callTool(t, c, "score", map[string]interface{}{
		"category": "chance",
	})
	text := getText(t, result)

	if result.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if !contains(text, "chance") {
		t.Errorf("expected category name in output, got:\n%s", text)
	}
	if !contains(text, "points") {
		t.Errorf("expected points in output, got:\n%s", text)
	}
}

func TestScoreBeforeRoll(t *testing.T) {
	c := setupClient(t)

	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})

	result := callTool(t, c, "score", map[string]interface{}{
		"category": "chance",
	})
	text := getText(t, result)

	if !result.IsError {
		t.Error("expected error for score before roll")
	}
	if !contains(text, "must roll first") {
		t.Errorf("expected 'must roll first' error, got: %s", text)
	}
}

func TestGetState(t *testing.T) {
	c := setupClient(t)

	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})

	result := callTool(t, c, "get_state", nil)
	text := getText(t, result)

	if result.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	for _, want := range []string{"Round:", "Current Player:", "Phase:", "Roll Count:"} {
		if !contains(text, want) {
			t.Errorf("expected %q in output, got:\n%s", want, text)
		}
	}
}

func TestGetScorecard(t *testing.T) {
	c := setupClient(t)

	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})

	result := callTool(t, c, "get_scorecard", nil)
	text := getText(t, result)

	if result.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if !contains(text, "You") {
		t.Errorf("expected player name 'You', got:\n%s", text)
	}
	if !contains(text, "AI-1") {
		t.Errorf("expected AI player name, got:\n%s", text)
	}
	if !contains(text, "Total") {
		t.Errorf("expected Total row, got:\n%s", text)
	}
}

func TestGetScorecardByPlayerID(t *testing.T) {
	c := setupClient(t)

	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})

	result := callTool(t, c, "get_scorecard", map[string]interface{}{
		"player_id": "player-0",
	})
	text := getText(t, result)

	if result.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if !contains(text, "You") {
		t.Errorf("expected player name 'You', got:\n%s", text)
	}
	if contains(text, "AI-1") {
		t.Errorf("should not contain AI-1 when filtering by player_id, got:\n%s", text)
	}
}

func TestGetScorecardInvalidPlayer(t *testing.T) {
	c := setupClient(t)

	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})

	result := callTool(t, c, "get_scorecard", map[string]interface{}{
		"player_id": "nonexistent",
	})
	text := getText(t, result)

	if !result.IsError {
		t.Error("expected error for invalid player_id")
	}
	if !contains(text, "not found") {
		t.Errorf("expected 'not found' error, got: %s", text)
	}
}

func TestSendChatBeforeJoin(t *testing.T) {
	c := setupClient(t)

	// send_chat without join_game should error
	result := callTool(t, c, "send_chat", map[string]interface{}{
		"text": "hello",
	})
	text := getText(t, result)

	if !result.IsError {
		t.Error("expected error for send_chat before join_game")
	}
	if !contains(text, "Not connected to a game server") {
		t.Errorf("expected connection error, got: %s", text)
	}
}

func TestSendChatAfterNewGame(t *testing.T) {
	c := setupClient(t)

	// Start a local game, then try send_chat — should error because it's not an online game
	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1.0})

	result := callTool(t, c, "send_chat", map[string]interface{}{
		"text": "hello",
	})
	text := getText(t, result)

	if !result.IsError {
		t.Error("expected error for send_chat in local game")
	}
	if !contains(text, "Not connected to a game server") {
		t.Errorf("expected connection error, got: %s", text)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestWaitForTurnRequiresOnlineGame(t *testing.T) {
	c := setupClient(t)
	// No game started — should error
	result := callTool(t, c, "wait_for_turn", nil)
	text := getText(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, text, "Not connected")
}

func TestWaitForTurnRejectsLocalGame(t *testing.T) {
	c := setupClient(t)
	// Start a local game
	callTool(t, c, "new_game", map[string]interface{}{"opponents": 1})
	// wait_for_turn should reject local games
	result := callTool(t, c, "wait_for_turn", nil)
	text := getText(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, text, "Not connected")
}
