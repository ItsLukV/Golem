package protocol

import (
	"encoding/binary"
	"math"

	"github.com/google/uuid"
)

type Packet struct {
	PacketId int32
	Data     []byte
}

func (p *Packet) Encode() []byte {
	body := append(binary.AppendUvarint(nil, uint64(p.PacketId)), p.Data...)
	return append(binary.AppendUvarint(nil, uint64(len(body))), body...)
}

func (p *Packet) AppendString(s string) {
	p.AppendVarint(int32(len(s)))
	p.Data = append(p.Data, s...)
}

func (p *Packet) AppendVarint(i int32) {
	p.Data = binary.AppendUvarint(p.Data, uint64(i))
}

func (p *Packet) AppendInt(i int32) {
	p.Data = binary.BigEndian.AppendUint32(p.Data, uint32(i))
}

func (p *Packet) AppendLong(l int64) {
	p.Data = binary.BigEndian.AppendUint64(p.Data, uint64(l))
}

func (p *Packet) AppendPrefixedArray(arr []byte) {
	p.AppendVarint(int32(len(arr)))
	p.Data = append(p.Data, arr...)
}

type ToByte interface {
	ToByte() []byte
}

func (p *Packet) Append(v ToByte) {
	p.Data = append(p.Data, v.ToByte()...)
}

type BitSet []int64

func (b BitSet) ToByte() []byte {
	out := binary.AppendUvarint(nil, uint64(len(b)))
	for _, l := range b {
		out = binary.BigEndian.AppendUint64(out, uint64(l))
	}
	return out
}

func AppendArray[T ToByte](p *Packet, arr []T) {
	for _, v := range arr {
		p.Data = append(p.Data, v.ToByte()...)
	}
}

func (p *Packet) Len() int {
	return len(p.Data)
}

func (p *Packet) AppendUUID(id uuid.UUID) {
	p.Data = append(p.Data, id[:]...)
}

func (p *Packet) AppendBool(b bool) {
	if b {
		p.Data = append(p.Data, 1)
	} else {
		p.Data = append(p.Data, 0)
	}
}

func (p *Packet) AppendByte(b byte) {
	p.Data = append(p.Data, b)
}

func (p *Packet) AppendStringArray(items []string) {
	p.AppendVarint(int32(len(items)))
	for _, s := range items {
		p.AppendString(s)
	}
}

func (p *Packet) AppendDouble(f float64) {
	p.Data = binary.BigEndian.AppendUint64(p.Data, math.Float64bits(f))
}

func (p *Packet) AppendFloat(f float32) {
	p.Data = binary.BigEndian.AppendUint32(p.Data, math.Float32bits(f))
}
