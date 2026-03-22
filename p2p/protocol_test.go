package p2p

import (
	"bytes"
	"testing"

	"github.com/edge2992/yatzcli/engine"
)

func sampleState() engine.GameState {
	return engine.GameState{
		Players: []engine.PlayerState{
			{ID: "player-0", Name: "Alice", Scorecard: engine.NewScorecard()},
			{ID: "player-1", Name: "Bob", Scorecard: engine.NewScorecard()},
		},
		CurrentPlayer:       "player-0",
		CurrentPlayerIndex:  0,
		Round:               1,
		Dice:                [5]int{1, 2, 3, 4, 5},
		RollCount:           1,
		Phase:               engine.PhaseRolling,
		AvailableCategories: []engine.Category{engine.Ones, engine.Twos},
	}
}

func TestMessage_RoundTrip_Handshake(t *testing.T) {
	msg := NewHandshakeMsg("Alice")
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got.Type != MsgHandshake {
		t.Fatalf("type = %q, want %q", got.Type, MsgHandshake)
	}
	payload, err := DecodeHandshake(got)
	if err != nil {
		t.Fatalf("DecodeHandshake: %v", err)
	}
	if payload.Name != "Alice" {
		t.Errorf("name = %q, want %q", payload.Name, "Alice")
	}
}

func TestMessage_RoundTrip_Action(t *testing.T) {
	tests := []struct {
		name string
		ap   ActionPayload
	}{
		{"roll", ActionPayload{Action: ActionRoll}},
		{"hold", ActionPayload{Action: ActionHold, Indices: []int{0, 2, 4}}},
		{"score", ActionPayload{Action: ActionScore, Category: string(engine.FullHouse)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewActionMsg(tt.ap)
			var buf bytes.Buffer
			if err := WriteMessage(&buf, msg); err != nil {
				t.Fatalf("WriteMessage: %v", err)
			}
			got, err := ReadMessage(&buf)
			if err != nil {
				t.Fatalf("ReadMessage: %v", err)
			}
			if got.Type != MsgAction {
				t.Fatalf("type = %q, want %q", got.Type, MsgAction)
			}
			payload, err := DecodeAction(got)
			if err != nil {
				t.Fatalf("DecodeAction: %v", err)
			}
			if payload.Action != tt.ap.Action {
				t.Errorf("action = %q, want %q", payload.Action, tt.ap.Action)
			}
			if len(payload.Indices) != len(tt.ap.Indices) {
				t.Errorf("indices len = %d, want %d", len(payload.Indices), len(tt.ap.Indices))
			}
			if payload.Category != tt.ap.Category {
				t.Errorf("category = %q, want %q", payload.Category, tt.ap.Category)
			}
		})
	}
}

func TestMessage_RoundTrip_StateUpdate(t *testing.T) {
	state := sampleState()
	msg := NewStateUpdateMsg(state)
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got.Type != MsgStateUpdate {
		t.Fatalf("type = %q, want %q", got.Type, MsgStateUpdate)
	}
	payload, err := DecodeState(got)
	if err != nil {
		t.Fatalf("DecodeState: %v", err)
	}
	if payload.State.CurrentPlayer != "player-0" {
		t.Errorf("current player = %q, want %q", payload.State.CurrentPlayer, "player-0")
	}
	if payload.State.Dice != state.Dice {
		t.Errorf("dice = %v, want %v", payload.State.Dice, state.Dice)
	}
	if len(payload.State.Players) != 2 {
		t.Errorf("players len = %d, want 2", len(payload.State.Players))
	}
}

func TestMessage_RoundTrip_Error(t *testing.T) {
	msg := NewErrorMsg("something went wrong")
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got.Type != MsgError {
		t.Fatalf("type = %q, want %q", got.Type, MsgError)
	}
	payload, err := DecodeError(got)
	if err != nil {
		t.Fatalf("DecodeError: %v", err)
	}
	if payload.Message != "something went wrong" {
		t.Errorf("message = %q, want %q", payload.Message, "something went wrong")
	}
}

func TestMessage_RoundTrip_HandshakeBackwardCompat(t *testing.T) {
	msg := NewHandshakeMsg("Bob")
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	payload, err := DecodeHandshake(got)
	if err != nil {
		t.Fatalf("DecodeHandshake: %v", err)
	}
	if payload.Name != "Bob" {
		t.Errorf("name = %q, want %q", payload.Name, "Bob")
	}
	if payload.PlayerID != "" {
		t.Errorf("player_id = %q, want empty (omitempty)", payload.PlayerID)
	}
}

func TestMessage_RoundTrip_HandshakeWithPlayerID(t *testing.T) {
	msg := newMessage(MsgHandshake, HandshakePayload{Name: "Alice", PlayerID: "player-0"})
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got.Type != MsgHandshake {
		t.Fatalf("type = %q, want %q", got.Type, MsgHandshake)
	}
	payload, err := DecodeHandshake(got)
	if err != nil {
		t.Fatalf("DecodeHandshake: %v", err)
	}
	if payload.Name != "Alice" {
		t.Errorf("name = %q, want %q", payload.Name, "Alice")
	}
	if payload.PlayerID != "player-0" {
		t.Errorf("player_id = %q, want %q", payload.PlayerID, "player-0")
	}
}

func TestMessage_RoundTrip_Chat(t *testing.T) {
	msg := NewChatMsg("player-0", "Alice", "Hello, world!")
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got.Type != MsgChat {
		t.Fatalf("type = %q, want %q", got.Type, MsgChat)
	}
	payload, err := DecodeChat(got)
	if err != nil {
		t.Fatalf("DecodeChat: %v", err)
	}
	if payload.PlayerID != "player-0" {
		t.Errorf("player_id = %q, want %q", payload.PlayerID, "player-0")
	}
	if payload.Name != "Alice" {
		t.Errorf("name = %q, want %q", payload.Name, "Alice")
	}
	if payload.Text != "Hello, world!" {
		t.Errorf("text = %q, want %q", payload.Text, "Hello, world!")
	}
}

func TestMessage_RoundTrip_GameStart(t *testing.T) {
	state := sampleState()
	msg := NewGameStartMsg(state)
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got.Type != MsgGameStart {
		t.Fatalf("type = %q, want %q", got.Type, MsgGameStart)
	}
	payload, err := DecodeState(got)
	if err != nil {
		t.Fatalf("DecodeState: %v", err)
	}
	if payload.State.Round != 1 {
		t.Errorf("round = %d, want 1", payload.State.Round)
	}
}
