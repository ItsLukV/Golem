package world

import (
	"bytes"
	"encoding/binary"

	"github.com/itslukv/minecraftserver/internal/protocol"
)

const (
	Sections          = 384 / 16
	lightSections     = Sections + 2
	lightSectionBytes = 16 * 16 * 16 / 2
)

type LightData struct {
	SkyLightMask        protocol.BitSet
	BlockLightMask      protocol.BitSet
	EmptySkyLightMask   protocol.BitSet
	EmptyBlockLightMask protocol.BitSet
	SkyLight            [][]byte
	BlockLight          [][]byte
}

func (ld LightData) ToByte() []byte {
	out := []byte{}
	out = append(out, ld.SkyLightMask.ToByte()...)
	out = append(out, ld.BlockLightMask.ToByte()...)
	out = append(out, ld.EmptySkyLightMask.ToByte()...)
	out = append(out, ld.EmptyBlockLightMask.ToByte()...)
	out = appendLightArrays(out, ld.SkyLight)
	out = appendLightArrays(out, ld.BlockLight)
	return out
}

func appendLightArrays(out []byte, arrays [][]byte) []byte {
	out = binary.AppendUvarint(out, uint64(len(arrays)))
	for _, a := range arrays {
		out = binary.AppendUvarint(out, uint64(len(a)))
		out = append(out, a...)
	}
	return out
}

var litSection = bytes.Repeat([]byte{0xFF}, lightSectionBytes)

func FullBrightLight() LightData {
	sky := make([][]byte, lightSections)
	for i := range sky {
		sky[i] = litSection
	}
	all := protocol.BitSet{int64(1)<<lightSections - 1}
	return LightData{
		SkyLightMask:        all,
		EmptyBlockLightMask: all,
		SkyLight:            sky,
	}
}

type ChunkSection struct {
	BlockCount  int16
	FluidCount  int16
	BlockStates PalettedContainer
	Biomes      PalettedContainer
}

func (cs ChunkSection) ToByte() []byte {
	out := []byte{}
	out = binary.BigEndian.AppendUint16(out, uint16(cs.BlockCount))
	out = binary.BigEndian.AppendUint16(out, uint16(cs.FluidCount))
	out = append(out, cs.BlockStates.ToByte()...)
	out = append(out, cs.Biomes.ToByte()...)
	return out
}

type PalettedContainer struct {
	BitsPerEntry uint8
	Palette      int32 // TODO handle more palettes
	DataArray    []int64
}

func (pc PalettedContainer) ToByte() []byte {
	out := []byte{}
	out = append(out, pc.BitsPerEntry)
	out = binary.AppendUvarint(out, uint64(pc.Palette))
	for _, v := range pc.DataArray {
		out = binary.BigEndian.AppendUint64(out, uint64(v))
	}
	return out
}
