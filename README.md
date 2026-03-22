# YatzCLI

A Yahtzee CLI game for engineers. Play against AI locally, challenge friends via P2P, or let Claude Code play for you via MCP.

## Install

```bash
go install github.com/edge2992/yatzcli/cmd/yatz@latest
```

Or download a binary from [GitHub Releases](https://github.com/edge2992/yatzcli/releases).

## Quick Start

### Play against AI

```bash
yatz play
yatz play -o 2 -n "Alice"  # 2 AI opponents, custom name
```

### MCP (Claude Code integration)

Add to your Claude Code MCP config:

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

Then ask Claude Code to play Yahtzee with you.

### P2P Online Play

```bash
# Player 1 (host)
yatz host --port 9876 --name Alice

# Player 2 (guest, on another machine)
yatz join 192.168.1.10:9876 --name Bob
```

### Matchmaking

```bash
yatz match --server wss://your-api-gateway-url --name Alice
```

## Commands

| Command | Description |
|---------|-------------|
| `yatz play` | Play locally against AI |
| `yatz mcp` | Start MCP server for LLM integration |
| `yatz host` | Host a P2P game |
| `yatz join <addr>` | Join a P2P game |
| `yatz match` | Find opponent via matchmaking |

## Controls (TUI)

**Rolling:** `r` roll, `1-5` toggle hold, `s` score selection, `q` quit
**Choosing:** `j/k` navigate, `enter` select category, `esc` back

## Architecture

- `engine/` - Pure game logic (state machine, scoring, dice)
- `cli/` - Interactive TUI (bubbletea)
- `mcp/` - MCP server for LLM integration
- `p2p/` - P2P host-authority online play
- `match/` - Matchmaking client
- `lambda/` - Serverless matchmaking handler (AWS)

## Development

```bash
go test ./...          # Run all tests
go build ./cmd/yatz/   # Build binary
```
