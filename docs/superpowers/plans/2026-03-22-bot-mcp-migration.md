# Bot MCP Migration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace per-action `claude -p` invocations with a single `claude -p` session using MCP tools for stable, context-aware Yahtzee gameplay.

**Architecture:** `yatz bot` generates a temp MCP config pointing to `yatz mcp`, then runs `claude -p --mcp-config <config>` once. Claude plays the entire game via MCP tool calls (join_game, roll_dice, hold_dice, score, wait_for_turn, send_chat) in a single agentic session.

**Tech Stack:** Go, cobra, mcp-go, exec.Command

**Spec:** `docs/superpowers/specs/2026-03-22-bot-mcp-migration-design.md`

---

### Task 1: Add `wait_for_turn` MCP tool

**Files:**
- Modify: `mcp/server.go:26-87` (tool registration in `newServer`)
- Modify: `mcp/server.go` (add handler method)
- Test: `mcp/server_test.go`

**Context:** The `wait_for_turn` tool type-asserts `gs.client` to `*p2p.RemoteClient`, same pattern as `handleSendChat` at `mcp/server.go:246-261`. `RemoteClient.WaitForTurn()` returns `(*GameState, bool, error)` where `bool` is `isGameOver` (see `p2p/guest.go:280-293`).

- [ ] **Step 1: Write the failing test**

Add to `mcp/server_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mcp/ -run TestWaitForTurn -v`
Expected: FAIL — tool "wait_for_turn" not found or panic

- [ ] **Step 3: Register tool and implement handler**

In `mcp/server.go`, add tool registration in `newServer()` after `sendChatTool` (after line 84):

```go
waitForTurnTool := mcp.NewTool("wait_for_turn",
	mcp.WithDescription("Wait for your turn during an online game. Blocks until it's your turn or game ends. Only use at game start if you're the second player. Do NOT use after scoring — score already waits for opponent."),
)
s.AddTool(waitForTurnTool, gs.handleWaitForTurn)
```

Add handler method after `handleSendChat`:

```go
func (gs *gameServer) handleWaitForTurn(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rc, ok := gs.client.(*p2p.RemoteClient)
	if !ok {
		return mcp.NewToolResultError("Not connected to a game server. Use join_game first."), nil
	}

	state, isGameOver, err := rc.WaitForTurn()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Connection error: %v", err)), nil
	}

	if isGameOver {
		var sb strings.Builder
		sb.WriteString("Game Over!\n\n")
		sb.WriteString(formatFinalScores(state))
		return mcp.NewToolResultText(sb.String()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Your turn!\n\n%s", formatState(state))), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/ -run TestWaitForTurn -v`
Expected: PASS

- [ ] **Step 5: Run all MCP tests**

Run: `go test ./mcp/ -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add mcp/server.go mcp/server_test.go
git commit -m "feat(mcp): add wait_for_turn tool for online game turn blocking"
```

---

### Task 2: Rewrite `bot/prompt.go`

**Files:**
- Rewrite: `bot/prompt.go`
- Rewrite: `bot/prompt_test.go`

**Context:** Remove all JSON schema/parsing logic (`ClaudeResponse`, `responseSchema`, `ResponseSchemaJSON`, `ParseResponse`, `BuildUserPrompt`, `BuildRetryPrompt`). Replace with `BuildMCPConfig` and `BuildPrompt`. Keep `BuildSystemPrompt` but rename to just build the strategy prompt for the `--system-prompt` flag. `bot/strategy.go` is unchanged.

- [ ] **Step 1: Write the failing tests**

Rewrite `bot/prompt_test.go`:

```go
package bot

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMCPConfig(t *testing.T) {
	config := BuildMCPConfig("/usr/local/bin/yatz")

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(config), &parsed)
	require.NoError(t, err)

	servers := parsed["mcpServers"].(map[string]interface{})
	yatzcli := servers["yatzcli"].(map[string]interface{})
	assert.Equal(t, "/usr/local/bin/yatz", yatzcli["command"])

	args := yatzcli["args"].([]interface{})
	assert.Equal(t, []interface{}{"mcp"}, args)
}

func TestBuildPrompt(t *testing.T) {
	prompt := BuildPrompt("localhost:9876", "TestBot", "test strategy")

	assert.Contains(t, prompt, "localhost:9876")
	assert.Contains(t, prompt, "TestBot")
	assert.Contains(t, prompt, "test strategy")
	assert.Contains(t, prompt, "join_game")
	assert.Contains(t, prompt, "wait_for_turn")
	assert.Contains(t, prompt, "roll_dice")
	assert.Contains(t, prompt, "score")
	assert.Contains(t, prompt, "send_chat")
	// Critical: instruction not to call wait_for_turn after score
	assert.Contains(t, prompt, "score した後は wait_for_turn を呼んではいけない")
}

func TestBuildSystemPrompt(t *testing.T) {
	prompt := BuildSystemPrompt("my strategy")
	assert.Contains(t, prompt, "ヤッツィー")
	assert.Contains(t, prompt, "my strategy")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./bot/ -run "TestBuildMCPConfig|TestBuildPrompt|TestBuildSystemPrompt" -v`
