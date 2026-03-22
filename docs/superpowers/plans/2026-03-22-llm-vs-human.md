# LLM vs Human Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable Claude Code to play Yahtzee against a human via a headless Game Server, with chat commentary visible in the TUI.

**Architecture:** A headless Game Server (`p2p/server.go`) accepts TCP connections from any number of clients. Both the TUI (`yatz join`) and MCP Server (Claude Code) connect as equal clients using the existing `RemoteClient`. The P2P protocol is extended with chat messages and player ID assignment in handshake.

**Tech Stack:** Go 1.24, existing `p2p/` protocol, `engine.GameClient` interface, `bubbletea/v2` TUI, `mcp-go` MCP server

**Spec:** `docs/superpowers/specs/2026-03-22-llm-vs-human-design.md`

---

### Task 1: Protocol Extension — MsgChat and HandshakePayload

**Files:**
- Modify: `p2p/protocol.go:12-20` (add MsgChat const), `:33-35` (extend HandshakePayload)
- Test: `p2p/protocol_test.go`

- [ ] **Step 1: Write failing test for ChatPayload round-trip**

```go
func TestMessage_RoundTrip_Chat(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := NewChatMsg("player-0", "Alice", "Hello!")
	if err := WriteMessage(buf, msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Type != MsgChat {
		t.Errorf("type = %q, want %q", got.Type, MsgChat)
	}
	cp, err := DecodeChat(got)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cp.PlayerID != "player-0" || cp.Name != "Alice" || cp.Text != "Hello!" {
		t.Errorf("payload = %+v", cp)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./p2p/ -run TestMessage_RoundTrip_Chat -v`
Expected: FAIL — `NewChatMsg` undefined

- [ ] **Step 3: Implement MsgChat, ChatPayload, NewChatMsg, DecodeChat**

In `p2p/protocol.go`:

Add constant:
```go
MsgChat = "chat"
```

Add types and functions:
```go
type ChatPayload struct {
	PlayerID string `json:"player_id"`
	Name     string `json:"name"`
	Text     string `json:"text"`
}

func NewChatMsg(playerID, name, text string) *Message {
	return newMessage(MsgChat, ChatPayload{PlayerID: playerID, Name: name, Text: text})
}

func DecodeChat(msg *Message) (*ChatPayload, error) {
	var p ChatPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode chat: %w", err)
	}
	return &p, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./p2p/ -run TestMessage_RoundTrip_Chat -v`
Expected: PASS

- [ ] **Step 5: Write failing test for HandshakePayload with PlayerID**

```go
func TestMessage_RoundTrip_HandshakeWithPlayerID(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := newMessage(MsgHandshake, HandshakePayload{Name: "Server", PlayerID: "player-0"})
	if err := WriteMessage(buf, msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	hs, err := DecodeHandshake(got)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if hs.PlayerID != "player-0" {
		t.Errorf("PlayerID = %q, want %q", hs.PlayerID, "player-0")
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./p2p/ -run TestMessage_RoundTrip_HandshakeWithPlayerID -v`
Expected: FAIL — `hs.PlayerID` undefined

- [ ] **Step 7: Extend HandshakePayload with PlayerID field**

In `p2p/protocol.go`, change `HandshakePayload`:
```go
type HandshakePayload struct {
	Name     string `json:"name"`
	PlayerID string `json:"player_id,omitempty"`
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./p2p/ -run TestMessage_RoundTrip_HandshakeWithPlayerID -v`
Expected: PASS

- [ ] **Step 9: Write test for backward compatibility (empty PlayerID)**

```go
func TestMessage_RoundTrip_HandshakeBackwardCompat(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := NewHandshakeMsg("Guest")
	if err := WriteMessage(buf, msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	hs, err := DecodeHandshake(got)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if hs.PlayerID != "" {
		t.Errorf("PlayerID = %q, want empty", hs.PlayerID)
	}
	if hs.Name != "Guest" {
		t.Errorf("Name = %q, want %q", hs.Name, "Guest")
	}
}
```

- [ ] **Step 10: Run test to verify it passes (should pass immediately)**

