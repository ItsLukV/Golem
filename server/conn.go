package server

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"sync"

	"github.com/itslukv/minecraftserver/internal/protocol"
)

type connState int

const (
	stateHandshaking connState = iota
	stateStatus
	stateLogin
	stateConfiguration
	statePlay
)

func (state connState) String() string {
	switch state {
	case stateHandshaking:
		return "Handshaking"
	case stateConfiguration:
		return "Configuration"
	case stateLogin:
		return "Login"
	case statePlay:
		return "Play"
	case stateStatus:
		return "Status"
	default:
		panic(fmt.Sprintf("unexpected main.connState: %#v", state))
	}
}

type connection struct {
	conn  net.Conn
	r     *bufio.Reader
	state connState

	// Guards conn against the keep-alive goroutine writing a frame in the
	// middle of one written from the read loop.
	w sync.Mutex

	player *Player
}

func (c *connection) readPacket() (packetID int32, body *bytes.Reader, err error) {
	return protocol.ReadPacket(c.r)
}

func (c *connection) writePacket(p protocol.Packet) error {
	frame := p.Encode()
	c.w.Lock()
	defer c.w.Unlock()
	_, err := c.conn.Write(frame)
	return err
}
