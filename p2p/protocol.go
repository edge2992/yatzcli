package p2p

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/edge2992/yatzcli/engine"
)

const (
	MsgHandshake   = "handshake"
	MsgGameStart   = "game_start"
	MsgTurnStart   = "turn_start"
	MsgAction      = "action"
	MsgStateUpdate = "state_update"
	MsgGameOver    = "game_over"
	MsgError       = "error"
)

const (
	ActionRoll  = "roll"
	ActionHold  = "hold"
	ActionScore = "score"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type HandshakePayload struct {
	Name string `json:"name"`
}

type ActionPayload struct {
	Action   string `json:"action"`
	Indices  []int  `json:"indices,omitempty"`
	Category string `json:"category,omitempty"`
}

type StatePayload struct {
	State engine.GameState `json:"state"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

// WriteMessage writes a length-prefixed (uint32 big-endian) JSON message to w.
func WriteMessage(w io.Writer, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return fmt.Errorf("write length: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	return nil
}

// ReadMessage reads a length-prefixed JSON message from r.
func ReadMessage(r io.Reader) (*Message, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}
	return &msg, nil
}

func newMessage(msgType string, payload any) *Message {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("marshal payload: %v", err))
	}
	return &Message{
		Type:    msgType,
		Payload: json.RawMessage(data),
	}
}

func NewHandshakeMsg(name string) *Message {
	return newMessage(MsgHandshake, HandshakePayload{Name: name})
}

func NewActionMsg(ap ActionPayload) *Message {
	return newMessage(MsgAction, ap)
}

func NewStateUpdateMsg(state engine.GameState) *Message {
	return newMessage(MsgStateUpdate, StatePayload{State: state})
}

func NewGameStartMsg(state engine.GameState) *Message {
	return newMessage(MsgGameStart, StatePayload{State: state})
}

func NewTurnStartMsg(state engine.GameState) *Message {
	return newMessage(MsgTurnStart, StatePayload{State: state})
}

func NewGameOverMsg(state engine.GameState) *Message {
	return newMessage(MsgGameOver, StatePayload{State: state})
}

func NewErrorMsg(errMsg string) *Message {
	return newMessage(MsgError, ErrorPayload{Message: errMsg})
}

func DecodeHandshake(msg *Message) (*HandshakePayload, error) {
	var p HandshakePayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode handshake: %w", err)
	}
	return &p, nil
}

func DecodeAction(msg *Message) (*ActionPayload, error) {
	var p ActionPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode action: %w", err)
	}
	return &p, nil
}

func DecodeState(msg *Message) (*StatePayload, error) {
	var p StatePayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	return &p, nil
}

func DecodeError(msg *Message) (*ErrorPayload, error) {
	var p ErrorPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return &p, nil
}
