package p2p

import (
	"fmt"
	"net"
	"sync"

	"github.com/edge2992/yatzcli/cli"
	"github.com/edge2992/yatzcli/engine"
)

// RemoteClient implements engine.GameClient by sending actions to the host
// over TCP and receiving state updates back. All reads from the connection
// happen in a single background goroutine (listen), and action responses
// are delivered via responseCh.
type RemoteClient struct {
	conn       net.Conn
	writeMu    sync.Mutex
	stateMu    sync.Mutex
	lastState  *engine.GameState
	playerID   string
	playerName string

	// responseCh delivers state_update/error responses to sendAction calls.
	responseCh chan responseResult
	// expectResponse is true when sendAction is waiting for a response.
	// state_updates received while false are treated as broadcast updates
	// (e.g., host's Roll/Hold/Score during host's turn) and are dropped
	// after updating lastState, preventing listener deadlock.
	expectResponse bool
	expectMu       sync.Mutex
	// turnCh delivers turn_start notifications (guest's turn begins).
	turnCh chan *engine.GameState
	// gameOverCh delivers game_over state.
	gameOverCh chan *engine.GameState
	// chatCh delivers chat messages from the host.
	chatCh chan *ChatPayload
	// listenErr holds any fatal error from the listener.
	listenErr error
	listenMu  sync.Mutex
}

type responseResult struct {
	state *engine.GameState
	err   error
}

// NewRemoteClient connects to the host and performs the handshake.
func NewRemoteClient(addr string, name string) (*RemoteClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	rc, err := newRemoteClientFromConn(conn, name)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return rc, nil
}

// newRemoteClientFromConn creates a RemoteClient from an existing connection.
func newRemoteClientFromConn(conn net.Conn, name string) (*RemoteClient, error) {
	// Send handshake
	if err := WriteMessage(conn, NewHandshakeMsg(name)); err != nil {
		return nil, fmt.Errorf("send handshake: %w", err)
	}

	// Receive host handshake
	msg, err := ReadMessage(conn)
	if err != nil {
		return nil, fmt.Errorf("read handshake: %w", err)
	}
	if msg.Type != MsgHandshake {
		return nil, fmt.Errorf("expected handshake, got %s", msg.Type)
	}
	hs, err := DecodeHandshake(msg)
	if err != nil {
		return nil, fmt.Errorf("decode handshake: %w", err)
	}
	playerID := hs.PlayerID
	if playerID == "" {
		playerID = "player-1" // backward compat with old host
	}

	// Wait for game_start
	msg, err = ReadMessage(conn)
	if err != nil {
		return nil, fmt.Errorf("read game_start: %w", err)
	}
	if msg.Type != MsgGameStart {
		return nil, fmt.Errorf("expected game_start, got %s", msg.Type)
	}
	sp, err := DecodeState(msg)
	if err != nil {
		return nil, fmt.Errorf("decode game_start: %w", err)
	}

	rc := &RemoteClient{
		conn:       conn,
		lastState:  &sp.State,
		playerID:   playerID,
		playerName: name,
		responseCh: make(chan responseResult, 1),
		turnCh:     make(chan *engine.GameState, 1),
		gameOverCh: make(chan *engine.GameState, 1),
		chatCh:     make(chan *ChatPayload, 16),
	}

	go rc.listen()

	return rc, nil
}

// listen reads all messages from the host and dispatches them.
func (rc *RemoteClient) listen() {
	for {
		msg, err := ReadMessage(rc.conn)
		if err != nil {
			connErr := fmt.Errorf("connection lost: %w", err)
			rc.listenMu.Lock()
			rc.listenErr = connErr
			rc.listenMu.Unlock()
			// Unblock anyone waiting
			select {
			case rc.responseCh <- responseResult{err: connErr}:
			default:
			}
			select {
			case rc.turnCh <- nil:
			default:
			}
			return
		}

		switch msg.Type {
		case MsgStateUpdate:
			sp, err := DecodeState(msg)
			if err != nil {
				rc.expectMu.Lock()
				expecting := rc.expectResponse
				rc.expectMu.Unlock()
				if expecting {
					rc.responseCh <- responseResult{err: fmt.Errorf("decode state_update: %w", err)}
				}
				continue
			}
			rc.setLastState(&sp.State)
			// Only deliver to responseCh if sendAction is waiting.
			// Otherwise this is a broadcast from the host's own turn.
			rc.expectMu.Lock()
			expecting := rc.expectResponse
			rc.expectMu.Unlock()
			if expecting {
				rc.responseCh <- responseResult{state: &sp.State}
			}

		case MsgError:
			rc.expectMu.Lock()
			expecting := rc.expectResponse
			rc.expectMu.Unlock()
			ep, err := DecodeError(msg)
			if err != nil {
				if expecting {
					rc.responseCh <- responseResult{err: fmt.Errorf("decode error response: %w", err)}
				}
				continue
			}
			if expecting {
				rc.responseCh <- responseResult{err: fmt.Errorf("%s", ep.Message)}
			}

		case MsgTurnStart:
			sp, err := DecodeState(msg)
			if err != nil {
				continue
			}
			rc.setLastState(&sp.State)
			rc.turnCh <- &sp.State

		case MsgChat:
			cp, err := DecodeChat(msg)
			if err != nil {
				continue
			}
			select {
			case rc.chatCh <- cp:
			default: // drop if buffer full
			}

		case MsgGameOver:
			sp, err := DecodeState(msg)
			if err != nil {
				continue
			}
			rc.setLastState(&sp.State)
			rc.gameOverCh <- &sp.State
			return
		}
	}
}

