package network

import (
	"fmt"
	"reflect"
)

type MockConnection struct {
	EncodedMessages []interface{}
	DecodedMessages []interface{}
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
	if len(m.DecodedMessages) == 0 {
		return fmt.Errorf("no more messages to decode")
	}
	reflect.ValueOf(e).Elem().Set(reflect.ValueOf(m.DecodedMessages[0]).Elem())
	m.DecodedMessages = m.DecodedMessages[1:]
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
