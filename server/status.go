package server

import (
	"encoding/binary"
	"fmt"

	"github.com/itslukv/minecraftserver/internal/protocol"
	cbStatus "github.com/itslukv/minecraftserver/internal/protocol/packetid/status/clientbound"
	sbStatus "github.com/itslukv/minecraftserver/internal/protocol/packetid/status/serverbound"
)

func (ms *MinecraftServer) handleStatus(c *connection) {
	if c.state != stateStatus {
		ms.Logger.Errorw("Wrong state, expected status", "state", c.state)
		return
	}
	for {
		packetID, p, err := c.readPacket()
		if err != nil {
			return
		}
		switch packetID {

		case sbStatus.StatusRequest:
			status := fmt.Sprintf(`{
				"version": {"name": "26.2", "protocol": 776},
				"players": {"max": %v, "online": %v},
				"description": {"text": "Hello from Go!"}
			}`, len(ms.players)+1, len(ms.players))
			packet := protocol.Packet{
				PacketId: cbStatus.StatusResponse,
			}
			packet.AppendString(status)
			if err := c.writePacket(packet); err != nil {
				return
			}

		case sbStatus.PingRequest:
			var nonce int64
			if err := binary.Read(p, binary.BigEndian, &nonce); err != nil {
				return
			}
			packet := protocol.Packet{
				PacketId: cbStatus.PongResponse,
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