Expected: FAIL — compilation error, `BuildMCPConfig` and `BuildPrompt` undefined

- [ ] **Step 3: Rewrite `bot/prompt.go`**

Replace entire contents of `bot/prompt.go`:

```go
package bot

import (
	"encoding/json"
	"fmt"
)

func BuildMCPConfig(yatzBinaryPath string) string {
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"yatzcli": map[string]interface{}{
				"command": yatzBinaryPath,
				"args":    []string{"mcp"},
			},
		},
	}
	b, _ := json.MarshalIndent(config, "", "  ")
	return string(b)
}

func BuildSystemPrompt(strategy string) string {
	return fmt.Sprintf(`あなたはヤッツィーの対戦プレイヤーです。戦略的にプレイしてください。

戦略:
%s`, strategy)
}

func BuildPrompt(addr, name, strategy string) string {
	return fmt.Sprintf(`ヤッツィーの対戦ゲームに参加してプレイしてください。

手順:
1. join_game で %s に接続（名前: %s）
2. join_game の結果を確認し、自分のターンでなければ wait_for_turn で待機
3. 自分のターンでは: roll_dice → 必要に応じて hold_dice → score
4. score の結果に次のターンの状態が含まれる。wait_for_turn は呼ばないこと。
5. score の結果で Phase が "Finished" なら終了
6. 3-5 を繰り返す

重要: score した後は wait_for_turn を呼んではいけない。score の返り値が既に次のターンの状態。

戦略:
%s

scoreした後はsend_chatで短い実況コメントを日本語で送ること。`, addr, name, strategy)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./bot/ -run "TestBuildMCPConfig|TestBuildPrompt|TestBuildSystemPrompt" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add bot/prompt.go bot/prompt_test.go
git commit -m "refactor(bot): replace JSON schema/parsing with MCP config and prompt builders"
```

---

### Task 3: Rewrite `bot/bot.go`

**Files:**
- Rewrite: `bot/bot.go`
- Modify: `cmd/yatz/bot.go`

**Context:** Remove all per-action logic (`callClaude`, `callClaudeWithRetry`, `playTurn`, `printFinalScore`, `maxRetries`). Remove `p2p` import. New `Bot` struct has only `addr`, `name`, `strategy` strings. `New()` returns `*Bot` (no error). `Run()` writes temp MCP config, builds prompt, exec's `claude -p` once with streaming.

- [ ] **Step 1: Rewrite `bot/bot.go`**

Replace entire contents:

```go
package bot

import (
	"fmt"
	"os"
	"os/exec"
)

type Bot struct {
	addr     string
	name     string
	strategy string
}

func New(addr, name, strategy string) *Bot {
	return &Bot{
		addr:     addr,
		name:     name,
		strategy: strategy,
	}
}

func (b *Bot) Run() error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI is required. Install it first: %w", err)
	}

	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	configFile, err := os.CreateTemp("", "yatz-mcp-*.json")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.WriteString(BuildMCPConfig(selfPath)); err != nil {
		configFile.Close()
		return fmt.Errorf("write MCP config: %w", err)
	}
	configFile.Close()

	prompt := BuildPrompt(b.addr, b.name, b.strategy)
	systemPrompt := BuildSystemPrompt(b.strategy)

	cmd := exec.Command(claudePath, "-p",
		"--mcp-config", configFile.Name(),
		"--allowedTools", "mcp__yatzcli__*",
		"--system-prompt", systemPrompt,
		prompt,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude exited with error: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Update `cmd/yatz/bot.go`**

`bot.New()` no longer returns an error. Edit only the changed lines (line 29-33 in the current file). Change:

```go
		b, err := bot.New(addr, name, strategy)
		if err != nil {
			return err
		}
		return b.Run()
```

To:

```go
		b := bot.New(addr, name, strategy)
		return b.Run()
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./cmd/yatz/`
Expected: BUILD SUCCESS

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add bot/bot.go cmd/yatz/bot.go
git commit -m "feat(bot): rewrite bot to use single claude -p session with MCP tools"
```

---

### Task 4: Verify and clean up

**Files:**
- Check: `bot/strategy.go` (should be unchanged)
- Check: all files compile and tests pass

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

- [ ] **Step 2: Run static analysis**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 3: Build the binary**

Run: `go build ./cmd/yatz/`
Expected: BUILD SUCCESS

- [ ] **Step 4: Verify bot --help**

Run: `./yatz bot --help`
Expected: Shows `--addr`, `--name`, `--strategy` flags

- [ ] **Step 5: Commit if any cleanup was needed**

Only if changes were made in this task.
