package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/edge2992/yatzcli/engine"
)

// TestE2E_FullGameP2P plays a complete 13-round 2-player Yahtzee game
// over the P2P protocol using net.Pipe(). Host (player-0) and guest
// (player-1) alternate turns, each rolling once then scoring.
func TestE2E_FullGameP2P(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	game := engine.NewGame([]string{"Alice", "Bob"}, rand.NewSource(42))

	host := &Host{
		game:      game,
		hostName:  "Alice",
		guestName: "Bob",
		conn:      hostConn,
	}

	type guestResult struct {
		finalState *engine.GameState
		err        error
	}
	guestDone := make(chan guestResult, 1)

	// Run guest protocol handling in a goroutine (reads from guestConn)
	go func() {
		// Read game_start
		msg, err := ReadMessage(guestConn)
		if err != nil {
			guestDone <- guestResult{err: err}
			return
		}
		if msg.Type != MsgGameStart {
			guestDone <- guestResult{err: errorf("expected game_start, got %s", msg.Type)}
			return
		}
		sp, _ := DecodeState(msg)
		if len(sp.State.Players) != 2 {
			guestDone <- guestResult{err: errorf("expected 2 players, got %d", len(sp.State.Players))}
			return
		}

		var lastState *engine.GameState

		for round := 1; round <= 13; round++ {
			// Read state_update from host's score action
			msg, err := ReadMessage(guestConn)
			if err != nil {
				guestDone <- guestResult{err: errorf("round %d: guest read host state_update: %v", round, err)}
				return
			}
			if msg.Type != MsgStateUpdate {
				guestDone <- guestResult{err: errorf("round %d: expected state_update, got %s", round, msg.Type)}
				return
			}

			// Read turn_start
			msg, err = ReadMessage(guestConn)
			if err != nil {
				guestDone <- guestResult{err: errorf("round %d: guest read turn_start: %v", round, err)}
				return
			}
			if msg.Type != MsgTurnStart {
				guestDone <- guestResult{err: errorf("round %d: expected turn_start, got %s", round, msg.Type)}
				return
			}

			// Roll
			if err := WriteMessage(guestConn, NewActionMsg(ActionPayload{Action: ActionRoll})); err != nil {
				guestDone <- guestResult{err: err}
				return
			}

			// Read state_update after roll
			msg, err = ReadMessage(guestConn)
			if err != nil {
				guestDone <- guestResult{err: err}
				return
			}
			sp, _ := DecodeState(msg)

			// Score first available category
			avail := sp.State.AvailableCategories
			if err := WriteMessage(guestConn, NewActionMsg(ActionPayload{
				Action:   ActionScore,
				Category: string(avail[0]),
			})); err != nil {
				guestDone <- guestResult{err: err}
				return
			}

			// Read state_update after score
			msg, err = ReadMessage(guestConn)
			if err != nil {
				guestDone <- guestResult{err: err}
				return
			}
			scoreSp, _ := DecodeState(msg)
			lastState = &scoreSp.State

			// If last round, expect game_over
			if round == 13 {
				msg, err = ReadMessage(guestConn)
				if err != nil {
					guestDone <- guestResult{err: err}
					return
				}
				if msg.Type != MsgGameOver {
					guestDone <- guestResult{err: errorf("expected game_over, got %s", msg.Type)}
					return
				}
				goSp, _ := DecodeState(msg)
				lastState = &goSp.State
			}
		}

		guestDone <- guestResult{finalState: lastState}
	}()

	// Host side: send game_start, then play turns
	gs := game.GetState()
	if err := host.sendGameStart(gs); err != nil {
		t.Fatalf("send game_start: %v", err)
	}

	for round := 1; round <= 13; round++ {
		// Host rolls and scores
		if err := game.Roll(); err != nil {
			t.Fatalf("round %d: host roll: %v", round, err)
		}
		avail := game.GetAvailableCategories()
		if err := game.Score(avail[0]); err != nil {
			t.Fatalf("round %d: host score: %v", round, err)
		}

		gs := game.GetState()
		if err := host.sendStateUpdate(gs); err != nil {
			t.Fatalf("round %d: send state_update: %v", round, err)
		}

		if gs.Phase == engine.PhaseFinished {
			break
		}

		// Handle guest turn
		guestState, err := host.handleGuestTurn()
		if err != nil {
			t.Fatalf("round %d: handleGuestTurn: %v", round, err)
		}

		if round < 13 && guestState.CurrentPlayer != "player-0" {
			t.Fatalf("round %d: expected player-0 after guest turn, got %s", round, guestState.CurrentPlayer)
		}
	}

	// Wait for guest to finish
	select {
	case result := <-guestDone:
		if result.err != nil {
			t.Fatalf("guest error: %v", result.err)
		}
		if result.finalState.Phase != engine.PhaseFinished {
			t.Fatalf("guest: expected PhaseFinished, got %d", result.finalState.Phase)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("guest timed out")
	}

	// Verify final state
	finalState := game.GetState()
	if finalState.Phase != engine.PhaseFinished {
		t.Fatalf("expected PhaseFinished, got %d", finalState.Phase)
	}
	for _, p := range finalState.Players {
		avail := p.Scorecard.AvailableCategories()
		if len(avail) != 0 {
			t.Errorf("player %s has %d unfilled categories", p.Name, len(avail))
		}
	}
	t.Logf("Game complete! Alice: %d, Bob: %d",
		finalState.Players[0].Scorecard.Total(),
		finalState.Players[1].Scorecard.Total())
}

// TestE2E_FullGameP2PWithHold plays a full game where the guest uses
// roll → hold → score each turn.
func TestE2E_FullGameP2PWithHold(t *testing.T) {
	hostConn, guestConn := net.Pipe()
	defer hostConn.Close()
	defer guestConn.Close()

	game := engine.NewGame([]string{"Host", "Guest"}, rand.NewSource(99))

	host := &Host{
		game:      game,
		hostName:  "Host",
		guestName: "Guest",
		conn:      hostConn,
	}

	type guestResult struct {
		err error
	}
	guestDone := make(chan guestResult, 1)

	go func() {
		// Read game_start
		msg, err := ReadMessage(guestConn)
		if err != nil {
			guestDone <- guestResult{err: err}
			return
		}
		if msg.Type != MsgGameStart {
			guestDone <- guestResult{err: errorf("expected game_start, got %s", msg.Type)}
			return
		}

		for round := 1; round <= 13; round++ {
			// Read state_update from host action
			msg, err := ReadMessage(guestConn)
			if err != nil {
				guestDone <- guestResult{err: err}
				return
			}
			if msg.Type != MsgStateUpdate {
				guestDone <- guestResult{err: errorf("round %d: expected state_update, got %s", round, msg.Type)}
				return
			}

			// Read turn_start
			msg, err = ReadMessage(guestConn)
			if err != nil {
				guestDone <- guestResult{err: err}
				return
			}
			if msg.Type != MsgTurnStart {
				guestDone <- guestResult{err: errorf("round %d: expected turn_start, got %s", round, msg.Type)}
				return
			}

			// Roll
			WriteMessage(guestConn, NewActionMsg(ActionPayload{Action: ActionRoll}))
			msg, _ = ReadMessage(guestConn) // state_update

			// Hold dice 0,1 and reroll
			WriteMessage(guestConn, NewActionMsg(ActionPayload{Action: ActionHold, Indices: []int{0, 1}}))
			msg, _ = ReadMessage(guestConn) // state_update
			sp, _ := DecodeState(msg)
			if sp.State.RollCount != 2 {
				guestDone <- guestResult{err: errorf("round %d: expected roll count 2, got %d", round, sp.State.RollCount)}
				return
			}

			// Score
			avail := sp.State.AvailableCategories
			WriteMessage(guestConn, NewActionMsg(ActionPayload{
				Action:   ActionScore,
				Category: string(avail[0]),
			}))
			ReadMessage(guestConn) // state_update after score

			if round == 13 {
				msg, _ = ReadMessage(guestConn) // game_over
				if msg.Type != MsgGameOver {
					guestDone <- guestResult{err: errorf("expected game_over, got %s", msg.Type)}
					return
				}
			}
		}
		guestDone <- guestResult{}
	}()

	// Host side
	gs := game.GetState()
	if err := host.sendGameStart(gs); err != nil {
		t.Fatalf("send game_start: %v", err)
	}

	for round := 1; round <= 13; round++ {
		// Host: roll, hold, score
		if err := game.Roll(); err != nil {
			t.Fatalf("round %d: host roll: %v", round, err)
		}
		if err := game.Hold([]int{0, 1}); err != nil {
			t.Fatalf("round %d: host hold: %v", round, err)
		}
		avail := game.GetAvailableCategories()
		if err := game.Score(avail[0]); err != nil {
			t.Fatalf("round %d: host score: %v", round, err)
		}

		gs := game.GetState()
		if err := host.sendStateUpdate(gs); err != nil {
			t.Fatalf("round %d: send state_update: %v", round, err)
		}

		if gs.Phase == engine.PhaseFinished {
			break
		}

		if _, err := host.handleGuestTurn(); err != nil {
			t.Fatalf("round %d: handleGuestTurn: %v", round, err)
		}
	}

	select {
	case result := <-guestDone:
		if result.err != nil {
			t.Fatalf("guest error: %v", result.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("guest timed out")
	}

	finalState := game.GetState()
	if finalState.Phase != engine.PhaseFinished {
		t.Fatalf("expected PhaseFinished, got %d", finalState.Phase)
	}
	for _, p := range finalState.Players {
		if len(p.Scorecard.AvailableCategories()) != 0 {
			t.Errorf("player %s has unfilled categories", p.Name)
		}
	}
	t.Logf("Game complete! Host: %d, Guest: %d",
		finalState.Players[0].Scorecard.Total(),
		finalState.Players[1].Scorecard.Total())
}

func errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
