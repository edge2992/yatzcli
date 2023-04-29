package network

import (
	"errors"
	"yatzcli/messages"
)

type MockConnection struct {
	EncodedMessages []interface{}
	DecodedMessages []interface{}
	decodeIndex     int
}

func NewMockConnection() *MockConnection {
	return &MockConnection{
		EncodedMessages: make([]interface{}, 0),
		DecodedMessages: make([]interface{}, 0),
	}
}

func (m *MockConnection) Encode(e interface{}) error {
	m.EncodedMessages = append(m.EncodedMessages, e)
	return nil
}

func (m *MockConnection) Decode(e interface{}) error {
	if m.decodeIndex >= len(m.DecodedMessages) {
		return errors.New("no more messages to decode")
	}
	msg := m.DecodedMessages[m.decodeIndex]
	m.decodeIndex++
	switch v := e.(type) {
	case *messages.Message:
		*v = *msg.(*messages.Message)
	default:
		return errors.New("unsupported type")
	}
	return nil
}

func (m *MockConnection) Close() error {
	return nil
}

func (m *MockConnection) PushMessage(e interface{}) {
	m.DecodedMessages = append(m.DecodedMessages, e)
}

func (m *MockConnection) TopEncodedMessage() interface{} {
	if len(m.EncodedMessages) == 0 {
		return nil
	}
	return m.EncodedMessages[len(m.EncodedMessages)-1]
}
