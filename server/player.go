package server

import (
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/itslukv/minecraftserver/internal/protocol"
	cbPlay "github.com/itslukv/minecraftserver/internal/protocol/packetid/play/clientbound"
)

type Player struct {
	EntityID int32
	name     string
	uuid     uuid.UUID

	mu  sync.RWMutex
	pos PlayerPos

	conn         *connection
	nextTeleport atomic.Int32
}

type PlayerPos struct {
	X, Y, Z    float64
	Yaw, Pitch float32
	OnGround   bool
}

func (p *Player) Name() string { return p.name }

func (p *Player) UUID() uuid.UUID { return p.uuid }

func (p *Player) Pos() PlayerPos {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pos
}

func (p *Player) setPos(pos PlayerPos) {
	p.mu.Lock()
	p.pos = pos
	p.mu.Unlock()
}

type TeleportFlags int32

const (
	RelX           TeleportFlags = 1 << iota // 0x0001
	RelY                                     // 0x0002
	RelZ                                     // 0x0004
	RelYaw                                   // 0x0008
	RelPitch                                 // 0x0010
	RelVelX                                  // 0x0020
	RelVelY                                  // 0x0040
	RelVelZ                                  // 0x0080
	RotateVelocity                           // 0x0100
)

func (p *Player) Teleport(pos PlayerPos) error {
	pk := protocol.Packet{PacketId: cbPlay.PlayerPosition}
	pk.AppendVarint(p.nextTeleport.Add(1)) // Teleport ID
	pk.AppendDouble(pos.X)
	pk.AppendDouble(pos.Y)
	pk.AppendDouble(pos.Z)
	pk.AppendDouble(0) // Velocity X
	pk.AppendDouble(0) // Velocity Y
	pk.AppendDouble(0) // Velocity Z
	pk.AppendFloat(pos.Yaw)
	pk.AppendFloat(pos.Pitch)
	pk.AppendInt(int32(RelVelX | RelVelY | RelVelZ)) // Relatives: all absolute

	p.setPos(pos)
	return p.conn.writePacket(pk)
}

func (p *Player) Disconnect() error {
	return p.conn.conn.Close()
}
