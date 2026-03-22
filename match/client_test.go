package match

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestFindMatch_ReceivesMatchResult(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		// Read registration
		var msg ClientMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatal(err)
		}
		if msg.Name != "Alice" {
			t.Errorf("expected name Alice, got %s", msg.Name)
		}
		if msg.Port != 9876 {
			t.Errorf("expected port 9876, got %d", msg.Port)
		}

		// Send match result
		conn.WriteJSON(MatchResult{
			OpponentAddr: "192.168.1.10:9876",
			OpponentName: "Bob",
			IsHost:       true,
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	result, err := FindMatch(wsURL, "Alice", 9876)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsHost {
		t.Error("expected to be host")
	}
	if result.OpponentAddr != "192.168.1.10:9876" {
		t.Errorf("expected opponent addr 192.168.1.10:9876, got %s", result.OpponentAddr)
	}
	if result.OpponentName != "Bob" {
		t.Errorf("expected opponent name Bob, got %s", result.OpponentName)
	}
}

func TestFindMatch_ConnectionError(t *testing.T) {
	_, err := FindMatch("ws://localhost:1", "Alice", 9876)
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

func TestGetFreePort(t *testing.T) {
	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port == 0 {
		t.Error("expected non-zero port")
	}
}
