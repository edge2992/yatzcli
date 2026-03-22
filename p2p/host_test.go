package p2p

import (
	"math/rand"
	"net"
	"testing"

	"github.com/edge2992/yatzcli/engine"
)

func TestHostGuestIntegration(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	errCh := make(chan error, 1)

	// Run host in background (with a deterministic RNG)
	go func() {
		errCh <- runHostWithConn(hostConn, "Alice", rand.NewSource(42))
	}()

	// Guest side: handshake
	if err := WriteMessage(guestConn, NewHandshakeMsg("Bob")); err != nil {
		t.Fatalf("send handshake: %v", err)
	}

	// Receive host handshake
	msg, err := ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read handshake: %v", err)
	}
	if msg.Type != MsgHandshake {
		t.Fatalf("expected handshake, got %s", msg.Type)
	}
	hs, _ := DecodeHandshake(msg)
	if hs.Name != "Alice" {
		t.Fatalf("expected host name Alice, got %s", hs.Name)
	}

	// Receive game_start
	msg, err = ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read game_start: %v", err)
	}
	if msg.Type != MsgGameStart {
		t.Fatalf("expected game_start, got %s", msg.Type)
	}
	sp, _ := DecodeState(msg)
	if len(sp.State.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(sp.State.Players))
	}
	if sp.State.Players[0].Name != "Alice" {
		t.Errorf("expected player-0 name Alice, got %s", sp.State.Players[0].Name)
	}
	if sp.State.Players[1].Name != "Bob" {
		t.Errorf("expected player-1 name Bob, got %s", sp.State.Players[1].Name)
	}

	// The host (Alice) is player-0 and goes first.
	// The host TUI is blocking, so the host side is waiting for TUI input.
	// We just verify that the handshake and game_start were sent correctly.
	// Closing the guest connection will cause the host to exit.
	guestConn.Close()

	// The host should exit (with an error from the closed connection or TUI)
	// We don't check the exact error since TUI may fail in test env
}

func TestHostHandshake(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runHostWithConn(hostConn, "Host", rand.NewSource(1))
	}()

	// Send handshake from guest
	if err := WriteMessage(guestConn, NewHandshakeMsg("Guest")); err != nil {
		t.Fatalf("send handshake: %v", err)
	}

	// Read host handshake reply
	msg, err := ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read handshake reply: %v", err)
	}
	if msg.Type != MsgHandshake {
		t.Fatalf("expected handshake, got %s", msg.Type)
	}
	hs, _ := DecodeHandshake(msg)
	if hs.Name != "Host" {
		t.Errorf("expected host name 'Host', got %s", hs.Name)
	}

	// Read game_start
	msg, err = ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read game_start: %v", err)
	}
	if msg.Type != MsgGameStart {
		t.Fatalf("expected game_start, got %s", msg.Type)
	}
	sp, _ := DecodeState(msg)
	if sp.State.Phase != engine.PhaseRolling {
		t.Errorf("expected PhaseRolling, got %d", sp.State.Phase)
	}
	if sp.State.CurrentPlayer != "player-0" {
		t.Errorf("expected current player player-0, got %s", sp.State.CurrentPlayer)
	}

	guestConn.Close()
}

