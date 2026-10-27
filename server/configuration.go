package server

import (
	"fmt"

	"github.com/itslukv/minecraftserver/internal/protocol"
	cbConfiguration "github.com/itslukv/minecraftserver/internal/protocol/packetid/configuration/clientbound"
	sbConfiguration "github.com/itslukv/minecraftserver/internal/protocol/packetid/configuration/serverbound"
)

func (ms *MinecraftServer) handleConfiguration(c *connection) error {
	if c.state != stateConfiguration {
		return fmt.Errorf("wrong state, expected configuration, got %s", c.state)
	}
	if err := ms.sendBrand(c); err != nil {
		return fmt.Errorf("send brand: %w", err)
	}
	err := ms.sendKnowPacks(c)
	if err != nil {
		return fmt.Errorf("send know pack: %w", err)
	}

	sentRegistries := false
	for {
		id, r, err := c.readPacket()
		if err != nil {
			return fmt.Errorf("read packet: %w", err)
		}

		switch id {
		case sbConfiguration.ClientInformation, sbConfiguration.CustomPayload, sbConfiguration.KeepAlive:
			// Ignore packet for now
			_ = r
		case sbConfiguration.SelectKnownPacks:
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
			if err := c.writePacket(protocol.Packet{PacketId: cbConfiguration.FinishConfiguration, Data: nil}); err != nil {
				return err

			}
		case sbConfiguration.FinishConfiguration:
			ms.Logger.Infoln("configuration finished")
			c.state = statePlay
			return nil
		default:
			return fmt.Errorf("unexpected configuration packet 0x%02X", id)
		}
	}
}

const serverBrand = "Golem"

func (ms *MinecraftServer) sendBrand(c *connection) error {
	p := protocol.Packet{
		PacketId: cbConfiguration.CustomPayload,
	}
	p.AppendString("minecraft:brand")
	p.AppendString(serverBrand)
	return c.writePacket(p)
}

func (ms *MinecraftServer) sendKnowPacks(c *connection) error {
	p := protocol.Packet{
		PacketId: cbConfiguration.SelectKnownPacks,
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
		p := protocol.Packet{
			PacketId: cbConfiguration.RegistryData,
		}
		p.AppendString(r.ID)
		p.AppendVarint(int32(len(r.Entries)))
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
	p := protocol.Packet{
		PacketId: cbConfiguration.UpdateTags,
	}
	p.AppendVarint(int32(len(ms.tags)))
	for _, r := range ms.tags {
		p.AppendString(r.ID)
		p.AppendVarint(int32(len(r.Tags)))
		for name, ids := range r.Tags {
			p.AppendString(name)
			p.AppendVarint(int32(len(ids)))
			for _, id := range ids {
				p.AppendVarint(int32(id))
			}
		}
	}
	ms.Logger.Infow("sending tags", "registries", len(ms.tags), "bytes", p.Len())
	return c.writePacket(p)
}