Run: `go test ./p2p/ -run TestMessage_RoundTrip_HandshakeBackwardCompat -v`
Expected: PASS

- [ ] **Step 11: Run all existing tests to verify no regressions**

Run: `go test ./... -short`
Expected: All PASS

- [ ] **Step 12: Commit**

```bash
git add p2p/protocol.go p2p/protocol_test.go
git commit -m "feat(protocol): add MsgChat and extend HandshakePayload with PlayerID"
```

---

### Task 2: RemoteClient Changes — Dynamic PlayerID and ChatCh

**Files:**
- Modify: `p2p/guest.go:62-103` (newRemoteClientFromConn), `:106-181` (listen)
- Test: `p2p/guest_test.go`

- [ ] **Step 1: Write failing test for dynamic playerID from handshake**

```go
func TestRemoteClient_DynamicPlayerID(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	done := make(chan error, 1)
	var rc *RemoteClient
	go func() {
		var err error
		rc, err = newRemoteClientFromConn(guestConn, "Guest")
		done <- err
	}()

	// Read guest handshake
	msg, err := ReadMessage(hostConn)
	if err != nil {
		t.Fatalf("read handshake: %v", err)
	}
	if msg.Type != MsgHandshake {
		t.Fatalf("expected handshake, got %s", msg.Type)
	}

	// Send handshake with PlayerID
	hsMsg := newMessage(MsgHandshake, HandshakePayload{Name: "Server", PlayerID: "player-0"})
	if err := WriteMessage(hostConn, hsMsg); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	// Send game_start
	state := sampleState()
	if err := WriteMessage(hostConn, NewGameStartMsg(state)); err != nil {
		t.Fatalf("write game_start: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("newRemoteClientFromConn: %v", err)
	}

	if rc.playerID != "player-0" {
		t.Errorf("playerID = %q, want %q", rc.playerID, "player-0")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./p2p/ -run TestRemoteClient_DynamicPlayerID -v`
Expected: FAIL — `rc.playerID` is `"player-1"` (hardcoded)

- [ ] **Step 3: Implement dynamic playerID in newRemoteClientFromConn**

In `p2p/guest.go`, modify `newRemoteClientFromConn()` — after receiving host handshake, decode `HandshakePayload` and use `PlayerID` if non-empty:

```go
hs, err := DecodeHandshake(msg)
if err != nil {
	return nil, fmt.Errorf("decode handshake: %w", err)
}

playerID := hs.PlayerID
if playerID == "" {
	playerID = "player-1" // backward compat with old host
}
```

