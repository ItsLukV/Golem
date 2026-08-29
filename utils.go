package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

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

func writePacket(w io.Writer, packetID int, payload []byte) error {
	body := append(binary.AppendUvarint(nil, uint64(packetID)), payload...)
	frame := append(binary.AppendUvarint(nil, uint64(len(body))), body...)
	_, err := w.Write(frame)
	return err
}

func appendString(b []byte, s string) []byte {
	b = binary.AppendUvarint(b, uint64(len(s)))
	return append(b, s...)
}
