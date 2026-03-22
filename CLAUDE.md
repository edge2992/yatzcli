# YatzCLI

Yahtzee CLI game in Go. Single binary with local AI play, MCP server for LLM integration, and P2P online play.

## Build & Test

```bash
make check             # Quick validation (vet + unit tests + build)
make test-short        # Unit tests only (skips E2E)
make test-e2e          # E2E tests only
make test-all          # All tests with verbose output
make test-coverage     # Unit tests with coverage report
make build             # Build binary
make lint              # Static analysis (go vet)

# AI Battle
yatz battle                                    # Greedy vs Statistical (default)
yatz battle --players "A:llm:personas/aggressive.md,D:llm:personas/defensive.md"
yatz battle --rounds 100 --quiet               # 100-game statistics
```

## Testing Workflow

1. **開発中の変更確認**: `make check` — vet、ユニットテスト、ビルドを一括実行
2. **E2Eテスト**: `make test-e2e` — P2PやMCPの統合テスト。ネットワーク系の変更時に実行
3. **全テスト**: `make test-all` — CIと同等の全テスト実行。PR作成前に推奨

### テスト規約
- `-short` フラグ: `testing.Short()` でガードされたテストはユニットテスト時にスキップ
- `TestE2E` プレフィクス: E2Eテストは `TestE2E_` で始める。`-run TestE2E` で選択実行

## Project Structure

```
cmd/yatz/   Entry point (cobra subcommands: play, mcp, host, join, match, battle)
engine/     Pure game logic (state machine, scoring, dice, AI, Strategy, Battle, GameClient interface)
cli/        Interactive TUI (bubbletea v2) + AI battle spectator
mcp/        MCP server for LLM integration (mcp-go, stdio transport)
p2p/        P2P host-authority online play (length-prefixed JSON over TCP)
match/      Matchmaking WebSocket client
lambda/     Serverless matchmaking handler (AWS Lambda + API Gateway + DynamoDB)
bot/        LLM bot integration (MCP config, system prompt, Claude API, LLM Strategy)
personas/   Markdown-based AI persona definitions for LLM Strategy
```

## Key Design Decisions

- **GameClient interface** (`engine/client.go`): abstracts local vs remote game access. `LocalClient` wraps `Game` for local play; `RemoteClient` (`p2p/guest.go`) communicates over TCP.
- **State machine**: 4 phases — `PhaseWaiting`, `PhaseRolling`, `PhaseChoosing`, `PhaseFinished`. Transitions enforced in `engine/game.go`.
- **Player IDs**: Use hyphens (`player-0`, `player-1`), not underscores.
- **Host-authority model**: Host runs the game engine; guest sends actions over TCP and receives state updates.
- **AI auto-play**: `LocalClient.Score()` triggers AI turns automatically via `runAITurns()`.
- **Scorecard**: `map[Category]*int` where `nil` = unfilled, `*0` = filled with zero.
- **Strategy pattern** (`engine/strategy.go`): `Strategy` interface abstracts AI decision-making. Implementations: `GreedyStrategy` (immediate best score), `StatisticalStrategy` (expected value), `LLMStrategy` (Claude API via `anthropic-sdk-go`).
- **Battle engine** (`engine/battle.go`): `RunBattle()` drives AI-vs-AI games. `OnTurnDone` callback streams results to TUI spectator.
- **LLM API Key**: `LLMStrategy` calls Claude API directly (not via MCP) for speed. Uses `--api-key` flag or `ANTHROPIC_API_KEY` env var.

## Dependencies

- `cobra` — CLI framework
- `charm.land/bubbletea/v2` — TUI framework
- `mark3labs/mcp-go` — MCP server
- `gorilla/websocket` — matchmaking client
- `aws-lambda-go`, `aws-sdk-go-v2` — serverless matchmaking
- `anthropic-sdk-go` — Claude API client for LLM Strategy
- `stretchr/testify` — test assertions
