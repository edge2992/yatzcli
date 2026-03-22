# YatzCLI Rebuild Design Spec

## Overview

Rebuild the YatzCLI Yahtzee game from scratch as a single binary (`yatz`) with three play modes: local AI play via MCP, interactive CLI, and P2P online play with serverless matchmaking.

**Target users:** Engineers with Claude Code or similar LLM tooling on their machines.

**Goals:**
- Simple installation: `go install` or download binary
- Play instantly: `yatz play` for local AI game, no server needed
- MCP-native: Claude Code can play via MCP tools
- Online play: P2P with lightweight serverless matchmaking

## Project Structure

```
yatzcli/
├── cmd/
│   └── yatz/
│       └── main.go           # Entry point (cobra subcommands)
├── engine/
│   ├── dice.go               # Dice rolling, holding
│   ├── scorecard.go          # Scorecard, category scoring
│   ├── game.go               # Game progression (turn management, win condition)
│   └── engine_test.go
├── mcp/
│   └── server.go             # MCP server (stdio transport)
├── p2p/
│   ├── host.go               # Host (server + player)
│   ├── guest.go              # Guest (client)
│   └── protocol.go           # P2P message definitions
├── match/
│   └── client.go             # Matchmaking API client
├── cli/
│   └── ui.go                 # Interactive TUI (human play)
├── lambda/
│   └── handler.go            # Matchmaking Lambda handler
├── go.mod
├── go.sum
└── CLAUDE.md
```

## 1. Game Engine (`engine/`)

Pure game logic as a state machine. No external dependencies.

### Data Structures

```go
type GamePhase int

const (
    PhaseWaiting  GamePhase = iota
    PhaseRolling
    PhaseChoosing
    PhaseFinished
)

type Game struct {
    Players   []Player
    Current   int        // Current player index
    Round     int        // 1-13
    Dice      [5]int     // Current dice values
    RollCount int        // 0-3 (rolls this turn)
    Phase     GamePhase
}

type Player struct {
    ID        string
    Name      string
    Scorecard Scorecard
}

type Scorecard struct {
    Scores map[Category]*int  // nil = unfilled, value = filled
}
```

### State Transitions

```
Waiting  → Rolling   (game start / turn start)
Rolling  → Rolling   (reroll, RollCount < 3)
Rolling  → Choosing  (RollCount == 3 or player chooses to score)
Choosing → Rolling   (next player's turn)
Choosing → Finished  (all 13 rounds complete)
```

### API

- `NewGame(players []string) *Game`
- `(g *Game) Roll() error` — Roll dice
- `(g *Game) Hold(indices []int) error` — Set dice to hold, then reroll unheld
- `(g *Game) Score(category Category) error` — Choose category, record score
- `(g *Game) GetScorecard(playerID string) Scorecard`
- `(g *Game) GetAvailableCategories() []Category`
- `(g *Game) GetState() GameState` — Read-only current state

All methods validate the current Phase and return errors on invalid operations.

**Hold + Roll semantics:** `Hold(indices)` sets which dice to keep and immediately rerolls the rest (combines hold and roll in one call). `Roll()` is used only for the initial roll of a turn (RollCount == 0). Subsequent rolls must go through `Hold()`.

**RNG:** The engine accepts an optional `rand.Source` for deterministic testing. If nil, it uses `rand.NewSource(time.Now().UnixNano())`.

### Scoring

- **Upper Section (Ones-Sixes):** Sum of matching die values. Bonus of 35 if upper total >= 63.
- **Lower Section:** Three of a Kind (sum), Four of a Kind (sum), Full House (25), Small Straight (30), Large Straight (40), Yahtzee (50), Chance (sum).
- **Zero scoring:** If a player chooses a category whose conditions are not met (e.g., Full House without a valid combination), 0 is recorded. This is not an error.
- **Yahtzee Bonus / Joker rules are not implemented in the initial release** (see Non-Goals).

## 2. GameClient Interface

Abstraction layer that decouples UI/MCP from local vs remote game access.

```go
type Category string  // e.g., "ones", "full_house", "yahtzee"

type GameClient interface {
    Roll() (*GameState, error)
    Hold(indices []int) (*GameState, error)
    Score(category Category) (*GameState, error)
    GetState() (*GameState, error)
}
```

`Category` is a string type. MCP and CLI layers use the same string values, so no conversion is needed at the interface boundary.

Implementations:
- `LocalClient` — Wraps `engine.Game` directly (for local play and host)
- `RemoteClient` — Communicates over TCP with P2P host

## 3. MCP Server (`mcp/`)

Exposes game as MCP tools via stdio transport. Started with `yatz mcp`.

### Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `new_game` | Start a new game | `opponents: int` (1-3, default 1) |
| `roll_dice` | Roll the dice | none |
| `hold_dice` | Hold dice and reroll others | `indices: []int` (0-4) |
| `score` | Choose scoring category | `category: string` |
| `get_state` | Get current game state | none |
| `get_scorecard` | Get scorecard | `player_id?: string` |

### AI Opponent

- Simple strategy: auto-plays during opponent turns (e.g., greedy highest-score category)
- AI turns resolve automatically when human calls `roll_dice` or `score`, so LLM is never blocked waiting

