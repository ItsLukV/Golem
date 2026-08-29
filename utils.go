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

func appendUUID(b []byte, id uuid.UUID) []byte {
	return append(b, id[:]...)
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

func (c *connection) writePacket(packetID int, payload []byte) error {
	body := append(binary.AppendUvarint(nil, uint64(packetID)), payload...)
	frame := append(binary.AppendUvarint(nil, uint64(len(body))), body...)
	_, err := c.conn.Write(frame)
	return err
}

func appendString(b []byte, s string) []byte {
	b = binary.AppendUvarint(b, uint64(len(s)))
	return append(b, s...)
}
