package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger() *zap.SugaredLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stdout"}
	logger, _ := cfg.Build()
	return logger.Sugar()
}

func main() {
	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	logger := newLogger()

	logger.Infoln("Server listening on :25565")
	ms := MinecraftServer{
		logger: logger,
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Infoln("accept error:", err)
			continue
		}

		go ms.handleConnection(conn)
	}
}

type connState int

const (
	stateHandshaking connState = iota
	stateStatus
	stateLogin
	stateConfiguration
	statePlay
)

type connection struct {
	conn  net.Conn
	r     *bufio.Reader
	state connState
}

func (c *connection) readPacket() (packetID uint64, body *bytes.Reader, err error) {
	length, err := binary.ReadUvarint(c.r)
	if err != nil {
		return 0, nil, err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		return 0, nil, err
	}
	p := bytes.NewReader(buf)
	id, err := binary.ReadUvarint(p)
	if err != nil {
		return 0, nil, err
	}
	return id, p, nil
}

type MinecraftServer struct {
	logger *zap.SugaredLogger
}

func (ms *MinecraftServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	c := &connection{
		conn:  conn,
		r:     bufio.NewReader(conn),
		state: stateHandshaking,
	}

	_, p, err := c.readPacket()
	if err != nil {
		if err != io.EOF {
			ms.logger.Errorln("read packet:", err)
		}
		return
	}

	protoVersion, _ := binary.ReadUvarint(p)
	address, err := readString(p)
	if err != nil {
		ms.logger.Errorln("read address:", err)
		return
	}
	var port uint16
	binary.Read(p, binary.BigEndian, &port)
	intent, _ := binary.ReadUvarint(p)

	ms.logger.Infow("handshake",
		"proto", protoVersion,
		"addr", address,
		"port", port,
		"intent", intent,
	)

	switch intent {
	case 1: // Status
		c.state = stateStatus
		ms.handleStatus(c)
	case 2: // Login
		c.state = stateLogin
		ms.handleLogin(c)
	case 3: // Transfer
	default:
		return
	}
}

func (ms *MinecraftServer) handleLogin(c *connection) {

}

func (ms *MinecraftServer) handleStatus(c *connection) {
	for {
		packetID, p, err := c.readPacket()
		if err != nil {
			return
		}
		switch packetID {

		case 0x00:
			status := `{
				"version": {"name": "26.2", "protocol": 776},
				"players": {"max": -1, "online": 0},
				"description": {"text": "Hello from Go!"}
			}`
			payload := appendString(nil, status)
			if err := writePacket(c.conn, 0x00, payload); err != nil {
				return
			}

		case 0x01:
			var nonce int64
			if err := binary.Read(p, binary.BigEndian, &nonce); err != nil {
				return
			}
			payload := make([]byte, 8)
			binary.BigEndian.PutUint64(payload, uint64(nonce))
			if err := writePacket(c.conn, 0x01, payload); err != nil {
				return
			}

			return

		default:
			return
		}
	}
}