func (rc *RemoteClient) setLastState(gs *engine.GameState) {
	rc.stateMu.Lock()
	defer rc.stateMu.Unlock()
	rc.lastState = gs
}

func (rc *RemoteClient) getLastState() *engine.GameState {
	rc.stateMu.Lock()
	defer rc.stateMu.Unlock()
	return rc.lastState
}

// sendAction sends an action and waits for the response from the listener.
func (rc *RemoteClient) sendAction(ap ActionPayload) (*engine.GameState, error) {
	rc.expectMu.Lock()
	rc.expectResponse = true
	rc.expectMu.Unlock()

	rc.writeMu.Lock()
	err := WriteMessage(rc.conn, NewActionMsg(ap))
	rc.writeMu.Unlock()
	if err != nil {
		rc.expectMu.Lock()
		rc.expectResponse = false
		rc.expectMu.Unlock()
		return nil, fmt.Errorf("send action: %w", err)
	}

	result := <-rc.responseCh

	rc.expectMu.Lock()
	rc.expectResponse = false
	rc.expectMu.Unlock()

	return result.state, result.err
}

func (rc *RemoteClient) Roll() (*engine.GameState, error) {
	return rc.sendAction(ActionPayload{Action: ActionRoll})
}

func (rc *RemoteClient) Hold(indices []int) (*engine.GameState, error) {
	return rc.sendAction(ActionPayload{Action: ActionHold, Indices: indices})
}

func (rc *RemoteClient) Score(category engine.Category) (*engine.GameState, error) {
	gs, err := rc.sendAction(ActionPayload{Action: ActionScore, Category: string(category)})
	if err != nil {
		return nil, err
	}

	if gs.Phase == engine.PhaseFinished {
		return gs, nil
	}

	// After scoring, it's the host's turn. Wait for turn_start or game_over.
	select {
	case state := <-rc.turnCh:
		if state == nil {
			// Listener errored
			rc.listenMu.Lock()
			err := rc.listenErr
			rc.listenMu.Unlock()
			return nil, err
		}
		return state, nil
	case state := <-rc.gameOverCh:
		return state, nil
	}
}

func (rc *RemoteClient) GetState() (*engine.GameState, error) {
	return rc.getLastState(), nil
}

// WaitForTurn blocks until it's this player's turn or game is over.
func (rc *RemoteClient) WaitForTurn() (*engine.GameState, bool, error) {
	select {
	case state := <-rc.turnCh:
		if state == nil {
			rc.listenMu.Lock()
			err := rc.listenErr
			rc.listenMu.Unlock()
			return nil, false, err
		}
		return state, false, nil
	case state := <-rc.gameOverCh:
		return state, true, nil
	}
}

func (rc *RemoteClient) ChatCh() <-chan *ChatPayload {
	return rc.chatCh
}

func (rc *RemoteClient) SendChat(playerID, name, text string) error {
	rc.writeMu.Lock()
	defer rc.writeMu.Unlock()
	return WriteMessage(rc.conn, NewChatMsg(playerID, name, text))
}

func (rc *RemoteClient) PlayerID() string {
	return rc.playerID
}

func (rc *RemoteClient) Close() error {
	return rc.conn.Close()
}

// RunGuest connects to a host and plays as the guest.
func RunGuest(addr string, name string) error {
	rc, err := NewRemoteClient(addr, name)
	if err != nil {
		return err
	}
	defer rc.Close()

	gs := rc.getLastState()

	// If the host goes first, wait for our turn before starting TUI
	if gs.CurrentPlayer != rc.playerID {
		state, isGameOver, err := rc.WaitForTurn()
		if err != nil {
			return err
		}
		if isGameOver {
			fmt.Printf("Game over before your turn! Final state received.\n")
			_ = state
			return nil
		}
	}

	chatCh := make(chan cli.ChatEntry, 16)
	go func() {
		for cp := range rc.ChatCh() {
			chatCh <- cli.ChatEntry{Name: cp.Name, Text: cp.Text}
		}
		close(chatCh)
	}()
	return cli.RunGame(rc, name, cli.WithChatChannel(chatCh))
}
