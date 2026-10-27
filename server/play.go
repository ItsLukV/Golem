package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/itslukv/minecraftserver/internal/protocol"
	cbPlay "github.com/itslukv/minecraftserver/internal/protocol/packetid/play/clientbound"
	sbPlay "github.com/itslukv/minecraftserver/internal/protocol/packetid/play/serverbound"
)

func (ms *MinecraftServer) handlePlay(c *connection) error {
	if err := ms.sendLogin(c); err != nil {
		return err
	}
	if err := ms.sendPlayerPosition(c); err != nil {
		return err
	}
	if err := ms.readAcceptTeleportation(c); err != nil {
		return err
	}
	if err := ms.sendPlayerInfoUpdate(c); err != nil {
		return err
	}
	if err := ms.sendGameEvent(c); err != nil {
		return err
	}
	if err := ms.sendSetChunkCacheCenter(c); err != nil {
		return err
	}
	if err := ms.sendChunks(c, 0, 0); err != nil {
		return err
	}
	if err := ms.fireJoin(c.player); err != nil {
		return fmt.Errorf("join rejected: %w", err)
	}
	return ms.playLoop(c)
}

func (ms *MinecraftServer) playLoop(c *connection) error {
	stop := make(chan struct{})
	defer close(stop)
	go ms.sendKeepAlives(c, stop)

	for {
		id, body, err := c.readPacket()
		if err != nil {
			if err == io.EOF {
				ms.Logger.Infow("client disconnected", "name", c.player.name)
				return nil
			}
			return fmt.Errorf("read packet: %w", err)
		}

		switch id {
		case sbPlay.KeepAlive:
			var token int64
			if err := binary.Read(body, binary.BigEndian, &token); err != nil {
				return fmt.Errorf("keep alive: %w", err)
			}
		case sbPlay.AcceptTeleportation:
			// Confirms a Teleport
			continue
		case sbPlay.MovePlayerPos, sbPlay.MovePlayerPosRot,
			sbPlay.MovePlayerRot, sbPlay.MovePlayerStatusOnly:
			pos := c.player.Pos()
			hasPos := id == sbPlay.MovePlayerPos || id == sbPlay.MovePlayerPosRot
			hasRot := id == sbPlay.MovePlayerPosRot || id == sbPlay.MovePlayerRot
			if err := readMovement(body, &pos, hasPos, hasRot); err != nil {
				ms.Logger.Errorw("movement",
					"id", fmt.Sprintf("0x%02X", id), "err", err)
				continue
			}
			prev := c.player.Pos()
			c.player.setPos(pos)
			if err := ms.fireMove(c.player, prev); err != nil {
				ms.Logger.Debugw("move rejected", "name", c.player.name, "err", err)
				if err := c.player.Teleport(prev); err != nil {
					return fmt.Errorf("rubber-band: %w", err)
				}
			}
		case sbPlay.PlayerInput:
			flags_raw, err := body.ReadByte()
			if err != nil {
				return err
			}
			flags := InputType(flags_raw)
			ms.fireOnPlayerInput(c.player, flags.Flags())
			continue
		case sbPlay.ClientTickEnd:
			continue
		case sbPlay.UseItemOn, sbPlay.UseItem:
			if err := ms.fireOnUseItem(c.player); err != nil {
				ms.Logger.Debugw("failed to use item", "name", c.player.name, "err", err)
			}
		default:
			ms.Logger.Debugw("unhandled play packet", "id", fmt.Sprintf("0x%02X", id))
		}
	}
}

func readMovement(body *bytes.Reader, pos *PlayerPos, hasPos, hasRot bool) error {
	if hasPos {
		for _, f := range []*float64{&pos.X, &pos.Y, &pos.Z} {
			if err := binary.Read(body, binary.BigEndian, f); err != nil {
				return err
			}
		}
	}
	if hasRot {
		for _, f := range []*float32{&pos.Yaw, &pos.Pitch} {
			if err := binary.Read(body, binary.BigEndian, f); err != nil {
				return err
			}
		}
	}
	var flags byte
	if err := binary.Read(body, binary.BigEndian, &flags); err != nil {
		return err
	}
	pos.OnGround = flags&0x01 != 0
	return nil
}

func (ms *MinecraftServer) sendKeepAlives(c *connection, stop <-chan struct{}) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			p := protocol.Packet{PacketId: cbPlay.KeepAlive}
			p.AppendLong(time.Now().UnixMilli())
			if err := c.writePacket(p); err != nil {
				ms.Logger.Errorw("keep alive failed", "err", err)
				return
			}
		}
	}
}