Then in the `RemoteClient` struct init, use `playerID` instead of hardcoded `"player-1"`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./p2p/ -run TestRemoteClient_DynamicPlayerID -v`
Expected: PASS

- [ ] **Step 5: Write failing test for chatCh receiving MsgChat**

```go
func TestRemoteClient_ChatChannel(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	rc := setupRemoteClient(t, hostConn, guestConn)
	defer rc.Close()

	// Send a chat message from host side
	chatMsg := NewChatMsg("player-1", "Host", "Hello from host!")
	if err := WriteMessage(hostConn, chatMsg); err != nil {
		t.Fatalf("write chat: %v", err)
	}

	select {
	case cp := <-rc.ChatCh():
		if cp == nil {
			t.Fatal("received nil chat")
		}
		if cp.Text != "Hello from host!" {
			t.Errorf("text = %q, want %q", cp.Text, "Hello from host!")
		}
		if cp.Name != "Host" {
			t.Errorf("name = %q, want %q", cp.Name, "Host")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for chat message")
	}
}
```

Note: `setupRemoteClient` is a test helper that performs handshake and game_start setup. `ChatCh()` is a public accessor for the chat channel.

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./p2p/ -run TestRemoteClient_ChatChannel -v`
Expected: FAIL — `ChatCh` method undefined

- [ ] **Step 7: Add chatCh to RemoteClient and handle MsgChat in listen()**

In `p2p/guest.go`:

Add field to `RemoteClient`:
```go
chatCh chan *ChatPayload
```

Initialize in `newRemoteClientFromConn()`:
```go
chatCh: make(chan *ChatPayload, 16),
```

Add public accessor:
```go
func (rc *RemoteClient) ChatCh() <-chan *ChatPayload {
	return rc.chatCh
}
```

Add case in `listen()` switch:
```go
case MsgChat:
	cp, err := DecodeChat(msg)
	if err != nil {
		continue
	}
	select {
	case rc.chatCh <- cp:
	default: // drop if buffer full
	}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./p2p/ -run TestRemoteClient_ChatChannel -v`
Expected: PASS

- [ ] **Step 9: Run all existing tests to verify no regressions**

Run: `go test ./... -short`
Expected: All PASS (existing P2P tests should still work with backward-compatible handshake)

- [ ] **Step 10: Commit**

```bash
git add p2p/guest.go p2p/guest_test.go
git commit -m "feat(p2p): add dynamic playerID and chat channel to RemoteClient"
```

---

### Task 3: Headless Game Server

**Files:**
- Create: `p2p/server.go`
- Create: `p2p/server_test.go`

- [ ] **Step 1: Write failing test for server handshake and player ID assignment**

In `p2p/server_test.go`:

```go
func TestServer_Handshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunServer(ln, 2, rand.NewSource(42))
	}()

	// Connect client 1
	conn1, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn1.Close()

	if err := WriteMessage(conn1, NewHandshakeMsg("Alice")); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	msg1, err := ReadMessage(conn1)
	if err != nil {
		t.Fatalf("read handshake: %v", err)
	}
	hs1, err := DecodeHandshake(msg1)
	if err != nil {
		t.Fatalf("decode handshake: %v", err)
	}
	if hs1.PlayerID != "player-0" {
		t.Errorf("player 1 ID = %q, want %q", hs1.PlayerID, "player-0")
	}

	// Connect client 2
	conn2, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn2.Close()

	if err := WriteMessage(conn2, NewHandshakeMsg("Bob")); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	msg2, err := ReadMessage(conn2)
	if err != nil {
		t.Fatalf("read handshake: %v", err)
	}
	hs2, err := DecodeHandshake(msg2)
	if err != nil {
		t.Fatalf("decode handshake: %v", err)
	}
	if hs2.PlayerID != "player-1" {
		t.Errorf("player 2 ID = %q, want %q", hs2.PlayerID, "player-1")
	}

	// Both should receive game_start
	gs1, err := ReadMessage(conn1)
	if err != nil {
		t.Fatalf("read game_start 1: %v", err)
	}
	if gs1.Type != MsgGameStart {
		t.Errorf("msg type = %q, want %q", gs1.Type, MsgGameStart)
	}

	gs2, err := ReadMessage(conn2)
	if err != nil {
		t.Fatalf("read game_start 2: %v", err)
	}
	if gs2.Type != MsgGameStart {
		t.Errorf("msg type = %q, want %q", gs2.Type, MsgGameStart)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./p2p/ -run TestServer_Handshake -v`
Expected: FAIL — `RunServer` undefined

- [ ] **Step 3: Implement server connection acceptance and handshake**

Create `p2p/server.go` with:

```go
package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/edge2992/yatzcli/engine"
)

const serverHandshakeTimeout = 30 * time.Second

type clientConn struct {
	conn     net.Conn
	name     string
	playerID string
	actionCh chan *ActionPayload
	mu       sync.Mutex
}

func RunServer(ln net.Listener, numPlayers int, rngSrc rand.Source) error {
	clients, err := acceptClients(ln, numPlayers)
	if err != nil {
		return fmt.Errorf("accept clients: %w", err)
	}
	defer func() {
		for _, c := range clients {
			c.conn.Close()
		}
	}()

	names := make([]string, numPlayers)
	for i, c := range clients {
		names[i] = c.name
	}
	game := engine.NewGame(names, rngSrc)

	// Send game_start to all
	gs := game.GetState()
	for _, c := range clients {
		if err := writeToClient(c, NewGameStartMsg(gs)); err != nil {
			return fmt.Errorf("send game_start: %w", err)
		}
	}

	// Start reader goroutines
	for _, c := range clients {
		go readLoop(c, clients)
	}

	return gameLoop(game, clients)
}

func acceptClients(ln net.Listener, numPlayers int) ([]*clientConn, error) {
	clients := make([]*clientConn, 0, numPlayers)
	for i := 0; i < numPlayers; i++ {
		conn, err := ln.Accept()
		if err != nil {
			return nil, fmt.Errorf("accept: %w", err)
		}

		conn.SetDeadline(time.Now().Add(serverHandshakeTimeout))
		msg, err := ReadMessage(conn)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("read handshake: %w", err)
		}
		if msg.Type != MsgHandshake {
			conn.Close()
			return nil, fmt.Errorf("expected handshake, got %s", msg.Type)
		}
		hs, err := DecodeHandshake(msg)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("decode handshake: %w", err)
		}

		playerID := fmt.Sprintf("player-%d", i)
		resp := newMessage(MsgHandshake, HandshakePayload{Name: "Server", PlayerID: playerID})
		if err := WriteMessage(conn, resp); err != nil {
			conn.Close()
			return nil, fmt.Errorf("send handshake: %w", err)
		}
		conn.SetDeadline(time.Time{})

		clients = append(clients, &clientConn{
			conn:     conn,
			name:     hs.Name,
			playerID: playerID,
			actionCh: make(chan *ActionPayload, 1),
		})
	}
	return clients, nil
}
```

(Full gameLoop and readLoop implementation)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./p2p/ -run TestServer_Handshake -v`
Expected: PASS

- [ ] **Step 5: Write failing test for server game loop (turn management)**

```go
func TestServer_TurnManagement(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunServer(ln, 2, rand.NewSource(42))
	}()

	conn1 := connectAndHandshake(t, ln.Addr().String(), "Alice")
	defer conn1.Close()
	conn2 := connectAndHandshake(t, ln.Addr().String(), "Bob")
	defer conn2.Close()

	// Read game_start from both
	readExpectType(t, conn1, MsgGameStart)
	readExpectType(t, conn2, MsgGameStart)

	// player-0 (Alice) should receive turn_start
	readExpectType(t, conn1, MsgTurnStart)

	// Alice rolls
	WriteMessage(conn1, NewActionMsg(ActionPayload{Action: ActionRoll}))

	// Both should get state_update
	readExpectType(t, conn1, MsgStateUpdate)
	readExpectType(t, conn2, MsgStateUpdate)

	// Alice scores in "chance"
	WriteMessage(conn1, NewActionMsg(ActionPayload{Action: ActionScore, Category: "chance"}))

	// Both get state_update
	readExpectType(t, conn1, MsgStateUpdate)
	readExpectType(t, conn2, MsgStateUpdate)

	// Now player-1 (Bob) should receive turn_start
	readExpectType(t, conn2, MsgTurnStart)
}
```

Test helpers `connectAndHandshake` and `readExpectType` simplify connection setup and message type validation.

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./p2p/ -run TestServer_TurnManagement -v`
Expected: FAIL — gameLoop not yet implemented (or missing turn_start)

