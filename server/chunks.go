package server

import (
	"fmt"

	"github.com/itslukv/minecraftserver/internal/block"
	"github.com/itslukv/minecraftserver/internal/protocol"
	cbPlay "github.com/itslukv/minecraftserver/internal/protocol/packetid/play/clientbound"
	"github.com/itslukv/minecraftserver/internal/world"
)

const viewDistance = 2

func (ms *MinecraftServer) sendChunks(c *connection, centerX, centerZ int32) error {
	if err := c.writePacket(protocol.Packet{PacketId: cbPlay.ChunkBatchStart}); err != nil {
		return fmt.Errorf("chunk batch start: %w", err)
	}

	var count int32
	for x := centerX - viewDistance; x <= centerX+viewDistance; x++ {
		for z := centerZ - viewDistance; z <= centerZ+viewDistance; z++ {
			if err := ms.sendLevelChunkWithLight(c, x, z); err != nil {
				return fmt.Errorf("chunk %d,%d: %w", x, z, err)
			}
			count++
		}
	}

	p := protocol.Packet{PacketId: cbPlay.ChunkBatchFinished}
	p.AppendVarint(count)
	ms.Logger.Infow("sent chunk batch", "chunks", count)
	return c.writePacket(p)
}

func (ms *MinecraftServer) sendLevelChunkWithLight(c *connection, chunkX, chunkZ int32) error {
	p := protocol.Packet{
		PacketId: cbPlay.LevelChunkWithLight,
	}

	p.AppendInt(chunkX) // Chunk X
	p.AppendInt(chunkZ) // Chunk Z

	p.AppendVarint(0) // Heightmaps

	const floorSections = 4
	stone := world.ChunkSection{
		BlockCount:  16 * 16 * 16,
		FluidCount:  0,
		BlockStates: world.PalettedContainer{BitsPerEntry: 0, Palette: block.OakLog},
		Biomes:      world.PalettedContainer{BitsPerEntry: 0, Palette: 0},
	}

	air := world.ChunkSection{
		BlockCount:  0,
		FluidCount:  0,
		BlockStates: world.PalettedContainer{BitsPerEntry: 0, Palette: block.Air},
		Biomes:      world.PalettedContainer{BitsPerEntry: 0, Palette: 0},
	}

	solid, empty := stone.ToByte(), air.ToByte()

	sections := make([]byte, 0, world.Sections*len(empty))
	for i := range world.Sections {
		if i < floorSections {
			sections = append(sections, solid...)
			continue
		}
		sections = append(sections, empty...)
	}
	p.AppendPrefixedArray(sections) // Data

	p.AppendPrefixedArray([]byte{}) // Block Entities

	p.Append(world.FullBrightLight()) // Light Data

	return c.writePacket(p)
}