### Configuration

```json
{
  "mcpServers": {
    "yatzcli": {
      "command": "yatz",
      "args": ["mcp"]
    }
  }
}
```

## 4. P2P Online Play (`p2p/`)

Host-authority model over TCP with JSON protocol.

### Connection Flow

```
Host: yatz host --port 9876
  → TCP listen, wait for connection

Guest: yatz join 192.168.1.10:9876
  → TCP connect to host
  → Handshake (name exchange)
  → Host starts game
```

### Protocol

JSON over TCP. Messages: `{type: string, payload: object}`.

| Type | Direction | Purpose |
|------|-----------|---------|
| `handshake` | Bidirectional | Name exchange |
| `game_start` | Host→Guest | Game started, initial state |
| `turn_start` | Host→Guest | Turn started notification |
| `action` | Guest→Host | Roll / Hold / Score |
| `state_update` | Host→Guest | Action result, latest state |
| `game_over` | Host→Guest | Game ended, final scores |
| `error` | Host→Guest | Error notification |

### Host Behavior

- Holds the `engine.Game` instance (authoritative state)
- Plays as a participant and serves game state to guest
- Host uses `LocalClient`, guest uses `RemoteClient`

**Host's own turn:** Host operates the engine directly via `LocalClient`. After each action (Roll/Hold/Score), host sends `state_update` to guest. Guest receives `turn_start` at the beginning of every turn (both host's and guest's) so it can display the current game state.

**Disconnection:** If TCP connection drops mid-game, the game ends immediately. No reconnection or state recovery in initial release.

**Invalid actions:** Host validates all guest actions via the engine. Invalid actions return an `error` message; game state is unchanged.

## 5. Matchmaking (`match/` + `lambda/`)

Serverless matchmaking to find opponents and establish P2P connections.

### Architecture

```
Client → API Gateway (WebSocket) → Lambda → DynamoDB
```

### Flow

1. `yatz match` connects to matchmaking API via WebSocket
2. Registers in DynamoDB waiting table with endpoint info
3. When opponent found, both receive each other's connection info
4. Earlier player becomes host
5. WebSocket disconnects, P2P connection begins

### DynamoDB Table

```
WaitingPlayers:
  - PlayerID (PK): uuid
  - Name: string
  - Endpoint: string (ip:port)
  - CreatedAt: timestamp
  - TTL: timestamp (5 min auto-delete)
```

### Endpoint Detection

When a client connects to the matchmaking WebSocket, the Lambda function detects the client's public IP from the API Gateway request context (`requestContext.identity.sourceIp`). The client provides its listening port via the initial message. Alternatively, users can specify `--endpoint ip:port` manually.

### Cost

- DynamoDB on-demand: ~$0 at low usage
- Lambda: pay-per-request only
- API Gateway WebSocket: connection-minute based pricing ($0.25/million connection-minutes). Connections are short-lived (disconnect after match), so cost is negligible at low usage.

### NAT Traversal

- Initial implementation assumes port forwarding (acceptable for engineer audience)
- STUN/TURN or relay server can be added later if needed

## 6. CLI UI (`cli/`)

Interactive TUI for human play.

- Uses `GameClient` interface (works for both local and P2P)
- Built with `charmbracelet/bubbletea` or `charmbracelet/huh` for modern TUI
- Replaces archived `survey` library from current codebase

## 7. Subcommands

| Command | Description |
|---------|-------------|
| `yatz play` | Local AI game (interactive CLI) |
| `yatz mcp` | Start MCP server |
| `yatz host` | Host P2P game |
| `yatz join <addr>` | Join P2P game |
| `yatz match` | Find opponent via matchmaking |

## 8. Dependencies

| Library | Purpose |
|---------|---------|
| `spf13/cobra` | Subcommand management |
| `charmbracelet/bubbletea` or `charmbracelet/huh` | TUI |
| `charmbracelet/lipgloss` | Styling |
| `mark3labs/mcp-go` | Go MCP SDK |
| `aws/aws-lambda-go` | Lambda handler |

## 9. Distribution

- `go install github.com/edge2992/yatzcli/cmd/yatz@latest`
- GoReleaser + GitHub Actions for multi-platform binary releases (linux/mac/windows, amd64/arm64)
- Release triggered by git tags

## 10. Testing Strategy

- **Engine:** Comprehensive unit tests with deterministic RNG (seeded `rand.Source`). Cover all scoring categories, state transitions, and edge cases (invalid operations, boundary conditions).
- **GameClient:** Test `LocalClient` against engine directly. Test `RemoteClient` with a mock TCP server.
- **P2P:** Integration tests using loopback TCP connections.
- **MCP:** Test tool handlers with mock engine.
- **Matchmaking Lambda:** Unit tests with mock DynamoDB.

## Non-Goals

- Web UI or mobile client
- Advanced AI strategies (simple greedy is sufficient)
- Yahtzee Bonus / Joker rules in initial release
- NAT traversal (STUN/TURN) in initial release
- Spectator mode
- Reconnection / state recovery on disconnect
- More than 2 players in P2P (can be added later)
