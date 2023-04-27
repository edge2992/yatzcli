package messages

import (
	"encoding/gob"
	"net"
)

type Connection interface {
	Encode(interface{}) error
	Decode(interface{}) error
	Close() error
}

type GobConnection struct {
	encoder *gob.Encoder
	decoder *gob.Decoder
	conn    net.Conn
}

func NewGobConnection(conn net.Conn) *GobConnection {
	return &GobConnection{
		conn:    conn,
		encoder: gob.NewEncoder(conn),
		decoder: gob.NewDecoder(conn),
	}
}

func (g *GobConnection) Encode(e interface{}) error {
	return g.encoder.Encode(e)
}

func (g *GobConnection) Decode(e interface{}) error {
	return g.decoder.Decode(e)
}

func (g *GobConnection) Close() error {
	return g.conn.Close()
}
