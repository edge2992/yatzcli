package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"sync"

	"github.com/edge2992/yatzcli/cli"
	"github.com/edge2992/yatzcli/engine"
)

// Host manages a P2P game session, holding the authoritative Game instance.
type Host struct {
	game      *engine.Game
	hostName  string
	guestName string
	port      int
	conn      net.Conn
	mu        sync.Mutex
}

// HostGameClient wraps a LocalClient and broadcasts state updates to the guest
// after each action. It implements engine.GameClient.
type HostGameClient struct {
	local *engine.LocalClient
	host  *Host
}

func (h *HostGameClient) Roll() (*engine.GameState, error) {
	gs, err := h.local.Roll()
	if err != nil {
		return nil, err
	}
	if err := h.host.sendStateUpdate(*gs); err != nil {
		return nil, fmt.Errorf("send state update: %w", err)
	}
	return gs, nil
}

func (h *HostGameClient) Hold(indices []int) (*engine.GameState, error) {
	gs, err := h.local.Hold(indices)
	if err != nil {
		return nil, err
	}
	if err := h.host.sendStateUpdate(*gs); err != nil {
		return nil, fmt.Errorf("send state update: %w", err)
	}
	return gs, nil
}

func (h *HostGameClient) Score(category engine.Category) (*engine.GameState, error) {
	gs, err := h.local.Score(category)
	if err != nil {
		return nil, err
	}
	if err := h.host.sendStateUpdate(*gs); err != nil {
		return nil, fmt.Errorf("send state update: %w", err)
	}
	if gs.Phase == engine.PhaseFinished {
		_ = h.host.sendGameOver(*gs)
		return gs, nil
	}
	// If it's now the guest's turn, handle their actions
	if gs.CurrentPlayer == "player-1" {
		finalState, err := h.host.handleGuestTurn()
		if err != nil {
			return nil, fmt.Errorf("handle guest turn: %w", err)
		}
		return finalState, nil
	}
	return gs, nil
}

func (h *HostGameClient) GetState() (*engine.GameState, error) {
	return h.local.GetState()
}

func (h *Host) sendStateUpdate(gs engine.GameState) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return WriteMessage(h.conn, NewStateUpdateMsg(gs))
}

func (h *Host) sendTurnStart(gs engine.GameState) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return WriteMessage(h.conn, NewTurnStartMsg(gs))
}

func (h *Host) sendGameOver(gs engine.GameState) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return WriteMessage(h.conn, NewGameOverMsg(gs))
}

func (h *Host) sendGameStart(gs engine.GameState) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return WriteMessage(h.conn, NewGameStartMsg(gs))
}

func (h *Host) sendError(errMsg string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return WriteMessage(h.conn, NewErrorMsg(errMsg))
}

// handleGuestTurn reads actions from the guest over TCP and applies them
// to the game engine. It loops until the guest scores (ending their turn)
// or the game finishes. Returns the final state after the guest's turn.
func (h *Host) handleGuestTurn() (*engine.GameState, error) {
	gs := h.game.GetState()
	if err := h.sendTurnStart(gs); err != nil {
		return nil, fmt.Errorf("send turn_start: %w", err)
	}

	for {
		msg, err := ReadMessage(h.conn)
		if err != nil {
			return nil, fmt.Errorf("read guest action: %w", err)
		}
		if msg.Type != MsgAction {
			return nil, fmt.Errorf("expected action message, got %s", msg.Type)
		}

		ap, err := DecodeAction(msg)
		if err != nil {
			return nil, fmt.Errorf("decode action: %w", err)
		}

		var actionErr error
		switch ap.Action {
		case ActionRoll:
			actionErr = h.game.Roll()
		case ActionHold:
			actionErr = h.game.Hold(ap.Indices)
		case ActionScore:
			actionErr = h.game.Score(engine.Category(ap.Category))
		default:
			actionErr = fmt.Errorf("unknown action: %s", ap.Action)
		}

		if actionErr != nil {
			if err := h.sendError(actionErr.Error()); err != nil {
				return nil, fmt.Errorf("send error: %w", err)
			}
			continue
		}

		state := h.game.GetState()
		if err := h.sendStateUpdate(state); err != nil {
			return nil, fmt.Errorf("send state_update: %w", err)
		}

		if state.Phase == engine.PhaseFinished {
			_ = h.sendGameOver(state)
			return &state, nil
		}

		// Score action ends the guest's turn
		if ap.Action == ActionScore {
			return &state, nil
		}
	}
}

// RunHost starts a P2P game as host. It listens on the given port, accepts
// one guest connection, performs a handshake, then runs the game.
func RunHost(port int, name string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer ln.Close()

	fmt.Printf("Waiting for guest on port %d...\n", port)

	conn, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("accept: %w", err)
	}
	defer conn.Close()

	return runHostWithConn(conn, name, nil)
}

// runHostWithConn runs the host game logic on an already-established connection.
// rngSrc can be nil for production (uses time-based seed).
func runHostWithConn(conn net.Conn, hostName string, rngSrc rand.Source) error {
	// Handshake: receive guest name, send host name
	msg, err := ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("read handshake: %w", err)
	}
	if msg.Type != MsgHandshake {
		return fmt.Errorf("expected handshake, got %s", msg.Type)
	}
	hs, err := DecodeHandshake(msg)
	if err != nil {
		return fmt.Errorf("decode handshake: %w", err)
	}
	guestName := hs.Name

	if err := WriteMessage(conn, NewHandshakeMsg(hostName)); err != nil {
		return fmt.Errorf("send handshake: %w", err)
	}

	// Create game
	game := engine.NewGame([]string{hostName, guestName}, rngSrc)
	localClient := engine.NewLocalClient(game, "player-0", nil)

	host := &Host{
		game:      game,
		hostName:  hostName,
		guestName: guestName,
		conn:      conn,
	}

	hostClient := &HostGameClient{
		local: localClient,
		host:  host,
	}

	// Send game_start
	gs := game.GetState()
	if err := host.sendGameStart(gs); err != nil {
		return fmt.Errorf("send game_start: %w", err)
	}

	// Run TUI for host player
	return cli.RunGame(hostClient, hostName)
}
