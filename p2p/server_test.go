package p2p

import (
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/edge2992/yatzcli/engine"
)

func connectAndHandshake(t *testing.T, addr, name string) (net.Conn, string) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Send handshake with name
	if err := WriteMessage(conn, NewHandshakeMsg(name)); err != nil {
		conn.Close()
		t.Fatalf("send handshake: %v", err)
	}

	// Read handshake response (server name + assigned playerID)
	msg := readExpectType(t, conn, MsgHandshake)
	hs, err := DecodeHandshake(msg)
	if err != nil {
		conn.Close()
		t.Fatalf("decode handshake: %v", err)
	}

	return conn, hs.PlayerID
}

func readExpectType(t *testing.T, conn net.Conn, expectedType string) *Message {
	t.Helper()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	msg, err := ReadMessage(conn)
	conn.SetDeadline(time.Time{})
	if err != nil {
		t.Fatalf("read message (expected %s): %v", expectedType, err)
	}
	if msg.Type != expectedType {
		t.Fatalf("expected %s, got %s", expectedType, msg.Type)
	}
	return msg
}

func TestServer_Handshake(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping server test in short mode")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunServer(ln, 2, rand.NewSource(42))
	}()

	// Connect two clients
	conn1, pid1 := connectAndHandshake(t, addr, "Alice")
	defer conn1.Close()
	conn2, pid2 := connectAndHandshake(t, addr, "Bob")
	defer conn2.Close()

	// Verify assigned playerIDs
	if pid1 != "player-0" {
		t.Errorf("expected player-0, got %s", pid1)
	}
	if pid2 != "player-1" {
		t.Errorf("expected player-1, got %s", pid2)
	}

	// Both should receive game_start
	msg1 := readExpectType(t, conn1, MsgGameStart)
	sp1, _ := DecodeState(msg1)
	if len(sp1.State.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(sp1.State.Players))
	}
	if sp1.State.Players[0].Name != "Alice" {
		t.Errorf("expected player-0 name Alice, got %s", sp1.State.Players[0].Name)
	}
	if sp1.State.Players[1].Name != "Bob" {
		t.Errorf("expected player-1 name Bob, got %s", sp1.State.Players[1].Name)
	}

	msg2 := readExpectType(t, conn2, MsgGameStart)
	sp2, _ := DecodeState(msg2)
	if len(sp2.State.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(sp2.State.Players))
	}

	// player-0 should get turn_start since they go first
	readExpectType(t, conn1, MsgTurnStart)

	// Close connections to let server exit
	conn1.Close()
	conn2.Close()
}

func TestServer_TurnManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping server test in short mode")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunServer(ln, 2, rand.NewSource(42))
	}()

	conn1, _ := connectAndHandshake(t, addr, "Alice")
	defer conn1.Close()
	conn2, _ := connectAndHandshake(t, addr, "Bob")
	defer conn2.Close()

	// Both get game_start
	readExpectType(t, conn1, MsgGameStart)
	readExpectType(t, conn2, MsgGameStart)

	// player-0 (Alice) gets turn_start
	readExpectType(t, conn1, MsgTurnStart)

	// Alice rolls
	if err := WriteMessage(conn1, NewActionMsg(ActionPayload{Action: ActionRoll})); err != nil {
		t.Fatalf("send roll: %v", err)
	}

	// Both get state_update
	msg1 := readExpectType(t, conn1, MsgStateUpdate)
	sp1, _ := DecodeState(msg1)
	if sp1.State.RollCount != 1 {
		t.Errorf("expected roll count 1, got %d", sp1.State.RollCount)
	}
	readExpectType(t, conn2, MsgStateUpdate)

	// Alice scores
	avail := sp1.State.AvailableCategories
	if len(avail) == 0 {
		t.Fatal("no available categories")
	}
	if err := WriteMessage(conn1, NewActionMsg(ActionPayload{
		Action:   ActionScore,
		Category: string(avail[0]),
	})); err != nil {
		t.Fatalf("send score: %v", err)
	}

	// Both get state_update from score
	readExpectType(t, conn1, MsgStateUpdate)
	readExpectType(t, conn2, MsgStateUpdate)

	// Now player-1 (Bob) should get turn_start
	readExpectType(t, conn2, MsgTurnStart)

	// Clean up
	conn1.Close()
	conn2.Close()
}

