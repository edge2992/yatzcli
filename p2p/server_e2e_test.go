package p2p

import (
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/edge2992/yatzcli/engine"
)

func TestE2E_FullGameViaServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			panic("TestE2E_FullGameViaServer timed out after 30s")
		}
	}()
	defer close(done)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- RunServer(ln, 2, rand.NewSource(42))
	}()

	// Connect both clients concurrently because NewRemoteClient blocks
	// until game_start, which is only sent after all clients connect.
	type connectResult struct {
		rc  *RemoteClient
		err error
	}
	ch1 := make(chan connectResult, 1)
	ch2 := make(chan connectResult, 1)
	go func() {
		rc, err := NewRemoteClient(ln.Addr().String(), "Alice")
		ch1 <- connectResult{rc, err}
	}()
	go func() {
		rc, err := NewRemoteClient(ln.Addr().String(), "Bob")
		ch2 <- connectResult{rc, err}
	}()

	r := <-ch1
	if r.err != nil {
		t.Fatalf("connect Alice: %v", r.err)
	}
	rc1 := r.rc
	defer rc1.Close()

	r = <-ch2
	if r.err != nil {
		t.Fatalf("connect Bob: %v", r.err)
	}
	rc2 := r.rc
	defer rc2.Close()

	// playTurns plays all 13 rounds for a given client.
	// It calls WaitForTurn, then loops: Roll → Score (first available).
	// Score() blocks until the next turn starts (or game over).
	playTurns := func(rc *RemoteClient, name string) (*engine.GameState, error) {
		state, isGameOver, err := rc.WaitForTurn()
		if err != nil {
			return nil, errorf("%s: WaitForTurn: %v", name, err)
		}
		if isGameOver {
			return state, nil
		}

		for round := 1; round <= 13; round++ {
			state, err = rc.Roll()
			if err != nil {
				return nil, errorf("%s round %d: Roll: %v", name, round, err)
			}

			avail := state.AvailableCategories
			if len(avail) == 0 {
				return nil, errorf("%s round %d: no available categories", name, round)
			}

			state, err = rc.Score(avail[0])
			if err != nil {
				return nil, errorf("%s round %d: Score: %v", name, round, err)
			}

			if state.Phase == engine.PhaseFinished {
				return state, nil
			}
		}

		return state, nil
	}

	type playerResult struct {
		state *engine.GameState
		err   error
	}

	p1Done := make(chan playerResult, 1)
	p2Done := make(chan playerResult, 1)

	go func() {
		state, err := playTurns(rc1, "Alice")
		p1Done <- playerResult{state, err}
	}()
	go func() {
		state, err := playTurns(rc2, "Bob")
		p2Done <- playerResult{state, err}
	}()

	r1 := <-p1Done
	if r1.err != nil {
		t.Fatalf("Alice error: %v", r1.err)
	}
	r2 := <-p2Done
	if r2.err != nil {
		t.Fatalf("Bob error: %v", r2.err)
	}

	if err := <-serverErr; err != nil {
		t.Fatalf("server error: %v", err)
	}

	// Verify both players see PhaseFinished
	if r1.state.Phase != engine.PhaseFinished {
		t.Fatalf("Alice: expected PhaseFinished, got %d", r1.state.Phase)
	}
	if r2.state.Phase != engine.PhaseFinished {
		t.Fatalf("Bob: expected PhaseFinished, got %d", r2.state.Phase)
	}

	// Log final scores
	for _, p := range r1.state.Players {
		t.Logf("Player %s: %d points", p.Name, p.Scorecard.Total())
	}
}
