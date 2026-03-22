package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/edge2992/yatzcli/engine"
)

type clientConn struct {
	conn     net.Conn
	name     string
	playerID string
	actionCh chan *ActionPayload
	mu       sync.Mutex // protects conn writes
}

func writeToClient(cc *clientConn, msg *Message) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return WriteMessage(cc.conn, msg)
}

func broadcast(clients []*clientConn, msg *Message) {
	for _, cc := range clients {
		_ = writeToClient(cc, msg)
	}
}

func notifyDisconnect(clients []*clientConn, disconnectedName string, excludeIdx int) {
	errMsg := NewErrorMsg(fmt.Sprintf("player %q disconnected", disconnectedName))
	for i, cc := range clients {
		if i == excludeIdx {
			continue
		}
		_ = writeToClient(cc, errMsg)
	}
}

func acceptClients(ln net.Listener, numPlayers int) ([]*clientConn, error) {
	clients := make([]*clientConn, 0, numPlayers)
	for i := 0; i < numPlayers; i++ {
		conn, err := ln.Accept()
		if err != nil {
			// Close already-accepted connections
			for _, cc := range clients {
				cc.conn.Close()
			}
			return nil, fmt.Errorf("accept client %d: %w", i, err)
		}

		// Handshake with timeout
		conn.SetDeadline(time.Now().Add(30 * time.Second))
		msg, err := ReadMessage(conn)
		if err != nil {
			conn.Close()
			for _, cc := range clients {
				cc.conn.Close()
			}
			return nil, fmt.Errorf("read handshake from client %d: %w", i, err)
		}
		if msg.Type != MsgHandshake {
			conn.Close()
			for _, cc := range clients {
				cc.conn.Close()
			}
			return nil, fmt.Errorf("expected handshake from client %d, got %s", i, msg.Type)
		}
		hs, err := DecodeHandshake(msg)
		if err != nil {
			conn.Close()
			for _, cc := range clients {
				cc.conn.Close()
			}
			return nil, fmt.Errorf("decode handshake from client %d: %w", i, err)
		}

		playerID := fmt.Sprintf("player-%d", i)

		// Respond with server name + assigned playerID
		resp := newMessage(MsgHandshake, HandshakePayload{
			Name:     "server",
			PlayerID: playerID,
		})
		if err := WriteMessage(conn, resp); err != nil {
			conn.Close()
			for _, cc := range clients {
				cc.conn.Close()
			}
			return nil, fmt.Errorf("send handshake to client %d: %w", i, err)
		}
		conn.SetDeadline(time.Time{}) // clear deadline

		clients = append(clients, &clientConn{
			conn:     conn,
			name:     hs.Name,
			playerID: playerID,
			actionCh: make(chan *ActionPayload, 8),
		})
	}
	return clients, nil
}

func readLoop(cc *clientConn, clients []*clientConn, clientIdx int) {
	for {
		msg, err := ReadMessage(cc.conn)
		if err != nil {
			notifyDisconnect(clients, cc.name, clientIdx)
			// Send nil to actionCh to signal disconnect
			close(cc.actionCh)
			return
		}

		switch msg.Type {
		case MsgAction:
			ap, err := DecodeAction(msg)
			if err != nil {
				_ = writeToClient(cc, NewErrorMsg(fmt.Sprintf("invalid action: %v", err)))
				continue
			}
			cc.actionCh <- ap

		case MsgChat:
			// Broadcast chat to all clients
			broadcast(clients, msg)

		default:
			_ = writeToClient(cc, NewErrorMsg(fmt.Sprintf("unexpected message type: %s", msg.Type)))
		}
	}
}

func gameLoop(game *engine.Game, clients []*clientConn) error {
	for {
		state := game.GetState()
		if state.Phase == engine.PhaseFinished {
			broadcast(clients, NewGameOverMsg(state))
			return nil
		}

		currentIdx := state.CurrentPlayerIndex
		cc := clients[currentIdx]

		// Send turn_start to current player
		if err := writeToClient(cc, NewTurnStartMsg(state)); err != nil {
			return fmt.Errorf("send turn_start to %s: %w", cc.playerID, err)
		}

		// Process actions from current player until they score
		if err := processPlayerTurn(game, cc, clients); err != nil {
			return err
		}
	}
}

func processPlayerTurn(game *engine.Game, cc *clientConn, clients []*clientConn) error {
	for {
		ap, ok := <-cc.actionCh
		if !ok {
			return fmt.Errorf("player %s disconnected", cc.playerID)
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
			if err := writeToClient(cc, NewErrorMsg(actionErr.Error())); err != nil {
				return fmt.Errorf("send error to %s: %w", cc.playerID, err)
			}
			continue
		}

		state := game.GetState()
		broadcast(clients, NewStateUpdateMsg(state))

		if state.Phase == engine.PhaseFinished {
			broadcast(clients, NewGameOverMsg(state))
			return nil
		}

		// Score action ends the turn
		if ap.Action == ActionScore {
			return nil
		}
	}
}

// RunServer accepts numPlayers TCP connections, runs a headless Yahtzee game,
// and broadcasts state updates to all clients.
func RunServer(ln net.Listener, numPlayers int, rngSrc rand.Source) error {
	clients, err := acceptClients(ln, numPlayers)
	if err != nil {
		return fmt.Errorf("accept clients: %w", err)
	}
	defer func() {
		for _, cc := range clients {
			cc.conn.Close()
		}
	}()

	// Build player names list
	names := make([]string, len(clients))
	for i, cc := range clients {
		names[i] = cc.name
	}

	game := engine.NewGame(names, rngSrc)

	// Broadcast game_start
	state := game.GetState()
	broadcast(clients, NewGameStartMsg(state))

	// Start per-client reader goroutines
	for i, cc := range clients {
		go readLoop(cc, clients, i)
	}

	return gameLoop(game, clients)
}
