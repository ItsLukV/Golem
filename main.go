package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/google/uuid"
	reg "github.com/itslukv/minecraftserver/registries"
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

	registries, err := reg.Load()
	if err != nil {
		log.Fatal(err)
	}
	logger.Infoln("Server listening on :25565")
	ms := MinecraftServer{
		logger:     logger,
		registries: registries,
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
	logger     *zap.SugaredLogger
	registries []reg.Registry
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
	if protoVersion != 776 {
		ms.logger.Errorf("wrong proto version. Expected: 776. Got: %v", protoVersion)
		return
	}
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
		if err := ms.handleLogin(c); err != nil {
			ms.logger.Errorln("handshake error:", err)
			return
		}
		if err := ms.handleConfiguration(c); err != nil {
			ms.logger.Errorw("configuration failed", "err", err)
			return
		}
	case 3: // Transfer
	default:
		return
	}
}

func (ms *MinecraftServer) handleLogin(c *connection) error {
	if c.state != stateLogin {
		return fmt.Errorf("wrong state, expected login, got: %s", c.state)
	}

	name, uuid, err := ms.parseLoginPacket(c)
	if err != nil {
		return err
	}
	ms.logger.Infow("login", "name", name, "uuid", uuid)

	err = ms.sendLoginSuccessPacket(c, *name, *uuid)
	if err != nil {
		return err
	}

	ackID, _, err := c.readPacket()
	if err != nil {
		return fmt.Errorf("read ackID: %v", err)
	}
	if ackID != 0x03 {
		return fmt.Errorf("expected login ack, got: %v", ackID)
	}
	ms.logger.Infow("Login ack", "name", name)
	c.state = stateConfiguration
	return nil
}

func (ms *MinecraftServer) parseLoginPacket(c *connection) (*string, *uuid.UUID, error) {
	_, body, err := c.readPacket()
	if err != nil {
		ms.logger.Errorw("read packet:", err)
		return nil, nil, err
	}
	name, err := readString(body)
	if err != nil {
		ms.logger.Errorln("read name:", err)
		return nil, nil, err
	}
	uuid, err := readUUID(body)
	if err != nil {
		ms.logger.Errorln("read uuid:", err)
		return nil, nil, err
	}
	return &name, &uuid, nil
}

func (ms *MinecraftServer) sendLoginSuccessPacket(c *connection, username string, uuid uuid.UUID) error {
	var loginSuccess []byte
	loginSuccess = appendUUID(loginSuccess, uuid)
	loginSuccess = appendString(loginSuccess, username)
	loginSuccess = binary.AppendUvarint(loginSuccess, 0)
	loginSuccess = appendUUID(loginSuccess, uuid)
	return c.writePacket(0x2, loginSuccess)
}

func (ms *MinecraftServer) handleConfiguration(c *connection) error {
	if c.state != stateConfiguration {
		return fmt.Errorf("wrong state, expected configuration, got %s", c.state)
	}
	err := ms.sendKnowPacks(c)
	if err != nil {
		return fmt.Errorf("send know pack: %w", err)
	}

	const (
		sbClientInfo = 0x00
		sbPluginMsg  = 0x02
		sbAckFinish  = 0x03
		sbKeepAlive  = 0x04
		sbKnownPacks = 0x07
	)

	sentRegistries := false
	for {
		id, r, err := c.readPacket()
		if err != nil {
			return fmt.Errorf("read packet: %w", err)
		}
		ms.logger.Debugw("config packet", "id", fmt.Sprintf("0x%02X", id))

		switch id {
		case sbClientInfo, sbPluginMsg, sbKeepAlive:
			// Ignore packet for now
			_ = r
		case sbKnownPacks:
			if sentRegistries {
				return fmt.Errorf("client sent Known Packs twice")
			}
			sentRegistries = true
			err := ms.sendRegistries(c)
			if err != nil {
				return err
			}
			if err := c.writePacket(0x0D, binary.AppendUvarint(nil, 0)); err != nil {
				return err
			}
			if err := c.writePacket(0x03, nil); err != nil {
				return err
			}
		case sbAckFinish:
			ms.logger.Infoln("configuration finished")
			c.state = statePlay
			return nil
		default:
			return fmt.Errorf("unexpected configuration packet 0x%02X", id)
		}
	}
}

func (ms *MinecraftServer) sendKnowPacks(c *connection) error {
	var b []byte
	b = binary.AppendUvarint(b, 1)
	b = appendString(b, "minecraft")
	b = appendString(b, "core")
	b = appendString(b, "26.2")
	if err := c.writePacket(0x0E, b); err != nil {
		return fmt.Errorf("failed to send packet")
	}
	return nil
}

func (ms *MinecraftServer) sendRegistries(c *connection) error {
	for _, r := range ms.registries {
		var p []byte
		p = appendString(p, r.ID)
		p = binary.AppendUvarint(p, uint64(len(r.Entries)))
		for _, e := range r.Entries {
			p = appendString(p, e.ID)
			p = append(p, 0)
		}
		if err := c.writePacket(0x07, p); err != nil {
			return fmt.Errorf("registry %s: %w", r.ID, err)
		}
	}
	return nil
}

func (ms *MinecraftServer) handleStatus(c *connection) {
	if c.state != stateStatus {
		ms.logger.Errorw("Wrong state, expected status", "state", c.state)
		return
	}
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
			if err := c.writePacket(0x00, payload); err != nil {
				return
			}

		case 0x01:
			var nonce int64
			if err := binary.Read(p, binary.BigEndian, &nonce); err != nil {
				return
			}
			payload := make([]byte, 8)
			binary.BigEndian.PutUint64(payload, uint64(nonce))
			if err := c.writePacket(0x01, payload); err != nil {
				return
			}

			return

		default:
			return
		}
	}
}
