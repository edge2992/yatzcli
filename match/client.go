package match

import (
	"fmt"
	"net"

	"github.com/gorilla/websocket"
)

// MatchResult is received from the matchmaking server
type MatchResult struct {
	OpponentAddr string `json:"opponent_addr"`
	OpponentName string `json:"opponent_name"`
	IsHost       bool   `json:"is_host"`
}

// ClientMessage is sent to the matchmaking server
type ClientMessage struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

// FindMatch connects to the matchmaking WebSocket API and waits for a match.
// Returns the match result with opponent info and host/guest role.
func FindMatch(wsURL string, name string, port int) (*MatchResult, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to matchmaking: %w", err)
	}
	defer conn.Close()

	msg := ClientMessage{Name: name, Port: port}
	if err := conn.WriteJSON(msg); err != nil {
		return nil, fmt.Errorf("failed to send registration: %w", err)
	}

	var result MatchResult
	if err := conn.ReadJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to read match result: %w", err)
	}

	return &result, nil
}

// GetFreePort finds an available TCP port
func GetFreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
