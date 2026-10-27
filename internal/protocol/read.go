package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/google/uuid"
)

func ReadPacket(r *bufio.Reader) (packetID int32, body *bytes.Reader, err error) {
	length, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, nil, err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, nil, err
	}
	p := bytes.NewReader(buf)
	id, err := binary.ReadUvarint(p)
	if err != nil {
		return 0, nil, err
	}
	return int32(id), p, nil
}

func ReadUUID(r io.Reader) (uuid.UUID, error) {
	var id uuid.UUID
	_, err := io.ReadFull(r, id[:])
	return id, err
}

func ReadString(r *bytes.Reader) (string, error) {
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
