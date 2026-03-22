# Bot MCP Migration Design

## Overview

Migrate the `yatz bot` command from per-action `claude -p` invocations to a single `claude -p` session with MCP server integration. This resolves instability caused by stateless one-shot calls by maintaining full conversation context throughout the game.

## Problem

The current bot uses `exec.Command("claude", "-p", ...)` for every game action (roll/hold/score), resulting in:

- **No conversation context** between actions — Claude re-evaluates from scratch each time
- **Weak JSON schema constraints** — action/category/indices not enum-constrained, leading to invalid responses
- **Excessive process spawning** — up to 9 claude processes per turn (3 actions × 3 retries)
- **Fragile retry logic** — error context lost on retry attempts 1 and 2
- **No timeout** — `cmd.Output()` blocks indefinitely

## Solution

Replace per-action `claude -p` calls with a single `claude -p --mcp-config <config>` invocation. Claude uses MCP tools in an agentic loop to play the entire game within one session.

### Key Benefits

| Aspect | Before | After |
|--------|--------|-------|
| Process spawns | Up to 9 per turn | 1 per game |
| Conversation context | None | Full game history |
| Response format | Free-form JSON + schema | Structured MCP tool calls |
| Strategy consistency | None | Maintained across turns |
| Error handling | Manual retry in Go | Claude self-corrects |

## Architecture

```
yatz bot --addr localhost:9876
  │
  ├─ Generate temp MCP config JSON
  │    { "mcpServers": { "yatzcli": { "command": "<self>", "args": ["mcp"] } } }
  │
  ├─ Build prompt (strategy + game instructions)
  │
  └─ exec.Command("claude", "-p", "--mcp-config", config, "--allowedTools", "mcp__yatzcli__*", prompt)
       │
       └─ Claude agentic loop:
            join_game → [wait_for_turn if 2nd player] → [roll_dice → hold_dice → score] × 13 rounds
```

## Important: Score and Turn Waiting

`RemoteClient.Score()` internally blocks until the opponent's turn completes and returns the state for the bot's next turn. This means:

- **After `score` tool**: The returned state is already for the bot's next turn. Do NOT call `wait_for_turn` after scoring — it would deadlock.
- **`wait_for_turn` is only for**: The game start when the bot is the second player (opponent plays first).
- **The prompt must clearly instruct Claude** on this behavior.

## Changes

### 1. MCP Server: Add `wait_for_turn` tool

**File:** `mcp/server.go`

New tool that blocks until it's the player's turn or the game ends. Uses `RemoteClient.WaitForTurn()`. Only available during online games (RemoteClient). Type-asserts `gs.client` to `*p2p.RemoteClient` (same pattern as `handleSendChat`).

Returns: GameState when it's the player's turn, or game-over notification with final scores. If the opponent disconnects, the listener sends `nil` to `turnCh`, which unblocks `WaitForTurn()` and returns the listener error — no explicit timeout needed.

### 2. Bot Package: Rewrite to MCP approach

**File:** `bot/bot.go`

Remove:
- `callClaude()` — per-action exec.Command
- `callClaudeWithRetry()` — retry loop
- `playTurn()` — action dispatch switch
- `p2p.RemoteClient` dependency — MCP server handles connection
- `client` field from `Bot` struct

New `Bot` struct fields: `addr`, `name`, `strategy` (strings only, no network client).

New `Bot.Run()`:
1. Resolve `claude` binary via `exec.LookPath("claude")`
2. Get self binary path via `os.Executable()`
3. Write temp MCP config JSON to a temp file
4. Build prompt with strategy + game instructions
5. Execute single `claude -p --mcp-config <config> --allowedTools "mcp__yatzcli__*" <prompt>`
6. Set `cmd.Stdout = os.Stdout`, `cmd.Stderr = os.Stderr`, then `cmd.Run()` (not `cmd.Output()`) for streaming
7. Cleanup temp file via `defer os.Remove(configPath)`

**File:** `bot/prompt.go`

Remove:
- `ClaudeResponse` struct
- `responseSchema` / `ResponseSchemaJSON()`
- `ParseResponse()`
- `BuildRetryPrompt()`
- `BuildUserPrompt()` (replaced by `BuildPrompt`)

Keep:
- `BuildSystemPrompt()` — renamed, contains strategy + game procedure

Add:
- `BuildMCPConfig(yatzBinaryPath string) string` — generate MCP config JSON
- `BuildPrompt(addr, name, strategy string) string` — generate the single prompt

**Prompt content:**
```
ヤッツィーの対戦ゲームに参加してプレイしてください。

手順:
1. join_game で {addr} に接続（名前: {name}）
2. join_game の結果を確認し、自分のターンでなければ wait_for_turn で待機
3. 自分のターンでは: roll_dice → 必要に応じて hold_dice → score
4. score の結果に次のターンの状態が含まれる。wait_for_turn は呼ばないこと。
5. score の結果で Phase が "finished" なら終了
6. 3-5 を繰り返す

重要: score した後は wait_for_turn を呼んではいけない。score の返り値が既に次のターンの状態。

戦略: {strategy}

scoreした後はsend_chatで短い実況コメントを送ること。
```

### 3. Bot Command: Minimal changes

**File:** `cmd/yatz/bot.go`

Same flags (`--addr`, `--name`, `--strategy`). The `RunE` function calls simplified `bot.New(addr, name, strategy)` (no network connection) and `bot.Run()`.

### 4. Tests

**File:** `bot/prompt_test.go`

- Remove: JSON schema tests, ParseResponse tests
- Update: prompt building tests for new format
- Add: MCP config generation test, prompt content test

## Error Handling

- `claude` CLI not found → `exec.LookPath` error with message: "claude CLI is required. Install it first."
- `claude` process exits with error → display exit code (stderr already streamed)
- Temp file cleanup → `defer os.Remove(configPath)`
- MCP server process lifecycle: `claude` manages child process via stdio transport; when stdin closes (claude exits), MCP server exits. No explicit cleanup needed.
- Opponent disconnection during `wait_for_turn`: `RemoteClient.WaitForTurn()` returns error from listener, MCP tool returns error text, Claude can handle gracefully.

## Strategy

`bot/strategy.go` remains unchanged. The default strategy text is embedded in the system prompt passed to Claude.