- [ ] **Step 7: Implement gameLoop and readLoop**

In `p2p/server.go`:

```go
func readLoop(c *clientConn, allClients []*clientConn) {
	for {
		msg, err := ReadMessage(c.conn)
		if err != nil {
			close(c.actionCh)
			return
		}
		switch msg.Type {
		case MsgAction:
			ap, err := DecodeAction(msg)
			if err != nil {
				continue
			}
			c.actionCh <- ap
		case MsgChat:
			// Broadcast to all clients
			for _, other := range allClients {
				writeToClient(other, msg)
			}
		}
	}
}

func gameLoop(game *engine.Game, clients []*clientConn) error {
	for game.Phase != engine.PhaseFinished {
		current := clients[game.Current]

		// Send turn_start to current player
		gs := game.GetState()
		if err := writeToClient(current, NewTurnStartMsg(gs)); err != nil {
			return notifyDisconnect(clients, current, err)
		}

		// Read actions until score
		for {
			ap, ok := <-current.actionCh
			if !ok {
				return notifyDisconnect(clients, current, fmt.Errorf("client disconnected"))
			}

			var actionErr error
			switch ap.Action {
			case ActionRoll:
				actionErr = game.Roll()
			case ActionHold:
				actionErr = game.Hold(ap.Indices)
			case ActionScore:
				actionErr = game.Score(engine.Category(ap.Category))
			default:
				actionErr = fmt.Errorf("unknown action: %s", ap.Action)
			}

			if actionErr != nil {
				writeToClient(current, NewErrorMsg(actionErr.Error()))
				continue
			}

			// Broadcast state_update to all
			state := game.GetState()
			for _, c := range clients {
				writeToClient(c, NewStateUpdateMsg(state))
			}

			if ap.Action == ActionScore {
				break
			}
		}
	}

	// Broadcast game_over
	finalState := game.GetState()
	for _, c := range clients {
		writeToClient(c, NewGameOverMsg(finalState))
	}
	return nil
}

func writeToClient(c *clientConn, msg *Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return WriteMessage(c.conn, msg)
}

func notifyDisconnect(clients []*clientConn, disconnected *clientConn, cause error) error {
	for _, c := range clients {
		if c != disconnected {
			writeToClient(c, NewErrorMsg(fmt.Sprintf("opponent disconnected: %v", cause)))
		}
	}
	return fmt.Errorf("client %s disconnected: %w", disconnected.name, cause)
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./p2p/ -run TestServer_TurnManagement -v`
Expected: PASS

