package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/google/uuid"
)

func readUUID(r io.Reader) (uuid.UUID, error) {
	var id uuid.UUID
	_, err := io.ReadFull(r, id[:])
	return id, err
}

func readString(r *bytes.Reader) (string, error) {
	length, err := binary.ReadUvarint(r)
	if err != nil {
		return "", err
	}
	if length > 32767 {
		return "", fmt.Errorf("string too long: %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// Deprecated
func (c *connection) writePacket2(packetID int, payload []byte) error {
	body := append(binary.AppendUvarint(nil, uint64(packetID)), payload...)
	frame := append(binary.AppendUvarint(nil, uint64(len(body))), body...)
	_, err := c.conn.Write(frame)
	return err
}

func (c *connection) writePacket(p Packet) error {
	body := append(binary.AppendUvarint(nil, uint64(p.PacketId)), p.Data...)
	frame := append(binary.AppendUvarint(nil, uint64(len(body))), body...)
	_, err := c.conn.Write(frame)
	return err
}

type Packet struct {
	PacketId int
	Data     []byte
}

func (p *Packet) AppendString(s string) {
	p.AppendVarint(len(s))
	p.Data = append(p.Data, s...)
}

func (p *Packet) AppendVarint(i int) {
	p.Data = binary.AppendUvarint(p.Data, uint64(i))
}

func (p *Packet) AppendInt(i int) {
	p.Data = binary.BigEndian.AppendUint32(p.Data, uint32(i))
}

func (p *Packet) AppendLong(l int64) {
	p.Data = binary.BigEndian.AppendUint64(p.Data, uint64(l))
}

func (p *Packet) AppendArray(arr []byte) {
	p.AppendVarint(len(arr))
	p.Data = append(p.Data, arr...)
}

func (p *Packet) Len() int {
	return len(p.Data)
}

func (p *Packet) AppendUUID(id uuid.UUID) {
	p.Data = append(p.Data, id[:]...)
}

func (p Packet) AppendBool(b bool) {
	if b {
		p.Data = append(p.Data, 1)
	} else {
		p.Data = append(p.Data, 0)
	}
}
