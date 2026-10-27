package server

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/itslukv/minecraftserver/internal/protocol"
	"github.com/itslukv/minecraftserver/internal/registry"
	"go.uber.org/zap"
)

type MinecraftServer struct {
	Logger     *zap.SugaredLogger
	registries []registry.Registry
	tags       []registry.TagRegistry

	mu      sync.RWMutex
	players map[uuid.UUID]*Player
	nextEID atomic.Int32

	hmu           sync.RWMutex
	onJoin        []func(*Player) error
	onMove        []func(*Player, PlayerPos) error
	onLeave       []func(*Player)
	onUseItem     []func(*Player) error
	onPlayerInput []func(*Player, []InputType) error
}

func (ms *MinecraftServer) registerPlayer(c *connection, name string, id uuid.UUID) *Player {
	p := &Player{
		EntityID: ms.nextEID.Add(1),
		name:     name,
		uuid:     id,
		conn:     c,
	}
	ms.mu.Lock()
	ms.players[id] = p
	online := len(ms.players)
	ms.mu.Unlock()
	ms.Logger.Infow("player joined", "name", name, "entity", p.EntityID, "online", online)
	return p
}

func (ms *MinecraftServer) unregisterPlayer(p *Player) {
	ms.mu.Lock()
	delete(ms.players, p.uuid)
	online := len(ms.players)
	ms.mu.Unlock()
	ms.Logger.Infow("player left", "name", p.name, "online", online)
}

func (ms *MinecraftServer) Players() []*Player {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	out := make([]*Player, 0, len(ms.players))
	for _, p := range ms.players {
		out = append(out, p)
	}
	return out
}

func NewServer(log *zap.SugaredLogger, regs []registry.Registry, tags []registry.TagRegistry) *MinecraftServer {
	return &MinecraftServer{
		Logger:     log,
		registries: regs,
		tags:       tags,
		players:    map[uuid.UUID]*Player{},
	}
}

func (ms *MinecraftServer) HandleConnection(conn net.Conn) {
	defer conn.Close()
	c := &connection{
		conn:  conn,
		r:     bufio.NewReader(conn),
		state: stateHandshaking,
	}
	defer func() {
		if c.player != nil {
			ms.unregisterPlayer(c.player)
			ms.fireLeave(c.player)
		}
	}()

	_, p, err := c.readPacket()
	if err != nil {
		if err != io.EOF {
			ms.Logger.Errorln("read packet:", err)
		}
		return
	}

	protoVersion, _ := binary.ReadUvarint(p)
	if protoVersion != 776 {
		ms.Logger.Errorf("wrong proto version. Expected: 776. Got: %v", protoVersion)
		return
	}
	address, err := protocol.ReadString(p)
	if err != nil {
		ms.Logger.Errorln("read address:", err)
		return
	}
	var port uint16
	binary.Read(p, binary.BigEndian, &port)
	intent, _ := binary.ReadUvarint(p)

	ms.Logger.Infow("handshake",
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
			ms.Logger.Errorln("handshake error:", err)
			return
		}
		if err := ms.handleConfiguration(c); err != nil {
			ms.Logger.Errorw("configuration failed", "err", err)
			return
		}
		if err := ms.handlePlay(c); err != nil {
			ms.Logger.Errorw("play failed", "err", err)
			return

		}
	case 3: // Transfer
	default:
		return
	}
}