func TestHandleGuestTurn(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	game := engine.NewGame([]string{"Host", "Guest"}, rand.NewSource(42))
	host := &Host{
		game:      game,
		hostName:  "Host",
		guestName: "Guest",
		conn:      hostConn,
	}

	// Simulate: advance to guest's turn by making host score
	if err := game.Roll(); err != nil {
		t.Fatalf("roll: %v", err)
	}
	if err := game.Score(engine.Ones); err != nil {
		t.Fatalf("score: %v", err)
	}

	// Now it should be player-1's turn
	gs := game.GetState()
	if gs.CurrentPlayer != "player-1" {
		t.Fatalf("expected player-1's turn, got %s", gs.CurrentPlayer)
	}

	// Run handleGuestTurn in background
	resultCh := make(chan struct {
		state *engine.GameState
		err   error
	}, 1)
	go func() {
		state, err := host.handleGuestTurn()
		resultCh <- struct {
			state *engine.GameState
			err   error
		}{state, err}
	}()

	// Guest reads turn_start
	msg, err := ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read turn_start: %v", err)
	}
	if msg.Type != MsgTurnStart {
		t.Fatalf("expected turn_start, got %s", msg.Type)
	}

	// Guest sends roll action
	if err := WriteMessage(guestConn, NewActionMsg(ActionPayload{Action: ActionRoll})); err != nil {
		t.Fatalf("send roll: %v", err)
	}

	// Guest reads state_update after roll
	msg, err = ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read state_update: %v", err)
	}
	if msg.Type != MsgStateUpdate {
		t.Fatalf("expected state_update, got %s", msg.Type)
	}
	sp, _ := DecodeState(msg)
	if sp.State.RollCount != 1 {
		t.Errorf("expected roll count 1, got %d", sp.State.RollCount)
	}

	// Guest sends score action
	avail := sp.State.AvailableCategories
	if len(avail) == 0 {
		t.Fatal("no available categories")
	}
	if err := WriteMessage(guestConn, NewActionMsg(ActionPayload{
		Action:   ActionScore,
		Category: string(avail[0]),
	})); err != nil {
		t.Fatalf("send score: %v", err)
	}

	// Guest reads state_update after score
	msg, err = ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read state_update after score: %v", err)
	}
	if msg.Type != MsgStateUpdate {
		t.Fatalf("expected state_update, got %s", msg.Type)
	}

	// handleGuestTurn should return
	result := <-resultCh
	if result.err != nil {
		t.Fatalf("handleGuestTurn error: %v", result.err)
	}
	// After guest scores, it should be host's turn again
	if result.state.CurrentPlayer != "player-0" {
		t.Errorf("expected player-0's turn, got %s", result.state.CurrentPlayer)
	}
}

func TestHandleGuestTurnInvalidAction(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	game := engine.NewGame([]string{"Host", "Guest"}, rand.NewSource(42))
	host := &Host{
		game:      game,
		hostName:  "Host",
		guestName: "Guest",
		conn:      hostConn,
	}

	// Advance to guest's turn
	if err := game.Roll(); err != nil {
		t.Fatalf("roll: %v", err)
	}
	if err := game.Score(engine.Ones); err != nil {
		t.Fatalf("score: %v", err)
	}

	resultCh := make(chan error, 1)
	go func() {
		_, err := host.handleGuestTurn()
		resultCh <- err
	}()

	// Read turn_start
	msg, err := ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read turn_start: %v", err)
	}
	if msg.Type != MsgTurnStart {
		t.Fatalf("expected turn_start, got %s", msg.Type)
	}

	// Send an invalid action (hold before rolling)
	if err := WriteMessage(guestConn, NewActionMsg(ActionPayload{Action: ActionHold, Indices: []int{0}})); err != nil {
		t.Fatalf("send hold: %v", err)
	}

	// Should receive an error message
	msg, err = ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if msg.Type != MsgError {
		t.Fatalf("expected error, got %s", msg.Type)
	}

	// Now send valid roll
	if err := WriteMessage(guestConn, NewActionMsg(ActionPayload{Action: ActionRoll})); err != nil {
		t.Fatalf("send roll: %v", err)
	}

	// Read state_update
	msg, err = ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read state_update: %v", err)
	}
	if msg.Type != MsgStateUpdate {
		t.Fatalf("expected state_update, got %s", msg.Type)
	}

	sp, _ := DecodeState(msg)
	avail := sp.State.AvailableCategories

	// Score to end turn
	if err := WriteMessage(guestConn, NewActionMsg(ActionPayload{
		Action:   ActionScore,
		Category: string(avail[0]),
	})); err != nil {
		t.Fatalf("send score: %v", err)
	}

	// Read state_update
	msg, err = ReadMessage(guestConn)
	if err != nil {
		t.Fatalf("read state_update: %v", err)
	}
	if msg.Type != MsgStateUpdate {
		t.Fatalf("expected state_update, got %s", msg.Type)
	}

	if err := <-resultCh; err != nil {
		t.Fatalf("handleGuestTurn error: %v", err)
	}
}
