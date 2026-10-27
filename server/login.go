package server

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/itslukv/minecraftserver/internal/protocol"
	cbLogin "github.com/itslukv/minecraftserver/internal/protocol/packetid/login/clientbound"
	sbLogin "github.com/itslukv/minecraftserver/internal/protocol/packetid/login/serverbound"
)

func (ms *MinecraftServer) handleLogin(c *connection) error {
	if c.state != stateLogin {
		return fmt.Errorf("wrong state, expected login, got: %s", c.state)
	}

	name, uuid, err := ms.parseLoginPacket(c)
	if err != nil {
		return err
	}
	ms.Logger.Infow("login", "name", name, "uuid", uuid)
	c.player = ms.registerPlayer(c, *name, *uuid)

	err = ms.sendLoginSuccessPacket(c, *name, *uuid)
	if err != nil {
		return err
	}

	ackID, _, err := c.readPacket()
	if err != nil {
		return fmt.Errorf("read ackID: %v", err)
	}

	if ackID != sbLogin.LoginAcknowledged {
		return fmt.Errorf("expected login ack, got: %v", ackID)
	}
	ms.Logger.Infow("Login ack", "name", name)
	c.state = stateConfiguration
	return nil
}

func (ms *MinecraftServer) parseLoginPacket(c *connection) (*string, *uuid.UUID, error) {
	_, body, err := c.readPacket()
	if err != nil {
		ms.Logger.Errorw("read packet:", err)
		return nil, nil, err
	}
	name, err := protocol.ReadString(body)
	if err != nil {
		ms.Logger.Errorln("read name:", err)
		return nil, nil, err
	}
	uuid, err := protocol.ReadUUID(body)
	if err != nil {
		ms.Logger.Errorln("read uuid:", err)
		return nil, nil, err
	}
	return &name, &uuid, nil
}

func (ms *MinecraftServer) sendLoginSuccessPacket(c *connection, username string, uuid uuid.UUID) error {
	p := protocol.Packet{
		PacketId: cbLogin.LoginFinished,
	}
	p.AppendUUID(uuid)
	p.AppendString(username)
	p.AppendVarint(0)
	p.AppendUUID(uuid)
	return c.writePacket(p)
}