func (ms *MinecraftServer) sendGameEvent(c *connection) error {
	p := protocol.Packet{
		PacketId: cbPlay.GameEvent,
	}

	p.AppendByte(13)
	p.AppendFloat(13)
	return c.writePacket(p)
}

func (ms *MinecraftServer) sendSetChunkCacheCenter(c *connection) error {
	p := protocol.Packet{
		PacketId: cbPlay.SetChunkCacheCenter,
	}

	p.AppendVarint(0)
	p.AppendVarint(0)
	return c.writePacket(p)
}

func (ms *MinecraftServer) sendLogin(c *connection) error {
	p := protocol.Packet{
		PacketId: cbPlay.Login,
	}
	p.AppendInt(c.player.EntityID)                       // Entity ID
	p.AppendBool(false)                                  // Is hardcore
	p.AppendStringArray([]string{"minecraft:overworld"}) // Dimension Names
	p.AppendVarint(1)                                    // Max Players (unused)
	p.AppendVarint(viewDistance)                         // View Distance
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
	ms.Logger.Infow("sending play login", "bytes", p.Len())
	return c.writePacket(p)
}

func (ms *MinecraftServer) sendPlayerPosition(c *connection) error {
	p := protocol.Packet{PacketId: cbPlay.PlayerPosition}
	p.AppendVarint(0) // Teleport ID
	p.AppendDouble(0) // X
	p.AppendDouble(0) // Y
	p.AppendDouble(0) // Z
	p.AppendDouble(0) // Velocity X
	p.AppendDouble(0) // Velocity Y
	p.AppendDouble(0) // Velocity Z
	p.AppendFloat(0)  // Yaw
	p.AppendFloat(0)  // Pitch
	p.AppendInt(0)    // Relatives (teleport flags bitfield)
	return c.writePacket(p)
}

func (ms *MinecraftServer) readAcceptTeleportation(c *connection) error {
	for {
		packetID, body, err := c.readPacket()
		if err != nil {
			return fmt.Errorf("read packet: %w", err)
		}
		if packetID != sbPlay.AcceptTeleportation {
			ms.Logger.Debugw("waiting for teleport confirm, skipping packet",
				"id", fmt.Sprintf("0x%02X", packetID))
			continue
		}

		TeleportID, err := binary.ReadUvarint(body)
		if err != nil {
			return err
		}
		if TeleportID != 0 {
			return fmt.Errorf("wrong teleport id expected 0 and got %v", TeleportID)
		}
		return nil
		// for 26.3:
		// var X float64
		// if err := binary.Read(body, binary.BigEndian, &X); err != nil {
		// 	return err
		// }
		// var Y float64
		// if err := binary.Read(body, binary.BigEndian, &Y); err != nil {
		// 	return err
		// }
		// var Z float64
		// if err := binary.Read(body, binary.BigEndian, &Z); err != nil {
		// 	return err
		// }
		// var yaw float32
		// if err := binary.Read(body, binary.BigEndian, &yaw); err != nil {
		// 	return err
		// }
		// var pitch float32
		// if err := binary.Read(body, binary.BigEndian, &pitch); err != nil {
		// 	return err
		// }
		// return nil
	}
}

type PlayerInfoAction byte

const (
	AddPlayer         PlayerInfoAction = 1 << iota // 0x01
	InitializeChat                                 // 0x02
	UpdateGameMode                                 // 0x04
	UpdateListed                                   // 0x08
	UpdateLatency                                  // 0x10
	UpdateDisplayName                              // 0x20
	UpdateListOrder                                // 0x40
	UpdateHat                                      // 0x80
)

func (ms *MinecraftServer) sendPlayerInfoUpdate(c *connection) error {
	actions := AddPlayer | UpdateGameMode | UpdateListed | UpdateLatency

	p := protocol.Packet{PacketId: cbPlay.PlayerInfoUpdate}
	p.AppendByte(byte(actions))
	p.AppendVarint(1) // Number of entries

	p.AppendUUID(c.player.uuid)   // Profile ID
	p.AppendString(c.player.name) // AddPlayer: name
	p.AppendVarint(0)             // AddPlayer: signed profile properties (none)
	p.AppendVarint(0)             // UpdateGameMode: survival
	p.AppendBool(true)            // UpdateListed: show in the tab list
	p.AppendVarint(0)             // UpdateLatency: ping in ms

	ms.Logger.Infow("sending player info update",
		"name", c.player.name,
		"actions", fmt.Sprintf("0x%02X", byte(actions)),
		"bytes", p.Len(),
	)
	return c.writePacket(p)
}