func TestServer_ChatBroadcast(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping server test in short mode")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	go func() {
		_ = RunServer(ln, 2, rand.NewSource(42))
	}()

	conn1, _ := connectAndHandshake(t, addr, "Alice")
	defer conn1.Close()
	conn2, _ := connectAndHandshake(t, addr, "Bob")
	defer conn2.Close()

	// Both get game_start
	readExpectType(t, conn1, MsgGameStart)
	readExpectType(t, conn2, MsgGameStart)

	// player-0 gets turn_start
	readExpectType(t, conn1, MsgTurnStart)

	// Bob sends a chat message
	if err := WriteMessage(conn2, NewChatMsg("player-1", "Bob", "Hello!")); err != nil {
		t.Fatalf("send chat: %v", err)
	}

	// Both clients should receive the chat broadcast
	chatMsg1 := readExpectType(t, conn1, MsgChat)
	cp1, err := DecodeChat(chatMsg1)
	if err != nil {
		t.Fatalf("decode chat: %v", err)
	}
	if cp1.Text != "Hello!" {
		t.Errorf("expected chat text 'Hello!', got %s", cp1.Text)
	}
	if cp1.PlayerID != "player-1" {
		t.Errorf("expected player-1, got %s", cp1.PlayerID)
	}

	chatMsg2 := readExpectType(t, conn2, MsgChat)
	cp2, err := DecodeChat(chatMsg2)
	if err != nil {
		t.Fatalf("decode chat: %v", err)
	}
	if cp2.Text != "Hello!" {
		t.Errorf("expected chat text 'Hello!', got %s", cp2.Text)
	}

	// Verify game still works after chat - Alice rolls
	if err := WriteMessage(conn1, NewActionMsg(ActionPayload{Action: ActionRoll})); err != nil {
		t.Fatalf("send roll: %v", err)
	}
	readExpectType(t, conn1, MsgStateUpdate)
	readExpectType(t, conn2, MsgStateUpdate)

	conn1.Close()
	conn2.Close()
}

func TestServer_InvalidAction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping server test in short mode")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	go func() {
		_ = RunServer(ln, 2, rand.NewSource(42))
	}()

	conn1, _ := connectAndHandshake(t, addr, "Alice")
	defer conn1.Close()
	conn2, _ := connectAndHandshake(t, addr, "Bob")
	defer conn2.Close()

	readExpectType(t, conn1, MsgGameStart)
	readExpectType(t, conn2, MsgGameStart)
	readExpectType(t, conn1, MsgTurnStart)

	// Alice sends hold before roll (invalid)
	if err := WriteMessage(conn1, NewActionMsg(ActionPayload{Action: ActionHold, Indices: []int{0}})); err != nil {
		t.Fatalf("send hold: %v", err)
	}

	// Alice should get error
	errMsg := readExpectType(t, conn1, MsgError)
	ep, _ := DecodeError(errMsg)
	if ep.Message == "" {
		t.Error("expected non-empty error message")
	}

	// Game should still work - Alice rolls
	if err := WriteMessage(conn1, NewActionMsg(ActionPayload{Action: ActionRoll})); err != nil {
		t.Fatalf("send roll: %v", err)
	}
	readExpectType(t, conn1, MsgStateUpdate)
	readExpectType(t, conn2, MsgStateUpdate)

	conn1.Close()
	conn2.Close()
}

func TestServer_FullGame(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping server test in short mode")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunServer(ln, 2, rand.NewSource(42))
	}()

	conn1, _ := connectAndHandshake(t, addr, "Alice")
	defer conn1.Close()
	conn2, _ := connectAndHandshake(t, addr, "Bob")
	defer conn2.Close()

	readExpectType(t, conn1, MsgGameStart)
	readExpectType(t, conn2, MsgGameStart)

	conns := [2]net.Conn{conn1, conn2}

	allCategories := []engine.Category{
		engine.Ones, engine.Twos, engine.Threes, engine.Fours, engine.Fives, engine.Sixes,
		engine.ThreeOfAKind, engine.FourOfAKind, engine.FullHouse, engine.SmallStraight,
		engine.LargeStraight, engine.Yahtzee, engine.Chance,
	}

	for round := 0; round < engine.MaxRounds; round++ {
		for player := 0; player < 2; player++ {
			current := conns[player]
			other := conns[1-player]

			// Current player gets turn_start
			readExpectType(t, current, MsgTurnStart)

			// Roll
			if err := WriteMessage(current, NewActionMsg(ActionPayload{Action: ActionRoll})); err != nil {
				t.Fatalf("round %d player %d roll: %v", round, player, err)
			}
			msg := readExpectType(t, current, MsgStateUpdate)
			readExpectType(t, other, MsgStateUpdate)

			sp, _ := DecodeState(msg)
			// Pick first available category
			cat := allCategories[round]
			// Make sure it's available
			found := false
			for _, c := range sp.State.AvailableCategories {
				if c == cat {
					found = true
					break
				}
			}
			if !found && len(sp.State.AvailableCategories) > 0 {
				cat = sp.State.AvailableCategories[0]
			}

			// Score
			if err := WriteMessage(current, NewActionMsg(ActionPayload{
				Action:   ActionScore,
				Category: string(cat),
			})); err != nil {
				t.Fatalf("round %d player %d score: %v", round, player, err)
			}

			stateMsg := readExpectType(t, current, MsgStateUpdate)
			readExpectType(t, other, MsgStateUpdate)

			sp2, _ := DecodeState(stateMsg)
			if sp2.State.Phase == engine.PhaseFinished {
				// Both should get game_over
				readExpectType(t, current, MsgGameOver)
				readExpectType(t, other, MsgGameOver)

				// Server should return
				if err := <-errCh; err != nil {
					t.Fatalf("server error: %v", err)
				}
				return
			}
		}
	}
	t.Fatal("game did not finish after all rounds")
}
