package server

import (
	"github.com/google/uuid"
	cbPlay "github.com/itslukv/minecraftserver/internal/protocol/packetid/play/clientbound"
)

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

func (p *Player) Teleport(pos PlayerPos) error {
	pk := Packet{PacketId: cbPlay.PlayerPosition}
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
