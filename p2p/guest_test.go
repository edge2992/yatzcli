package p2p

import (
	"math/rand"
	"net"
	"testing"

	"github.com/edge2992/yatzcli/engine"
)

// mockHost simulates a host for testing RemoteClient.
type mockHost struct {
	conn net.Conn
	game *engine.Game
}

func newMockHost(conn net.Conn, rngSrc rand.Source) (*mockHost, error) {
	msg, err := ReadMessage(conn)
	if err != nil {
		return nil, err
	}
	if msg.Type != MsgHandshake {
		return nil, err
	}
	hs, _ := DecodeHandshake(msg)

	if err := WriteMessage(conn, NewHandshakeMsg("MockHost")); err != nil {
		return nil, err
	}

	game := engine.NewGame([]string{"MockHost", hs.Name}, rngSrc)

	gs := game.GetState()
	if err := WriteMessage(conn, NewGameStartMsg(gs)); err != nil {
		return nil, err
	}

	return &mockHost{conn: conn, game: game}, nil
}

func (mh *mockHost) handleAction() error {
	msg, err := ReadMessage(mh.conn)
	if err != nil {
		return err
	}
	if msg.Type != MsgAction {
		return err
	}
	ap, _ := DecodeAction(msg)

	var actionErr error
	switch ap.Action {
	case ActionRoll:
		actionErr = mh.game.Roll()
	case ActionHold:
		actionErr = mh.game.Hold(ap.Indices)
	case ActionScore:
		actionErr = mh.game.Score(engine.Category(ap.Category))
	}

	if actionErr != nil {
		return WriteMessage(mh.conn, NewErrorMsg(actionErr.Error()))
	}

	gs := mh.game.GetState()
	return WriteMessage(mh.conn, NewStateUpdateMsg(gs))
}

func TestRemoteClient_Roll(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	mockHostCh := make(chan *mockHost, 1)
	go func() {
		mh, err := newMockHost(hostConn, rand.NewSource(42))
		if err != nil {
			return
		}
		mockHostCh <- mh
	}()

	rc, err := newRemoteClientFromConn(guestConn, "TestGuest")
	if err != nil {
		t.Fatalf("newRemoteClientFromConn: %v", err)
	}
	defer rc.Close()

	mh := <-mockHostCh

	// GetState should return initial state
	gs, err := rc.GetState()
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if gs.Phase != engine.PhaseRolling {
		t.Errorf("expected PhaseRolling, got %d", gs.Phase)
	}

	// Advance host game to guest's turn
	if err := mh.game.Roll(); err != nil {
		t.Fatalf("mock host roll: %v", err)
	}
	if err := mh.game.Score(engine.Ones); err != nil {
		t.Fatalf("mock host score: %v", err)
	}

	// Send turn_start to guest
	mockState := mh.game.GetState()
	if err := WriteMessage(hostConn, NewTurnStartMsg(mockState)); err != nil {
		t.Fatalf("send turn_start: %v", err)
	}

	// Wait for turn start
	state, isGameOver, err := rc.WaitForTurn()
	if err != nil {
		t.Fatalf("WaitForTurn: %v", err)
	}
	if isGameOver {
		t.Fatal("unexpected game over")
	}
	if state.CurrentPlayer != "player-1" {
		t.Errorf("expected player-1's turn, got %s", state.CurrentPlayer)
	}

	// Handle guest's roll action on the mock host side
	go func() {
		_ = mh.handleAction()
	}()

	gs, err = rc.Roll()
	if err != nil {
		t.Fatalf("Roll: %v", err)
	}
	if gs.RollCount != 1 {
		t.Errorf("expected roll count 1, got %d", gs.RollCount)
	}
	if gs.CurrentPlayer != "player-1" {
		t.Errorf("expected player-1's turn, got %s", gs.CurrentPlayer)
	}
}

func TestRemoteClient_Hold(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	mockHostCh := make(chan *mockHost, 1)
	go func() {
		mh, err := newMockHost(hostConn, rand.NewSource(42))
		if err != nil {
			return
		}
		mockHostCh <- mh
	}()

	rc, err := newRemoteClientFromConn(guestConn, "TestGuest")
	if err != nil {
		t.Fatalf("newRemoteClientFromConn: %v", err)
	}
	defer rc.Close()

	mh := <-mockHostCh

	// Advance to guest's turn
	if err := mh.game.Roll(); err != nil {
		t.Fatalf("mock host roll: %v", err)
	}
	if err := mh.game.Score(engine.Ones); err != nil {
		t.Fatalf("mock host score: %v", err)
	}
	mockState := mh.game.GetState()
	if err := WriteMessage(hostConn, NewTurnStartMsg(mockState)); err != nil {
		t.Fatalf("send turn_start: %v", err)
	}

	// Wait for turn start
	_, _, err = rc.WaitForTurn()
	if err != nil {
		t.Fatalf("WaitForTurn: %v", err)
	}

	// Roll first
	go func() { _ = mh.handleAction() }()
	gs, err := rc.Roll()
	if err != nil {
		t.Fatalf("Roll: %v", err)
	}
	if gs.RollCount != 1 {
		t.Errorf("expected roll count 1, got %d", gs.RollCount)
	}

	// Hold some dice
	go func() { _ = mh.handleAction() }()
	gs, err = rc.Hold([]int{0, 1})
	if err != nil {
		t.Fatalf("Hold: %v", err)
	}
	if gs.RollCount != 2 {
		t.Errorf("expected roll count 2, got %d", gs.RollCount)
	}
}

