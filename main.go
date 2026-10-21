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

	registries, err := reg.LoadRegistries()
	if err != nil {
		logger.Fatal(err)
	}

	tags, err := reg.LoadTags(registries)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Infoln("Server listening on :25565")
	ms := MinecraftServer{
		logger:     logger,
		registries: registries,
		tags:       tags,
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
	tags       []reg.TagRegistry
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
		if err := ms.handlePlay(c); err != nil {
			ms.logger.Errorw("play failed", "err", err)
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
	p := Packet{
		PacketId: 0x2,
	}
	p.AppendUUID(uuid)
	p.AppendString(username)
	p.AppendVarint(0)
	p.AppendUUID(uuid)
	return c.writePacket(p)
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
			err = ms.sendTags(c)
			if err != nil {
				return err
			}
			if err := c.writePacket(Packet{PacketId: 0x03, Data: nil}); err != nil {
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
	p := Packet{
		PacketId: 0x0E,
	}
	p.AppendVarint(1)
	p.AppendString("minecraft")
	p.AppendString("core")
	p.AppendString("26.2")
	if err := c.writePacket(p); err != nil {
		return fmt.Errorf("failed to send packet")
	}
	return nil
}

func (ms *MinecraftServer) sendRegistries(c *connection) error {
	for _, r := range ms.registries {
		p := Packet{
			PacketId: 0x07,
		}
		p.AppendString(r.ID)
		p.AppendVarint(len(r.Entries))
		for _, e := range r.Entries {
			p.AppendString(e.ID)
			p.Data = append(p.Data, 0)
		}
		if err := c.writePacket(p); err != nil {
			return fmt.Errorf("registry %s: %w", r.ID, err)
		}
	}
	return nil
}

func (ms *MinecraftServer) sendTags(c *connection) error {
	p := Packet{
		PacketId: 0x0D,
	}
	p.AppendVarint(len(ms.tags))
	for _, r := range ms.tags {
		p.AppendString(r.ID)
		p.AppendVarint(len(r.Tags))
		for name, ids := range r.Tags {
			p.AppendString(name)
			p.AppendVarint(len(ids))
			for _, id := range ids {
				p.AppendVarint(id)
			}
		}
	}
	ms.logger.Infow("sending tags", "registries", len(ms.tags), "bytes", p.Len())
	return c.writePacket(p)
}

func (ms *MinecraftServer) handlePlay(c *connection) error {
	p := Packet{
		PacketId: 0x31,
	}
	p.AppendInt(1)                                       // Entity ID
	p.AppendBool(false)                                  // Is hardcore
	p.AppendStringArray([]string{"minecraft:overworld"}) // Dimension Names
	p.AppendVarint(1)                                    // Max Players (unused)
	p.AppendVarint(8)                                    // View Distance
	p.AppendVarint(8)                                    // Simulation Distance
	p.AppendBool(false)                                  // Reduced Debug Info
	p.AppendBool(true)                                   // Enable respawn screen
	p.AppendBool(false)                                  // Do limited crafting (unused)
	p.AppendVarint(0)                                    // Dimension Type (index into minecraft:dimension_type => overworld)
	p.AppendString("minecraft:overworld")                // Dimension Name
	p.AppendLong(0)                                      // Hashed seed
	p.AppendByte(0)                                      // Game mode (unsigned byte)
	p.AppendByte(0)                                      // Previous Game mode (byte)
	p.AppendBool(false)                                  // Is Debug
	p.AppendBool(false)                                  // Is Flat
	p.AppendBool(false)                                  // Has death location
	p.AppendVarint(0)                                    // Portal cooldown
	p.AppendVarint(50)                                   // Sea level
	p.AppendBool(false)                                  // Online mode
	p.AppendBool(false)                                  // Enforces Secure Chat
	ms.logger.Infow("sending play login", "bytes", p.Len())
	return c.writePacket(p)
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
			packet := Packet{
				PacketId: 0x00,
			}
			packet.AppendString(status)
			if err := c.writePacket(packet); err != nil {
				return
			}

		case 0x01:
			var nonce int64
			if err := binary.Read(p, binary.BigEndian, &nonce); err != nil {
				return
			}
			packet := Packet{
				PacketId: 0x01,
			}
			packet.AppendLong(nonce)
			if err := c.writePacket(packet); err != nil {
				return
			}

			return

		default:
			return
		}
	}
}