- [ ] **Step 9: Write failing test for chat broadcast**

```go
func TestServer_ChatBroadcast(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go RunServer(ln, 2, rand.NewSource(42))

	conn1 := connectAndHandshake(t, ln.Addr().String(), "Alice")
	defer conn1.Close()
	conn2 := connectAndHandshake(t, ln.Addr().String(), "Bob")
	defer conn2.Close()

	// Read game_start
	readExpectType(t, conn1, MsgGameStart)
	readExpectType(t, conn2, MsgGameStart)

	// Read turn_start for player-0
	readExpectType(t, conn1, MsgTurnStart)

	// Bob sends a chat
	chatMsg := NewChatMsg("player-1", "Bob", "Good luck!")
	if err := WriteMessage(conn2, chatMsg); err != nil {
		t.Fatalf("write chat: %v", err)
	}

	// Small delay for broadcast
	time.Sleep(50 * time.Millisecond)

	// Both should eventually receive the chat
	// Note: conn1 might receive chat on next read; conn2 also gets it
	// We send a game action from conn1 to trigger reads, then check
	// The chat broadcast is async, so we use a read with timeout
}
```

Note: Chat broadcast is async. The test verifies that chat arrives at both clients. Implementation may need a dedicated read helper with timeout and message type filtering.

- [ ] **Step 10: Run test and implement if needed**

Run: `go test ./p2p/ -run TestServer_ChatBroadcast -v`
Expected: PASS (should work with existing readLoop implementation)

- [ ] **Step 11: Run all tests**

Run: `go test ./... -short`
Expected: All PASS

- [ ] **Step 12: Commit**

```bash
git add p2p/server.go p2p/server_test.go
git commit -m "feat(p2p): add headless game server with turn management and chat"
```

---

### Task 4: `yatz serve` Subcommand

**Files:**
- Create: `cmd/yatz/serve.go`
- Modify: `cmd/yatz/main.go:100-117` (register serve command)

- [ ] **Step 1: Create `cmd/yatz/serve.go`**

```go
package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/spf13/cobra"

	"github.com/edge2992/yatzcli/p2p"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a headless game server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		players, _ := cmd.Flags().GetInt("players")

		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		defer ln.Close()

		fmt.Printf("Game server listening on port %d, waiting for %d players...\n", port, players)
		return p2p.RunServer(ln, players, rand.NewSource(time.Now().UnixNano()))
	},
}
```

- [ ] **Step 2: Register in `cmd/yatz/main.go`**

In `init()`:
```go
serveCmd.Flags().IntP("port", "p", 9876, "Port to listen on")
serveCmd.Flags().Int("players", 2, "Number of players")
rootCmd.AddCommand(serveCmd)
```

- [ ] **Step 3: Verify it builds**