func TestRemoteClient_ErrorResponse(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	mockHostCh := make(chan *mockHost, 1)
	go func() {
		mh, err := newMockHost(hostConn, rand.NewSource(42))
		if err != nil {
			return
		}
		mockHostCh <- mh
	}()

	rc, err := newRemoteClientFromConn(guestConn, "TestGuest")
	if err != nil {
		t.Fatalf("newRemoteClientFromConn: %v", err)
	}
	defer rc.Close()

	mh := <-mockHostCh

	// Advance to guest's turn
	if err := mh.game.Roll(); err != nil {
		t.Fatalf("mock host roll: %v", err)
	}
	if err := mh.game.Score(engine.Ones); err != nil {
		t.Fatalf("mock host score: %v", err)
	}
	mockState := mh.game.GetState()
	if err := WriteMessage(hostConn, NewTurnStartMsg(mockState)); err != nil {
		t.Fatalf("send turn_start: %v", err)
	}

	_, _, _ = rc.WaitForTurn()

	// Try to hold before rolling (should error)
	go func() { _ = mh.handleAction() }()
	_, err = rc.Hold([]int{0})
	if err == nil {
		t.Fatal("expected error from Hold before Roll")
	}
}

func TestRemoteClient_Handshake(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	go func() {
		_, _ = newMockHost(hostConn, rand.NewSource(42))
	}()

	rc, err := newRemoteClientFromConn(guestConn, "Player2")
	if err != nil {
		t.Fatalf("newRemoteClientFromConn: %v", err)
	}
	defer rc.Close()

	if rc.playerID != "player-1" {
		t.Errorf("expected playerID player-1, got %s", rc.playerID)
	}
	if rc.playerName != "Player2" {
		t.Errorf("expected playerName Player2, got %s", rc.playerName)
	}

	gs, err := rc.GetState()
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if len(gs.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(gs.Players))
	}
}

func TestRemoteClient_ScoreAndWaitForTurn(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	mockHostCh := make(chan *mockHost, 1)
	go func() {
		mh, err := newMockHost(hostConn, rand.NewSource(42))
		if err != nil {
			return
		}
		mockHostCh <- mh
	}()

	rc, err := newRemoteClientFromConn(guestConn, "TestGuest")
	if err != nil {
		t.Fatalf("newRemoteClientFromConn: %v", err)
	}
	defer rc.Close()

	mh := <-mockHostCh

	// Advance to guest's turn
	if err := mh.game.Roll(); err != nil {
		t.Fatalf("mock host roll: %v", err)
	}
	if err := mh.game.Score(engine.Ones); err != nil {
		t.Fatalf("mock host score: %v", err)
	}
	mockState := mh.game.GetState()
	if err := WriteMessage(hostConn, NewTurnStartMsg(mockState)); err != nil {
		t.Fatalf("send turn_start: %v", err)
	}

	_, _, _ = rc.WaitForTurn()

	// Roll
	go func() { _ = mh.handleAction() }()
	gs, err := rc.Roll()
	if err != nil {
		t.Fatalf("Roll: %v", err)
	}

	// Score - this should send action, get state_update, then wait for turn_start
	avail := gs.AvailableCategories
	if len(avail) == 0 {
		t.Fatal("no available categories")
	}

	// The mock host will handle score, then we simulate the host playing and
	// sending turn_start back
	go func() {
		// Handle score action
		if err := mh.handleAction(); err != nil {
			return
		}
		// Simulate host playing their turn
		if err := mh.game.Roll(); err != nil {
			return
		}
		if err := mh.game.Score(engine.Twos); err != nil {
			return
		}
		// Send turn_start for guest's next turn
		gs := mh.game.GetState()
		_ = WriteMessage(hostConn, NewTurnStartMsg(gs))
	}()

	gs, err = rc.Score(avail[0])
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	// After Score returns, it should be guest's turn again
	if gs.CurrentPlayer != "player-1" {
		t.Errorf("expected player-1's turn after Score, got %s", gs.CurrentPlayer)
	}
}