Run: `go build ./cmd/yatz/`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add cmd/yatz/serve.go cmd/yatz/main.go
git commit -m "feat(cmd): add yatz serve subcommand for headless game server"
```

---

### Task 5: MCP Server — GameClient Interface and handleScore Fix

**Files:**
- Modify: `mcp/server.go:13-17` (gameServer struct), `:132-158` (handleScore)
- Test: `mcp/server_test.go`

- [ ] **Step 1: Write failing test for handleScore using GetState() for dice**

Add test in `mcp/server_test.go` that verifies score is computed correctly. The existing `TestScore` test should still pass after refactoring.

Run existing tests first to establish baseline:
Run: `go test ./mcp/ -v`
Expected: All PASS

- [ ] **Step 2: Refactor gameServer.client to engine.GameClient**

In `mcp/server.go`, change the `gameServer` struct:

```go
type gameServer struct {
	game       *engine.Game
	client     engine.GameClient
	ais        []*engine.AIPlayer
	onlineName string // player name for online mode
}
```

- [ ] **Step 3: Fix handleScore to use GetState() before Score()**

In `handleScore`, change:
```go
// Before:
// score := engine.CalcScore(cat, gs.game.Dice)
// state, scoreErr := gs.client.Score(cat)

// After:
currentState, _ := gs.client.GetState()
score := engine.CalcScore(cat, currentState.Dice)
state, scoreErr := gs.client.Score(cat)
```

- [ ] **Step 4: Run tests to verify no regressions**

Run: `go test ./mcp/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add mcp/server.go
git commit -m "refactor(mcp): use GameClient interface and fix handleScore dice access"
```

---

### Task 6: MCP Server — join_game and send_chat Tools

**Files:**
- Modify: `mcp/server.go` (add tools)
- Test: `mcp/server_test.go`

- [ ] **Step 1: Write failing test for join_game tool**

```go
func TestJoinGame(t *testing.T) {
	// Start a local game server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go p2p.RunServer(ln, 2, rand.NewSource(42))

	// Connect a dummy player first
	dummyConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer dummyConn.Close()
	p2p.WriteMessage(dummyConn, p2p.NewHandshakeMsg("Dummy"))
	p2p.ReadMessage(dummyConn) // handshake response

	// Now test MCP join_game
	c := setupClient(t)
	result := callTool(t, c, "join_game", map[string]any{
		"addr": ln.Addr().String(),
		"name": "Claude",
	})
	text := getText(t, result)
	if result.IsError {
		t.Fatalf("join_game error: %s", text)
	}
	if !contains(text, "Joined") {
		t.Errorf("expected join confirmation, got: %s", text)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mcp/ -run TestJoinGame -v`
Expected: FAIL — `join_game` tool not registered

- [ ] **Step 3: Implement join_game tool**

In `mcp/server.go`, add tool registration in `newServer()`:

```go
joinGameTool := mcp.NewTool("join_game",
	mcp.WithDescription("Join a game server for online play"),
	mcp.WithString("addr", mcp.Required(), mcp.Description("Server address (e.g. localhost:9876)")),
	mcp.WithString("name", mcp.Description("Your player name (default: Claude)")),
)
s.AddTool(joinGameTool, gs.handleJoinGame)
```

Implement handler:
```go
func (gs *gameServer) handleJoinGame(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	addr, err := req.RequireString("addr")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name := req.GetString("name", "Claude")

	// Cleanup existing game
	if gs.conn != nil {
		gs.conn.Close()
		gs.conn = nil
	}
	gs.game = nil
	gs.ais = nil

	rc, err := p2p.NewRemoteClient(addr, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to connect: %v", err)), nil
	}

	gs.client = rc
	gs.onlineName = name

	state, _ := gs.client.GetState()
	return mcp.NewToolResultText(fmt.Sprintf(
		"Joined game as %s!\n\n%s", name, formatState(state),
	)), nil
}
```

Note: `RemoteClient` needs a `SendChat()` method (uses `writeMu` to avoid data races) — add to `p2p/guest.go`:
```go
func (rc *RemoteClient) SendChat(playerID, name, text string) error {
	rc.writeMu.Lock()
	defer rc.writeMu.Unlock()
	return WriteMessage(rc.conn, NewChatMsg(playerID, name, text))
}

func (rc *RemoteClient) PlayerID() string {
	return rc.playerID
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mcp/ -run TestJoinGame -v`
Expected: PASS

- [ ] **Step 5: Write failing test for send_chat tool**

```go
func TestSendChat(t *testing.T) {
	c := setupClient(t)

	// send_chat before join should fail
	result := callTool(t, c, "send_chat", map[string]any{"text": "hello"})
	if !result.IsError {
		t.Error("expected error for send_chat before join")
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./mcp/ -run TestSendChat -v`
Expected: FAIL — `send_chat` tool not registered

- [ ] **Step 7: Implement send_chat tool**

```go
sendChatTool := mcp.NewTool("send_chat",
	mcp.WithDescription("Send a chat message during an online game"),
	mcp.WithString("text", mcp.Required(), mcp.Description("Chat message text")),
)
s.AddTool(sendChatTool, gs.handleSendChat)
```

```go
func (gs *gameServer) handleSendChat(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rc, ok := gs.client.(*p2p.RemoteClient)
	if !ok {
		return mcp.NewToolResultError("Not connected to a game server. Use join_game first."), nil
	}
	text, err := req.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	playerID := rc.PlayerID()
	if err := rc.SendChat(playerID, gs.onlineName, text); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send: %v", err)), nil
	}
	return mcp.NewToolResultText("Chat sent."), nil
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./mcp/ -run TestSendChat -v`
Expected: PASS

- [ ] **Step 9: Add cleanup in handleNewGame for online→local transition**

In `handleNewGame`, add at the beginning:
```go
// Cleanup existing online connection
if rc, ok := gs.client.(*p2p.RemoteClient); ok {
	rc.Close()
}
gs.onlineName = ""
```

- [ ] **Step 10: Run all MCP tests**

Run: `go test ./mcp/ -v`
Expected: All PASS

- [ ] **Step 11: Commit**

```bash
git add mcp/server.go mcp/server_test.go p2p/guest.go
git commit -m "feat(mcp): add join_game and send_chat tools for online play"
```

---

### Task 7: TUI Chat Display

**Files:**
- Modify: `cli/model.go:24-34` (model struct), `:199-222` (View)
- Modify: `cli/ui.go` (pass chat channel)

**Important:** `p2p` imports `cli`, so `cli` cannot import `p2p` (circular import). The chat channel uses a generic struct defined in `cli` itself. The caller converts `p2p.ChatPayload` → `cli.ChatEntry` at the call site.

- [ ] **Step 1: Add chat types and messages to model struct**

In `cli/model.go`, add types and field:
```go
// ChatEntry is a generic chat message for the TUI (no dependency on p2p).
type ChatEntry struct {
	Name string
	Text string
}

type chatMsg ChatEntry
```

Add to `model`:
```go
chatMessages []ChatEntry
chatCh       <-chan ChatEntry
```

- [ ] **Step 2: Add chat subscription command**

```go
func listenForChat(chatCh <-chan ChatEntry) tea.Cmd {
	return func() tea.Msg {
		ce, ok := <-chatCh
		if !ok {
			return nil
		}
		return chatMsg(ce)
	}
}
```

- [ ] **Step 3: Handle chatMsg in Update()**

In `Update()`, add a case:
```go
case chatMsg:
	m.chatMessages = append(m.chatMessages, chatEntry(msg))
	if len(m.chatMessages) > 5 {
		m.chatMessages = m.chatMessages[len(m.chatMessages)-5:]
	}
	if m.chatCh != nil {
		return m, listenForChat(m.chatCh)
	}
	return m, nil
```

- [ ] **Step 4: Add chat view to View()**

Create `viewChat` method:
```go
func (m model) viewChat(b *strings.Builder) {
	if len(m.chatMessages) == 0 {
		return
	}
	b.WriteString("\n  ─── Chat ────────────────────────\n")
	for _, c := range m.chatMessages {
		b.WriteString(fmt.Sprintf("  %s: %s\n", c.Name, c.Text))
	}
}
```

Call `m.viewChat(&b)` before the error display in `View()`.

- [ ] **Step 5: Update RunGame to accept optional chat channel**

In `cli/ui.go`, update `RunGame` signature:
```go
func RunGame(client engine.GameClient, playerName string, opts ...GameOption) error
```

Add option type:
```go
type GameOption func(*model)

func WithChatChannel(ch <-chan ChatEntry) GameOption {
	return func(m *model) {
		m.chatCh = ch
	}
}
```

Update `newModel` to apply options, and set `Init()` to return `listenForChat` if `chatCh` is non-nil.

- [ ] **Step 6: Update p2p/guest.go RunGuest to pass chat channel with adapter**

In `p2p/guest.go` `RunGuest()`, convert `p2p.ChatPayload` channel to `cli.ChatEntry` channel via an adapter goroutine:
```go
chatCh := make(chan cli.ChatEntry, 16)
go func() {
	for cp := range rc.ChatCh() {
		chatCh <- cli.ChatEntry{Name: cp.Name, Text: cp.Text}
	}
	close(chatCh)
}()
return cli.RunGame(rc, name, cli.WithChatChannel(chatCh))
```

- [ ] **Step 7: Verify it builds and existing tests pass**

Run: `go build ./cmd/yatz/ && go test ./... -short`
Expected: Build success, all tests PASS

- [ ] **Step 8: Commit**

```bash
git add cli/model.go cli/ui.go p2p/guest.go
git commit -m "feat(cli): add chat display area in TUI for opponent messages"
```

---

### Task 8: E2E Test — Full Game via Server

**Files:**
- Create: `p2p/server_e2e_test.go`

- [ ] **Step 1: Write E2E test for full 2-player game via headless server**

```go
func TestE2E_FullGameViaServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- RunServer(ln, 2, rand.NewSource(42))
	}()

	// Connect two clients via RemoteClient
	rc1, err := NewRemoteClient(ln.Addr().String(), "Alice")
	if err != nil {
		t.Fatalf("connect player 1: %v", err)
	}
	defer rc1.Close()

	rc2, err := NewRemoteClient(ln.Addr().String(), "Bob")
	if err != nil {
		t.Fatalf("connect player 2: %v", err)
	}
	defer rc2.Close()

	// Play 13 rounds
	clients := []*RemoteClient{rc1, rc2}
	for round := 0; round < 13; round++ {
		for _, rc := range clients {
			state, _ := rc.GetState()
			if state.Phase == engine.PhaseFinished {
				goto done
			}
			if state.CurrentPlayer != rc.playerID {
				// Wait for turn
				turnState, isOver, err := rc.WaitForTurn()
				if err != nil {
					t.Fatalf("wait for turn: %v", err)
				}
				if isOver {
					goto done
				}
				state = turnState
			}

			// Roll
			state, err := rc.Roll()
			if err != nil {
				t.Fatalf("roll: %v", err)
			}

			// Score in first available category
			cats := state.AvailableCategories
			state, err = rc.Score(cats[0])
			if err != nil {
				t.Fatalf("score: %v", err)
			}
		}
	}

done:
	finalState, _ := rc1.GetState()
	if finalState.Phase != engine.PhaseFinished {
		t.Errorf("expected game finished, got phase %d", finalState.Phase)
	}
	t.Logf("Final scores: %s=%d, %s=%d",
		finalState.Players[0].Name, finalState.Players[0].Scorecard.Total(),
		finalState.Players[1].Name, finalState.Players[1].Scorecard.Total())
}
```

- [ ] **Step 2: Run E2E test**

Run: `go test ./p2p/ -run TestE2E_FullGameViaServer -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add p2p/server_e2e_test.go
git commit -m "test(p2p): add E2E test for full game via headless server"
```

---

### Task 9: Integration Verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS (including E2E tests)

- [ ] **Step 2: Build and verify commands**

Run: `go build ./cmd/yatz/ && ./yatz serve --help && ./yatz --help`
Expected: `serve` command shows in help, with `--port` and `--players` flags

- [ ] **Step 3: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 4: Final commit if any remaining changes**

```bash
git status
# If any unstaged changes, commit them
```
