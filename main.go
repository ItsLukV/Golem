package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	reg "github.com/itslukv/minecraftserver/registries"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger() *zap.SugaredLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stdout"}
	logger, _ := cfg.Build()
	return logger.Sugar()
}

func main() {
	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	logger := newLogger()

	registries, err := reg.LoadRegistries()
	if err != nil {
		logger.Fatal(err)
	}

	tags, err := reg.LoadTags(registries)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Infoln("Server listening on :25565")
	ms := MinecraftServer{
		logger:     logger,
		registries: registries,
		tags:       tags,
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Infoln("accept error:", err)
			continue
		}

		go ms.handleConnection(conn)
	}
}

type connState int

const (
	stateHandshaking connState = iota
	stateStatus
	stateLogin
	stateConfiguration
	statePlay
)

func (state connState) String() string {
	switch state {
	case stateHandshaking:
		return "Handshaking"
	case stateConfiguration:
		return "Configuration"
	case stateLogin:
		return "Login"
	case statePlay:
		return "Play"
	case stateStatus:
		return "Status"
	default:
		panic(fmt.Sprintf("unexpected main.connState: %#v", state))
	}
}

type connection struct {
	conn  net.Conn
	r     *bufio.Reader
	state connState

	// Guards conn against the keep-alive goroutine writing a frame in the
	// middle of one written from the read loop.
	w sync.Mutex

	// Set once the login packet is parsed.
	name string
	uuid uuid.UUID
}

func (c *connection) readPacket() (packetID uint64, body *bytes.Reader, err error) {
	length, err := binary.ReadUvarint(c.r)
	if err != nil {
		return 0, nil, err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		return 0, nil, err
	}
	p := bytes.NewReader(buf)
	id, err := binary.ReadUvarint(p)
	if err != nil {
		return 0, nil, err
	}
	return id, p, nil
}

type MinecraftServer struct {
	logger     *zap.SugaredLogger
	registries []reg.Registry
	tags       []reg.TagRegistry
}

func (ms *MinecraftServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	c := &connection{
		conn:  conn,
		r:     bufio.NewReader(conn),
		state: stateHandshaking,
	}

	_, p, err := c.readPacket()
	if err != nil {
		if err != io.EOF {
			ms.logger.Errorln("read packet:", err)
		}
		return
	}

	protoVersion, _ := binary.ReadUvarint(p)
	if protoVersion != 776 {
		ms.logger.Errorf("wrong proto version. Expected: 776. Got: %v", protoVersion)
		return
	}
	address, err := readString(p)
	if err != nil {
		ms.logger.Errorln("read address:", err)
		return
	}
	var port uint16
	binary.Read(p, binary.BigEndian, &port)
	intent, _ := binary.ReadUvarint(p)

	ms.logger.Infow("handshake",
		"proto", protoVersion,
		"addr", address,
		"port", port,
		"intent", intent,
	)

	switch intent {
	case 1: // Status
		c.state = stateStatus
		ms.handleStatus(c)
	case 2: // Login
		c.state = stateLogin
		if err := ms.handleLogin(c); err != nil {
			ms.logger.Errorln("handshake error:", err)
			return
		}
		if err := ms.handleConfiguration(c); err != nil {
			ms.logger.Errorw("configuration failed", "err", err)
			return
		}
		if err := ms.handlePlay(c); err != nil {
			ms.logger.Errorw("play failed", "err", err)
			return

		}
	case 3: // Transfer
	default:
		return
	}
}

func (ms *MinecraftServer) handleLogin(c *connection) error {
	if c.state != stateLogin {
		return fmt.Errorf("wrong state, expected login, got: %s", c.state)
	}

	name, uuid, err := ms.parseLoginPacket(c)
	if err != nil {
		return err
	}
	ms.logger.Infow("login", "name", name, "uuid", uuid)
	c.name, c.uuid = *name, *uuid

	err = ms.sendLoginSuccessPacket(c, *name, *uuid)
	if err != nil {
		return err
	}

	ackID, _, err := c.readPacket()
	if err != nil {
		return fmt.Errorf("read ackID: %v", err)
	}
	if ackID != LoginSbLoginAcknowledged {
		return fmt.Errorf("expected login ack, got: %v", ackID)
	}
	ms.logger.Infow("Login ack", "name", name)
	c.state = stateConfiguration
	return nil
}

func (ms *MinecraftServer) parseLoginPacket(c *connection) (*string, *uuid.UUID, error) {
	_, body, err := c.readPacket()
	if err != nil {
		ms.logger.Errorw("read packet:", err)
		return nil, nil, err
	}
	name, err := readString(body)
	if err != nil {
		ms.logger.Errorln("read name:", err)
		return nil, nil, err
	}
	uuid, err := readUUID(body)
	if err != nil {
		ms.logger.Errorln("read uuid:", err)
		return nil, nil, err
	}
	return &name, &uuid, nil
}

func (ms *MinecraftServer) sendLoginSuccessPacket(c *connection, username string, uuid uuid.UUID) error {
	p := Packet{
		PacketId: LoginCbLoginFinished,
	}
	p.AppendUUID(uuid)
	p.AppendString(username)
	p.AppendVarint(0)
	p.AppendUUID(uuid)
	return c.writePacket(p)
}

func (ms *MinecraftServer) handleConfiguration(c *connection) error {
	if c.state != stateConfiguration {
		return fmt.Errorf("wrong state, expected configuration, got %s", c.state)
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
		case ConfigSbClientInformation, ConfigSbCustomPayload, ConfigSbKeepAlive:
			// Ignore packet for now
			_ = r
		case ConfigSbSelectKnownPacks:
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
			if err := c.writePacket(Packet{PacketId: ConfigCbFinishConfiguration, Data: nil}); err != nil {
				return err
			}
		case ConfigSbFinishConfiguration:
			ms.logger.Infoln("configuration finished")
			c.state = statePlay
			return nil
		default:
			return fmt.Errorf("unexpected configuration packet 0x%02X", id)
		}
	}
}

func (ms *MinecraftServer) sendKnowPacks(c *connection) error {
	p := Packet{
		PacketId: ConfigCbSelectKnownPacks,
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
		p := Packet{
			PacketId: ConfigCbRegistryData,
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
	p := Packet{
		PacketId: ConfigCbUpdateTags,
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
	ms.logger.Infow("sending tags", "registries", len(ms.tags), "bytes", p.Len())
	return c.writePacket(p)
}

func (ms *MinecraftServer) handlePlay(c *connection) error {
	if err := ms.sendLogin(c); err != nil {
		return err
	}
	if err := ms.sendPlayerPosition(c); err != nil {
		return err
	}
	if err := ms.readAcceptTeleportation(c); err != nil {
		return err
	}
	if err := ms.sendPlayerInfoUpdate(c); err != nil {
		return err
	}
	if err := ms.sendGameEvent(c); err != nil {
		return err
	}
	if err := ms.sendSetChunkCacheCenter(c); err != nil {
		return err
	}
	if err := ms.sendChunks(c, 0, 0); err != nil {
		return err
	}
	return ms.playLoop(c)
}

const viewDistance = 2

func (ms *MinecraftServer) sendChunks(c *connection, centerX, centerZ int32) error {
	if err := c.writePacket(Packet{PacketId: PlayCbChunkBatchStart}); err != nil {
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

	p := Packet{PacketId: PlayCbChunkBatchFinished}
	p.AppendVarint(count)
	ms.logger.Infow("sent chunk batch", "chunks", count)
	return c.writePacket(p)
}

func (ms *MinecraftServer) sendLevelChunkWithLight(c *connection, chunkX, chunkZ int32) error {
	p := Packet{
		PacketId: PlayCbLevelChunkWithLight,
	}
	p.AppendInt(chunkX) // Chunk X
	p.AppendInt(chunkZ) // Chunk Z

	p.AppendVarint(0) // Heightmaps

	const floorSections = 4
	const stoneStateID = 1
	stone := ChunkSection{
		BlockCount:  16 * 16 * 16,
		FluidCount:  0,
		BlockStates: PalettedContainer{BitsPerEntry: 0, Palette: stoneStateID},
		Biomes:      PalettedContainer{BitsPerEntry: 0, Palette: 0},
	}

	air := ChunkSection{
		BlockCount:  0,
		FluidCount:  0,
		BlockStates: PalettedContainer{BitsPerEntry: 0, Palette: 0},
		Biomes:      PalettedContainer{BitsPerEntry: 0, Palette: 0},
	}

	solid, empty := stone.ToByte(), air.ToByte()

	sections := make([]byte, 0, worldSections*len(empty))
	for i := range worldSections {
		if i < floorSections {
			sections = append(sections, solid...)
			continue
		}
		sections = append(sections, empty...)
	}
	p.AppendPrefixedArray(sections) // Data

	p.AppendPrefixedArray([]byte{}) // Block Entities

	p.Append(fullBrightLight()) // Light Data

	return c.writePacket(p)
}

const (
	worldSections     = 384 / 16
	lightSections     = worldSections + 2
	lightSectionBytes = 16 * 16 * 16 / 2
)

type LightData struct {
	SkyLightMask        BitSet
	BlockLightMask      BitSet
	EmptySkyLightMask   BitSet
	EmptyBlockLightMask BitSet
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

func fullBrightLight() LightData {
	sky := make([][]byte, lightSections)
	for i := range sky {
		sky[i] = litSection
	}
	all := BitSet{int64(1)<<lightSections - 1}
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

func (ms *MinecraftServer) sendGameEvent(c *connection) error {
	p := Packet{
		PacketId: PlayCbGameEvent,
	}
	p.AppendByte(13)
	p.AppendFloat(13)
	return c.writePacket(p)
}

func (ms *MinecraftServer) sendSetChunkCacheCenter(c *connection) error {
	p := Packet{
		PacketId: PlayCbSetChunkCacheCenter,
	}
	p.AppendVarint(0)
	p.AppendVarint(0)
	return c.writePacket(p)
}

func (ms *MinecraftServer) playLoop(c *connection) error {
	stop := make(chan struct{})
	defer close(stop)
	go ms.sendKeepAlives(c, stop)

	unhandled := map[uint64]bool{}
	for {
		id, body, err := c.readPacket()
		if err != nil {
			if err == io.EOF {
				ms.logger.Infow("client disconnected", "name", c.name)
				return nil
			}
			return fmt.Errorf("read packet: %w", err)
		}

		switch id {
		case PlaySbKeepAlive:
			// Echo of our own token; only its arrival matters.
			var token int64
			if err := binary.Read(body, binary.BigEndian, &token); err != nil {
				return fmt.Errorf("keep alive: %w", err)
			}
		default:
			// Logged once per ID so the movement packets don't flood the log.
			if !unhandled[id] {
				unhandled[id] = true
				ms.logger.Debugw("unhandled play packet", "id", fmt.Sprintf("0x%02X", id))
			}
		}
	}
}

func (ms *MinecraftServer) sendKeepAlives(c *connection, stop <-chan struct{}) {
	return
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			p := Packet{PacketId: PlayCbKeepAlive}
			p.AppendLong(time.Now().UnixMilli())
			if err := c.writePacket(p); err != nil {
				ms.logger.Errorw("keep alive failed", "err", err)
				return
			}
		}
	}
}

func (ms *MinecraftServer) sendLogin(c *connection) error {
	p := Packet{
		PacketId: PlayCbLogin,
	}
	p.AppendInt(1)                                       // Entity ID
	p.AppendBool(false)                                  // Is hardcore
	p.AppendStringArray([]string{"minecraft:overworld"}) // Dimension Names
	p.AppendVarint(1)                                    // Max Players (unused)
	p.AppendVarint(viewDistance)                         // View Distance
	p.AppendVarint(8)                                    // Simulation Distance
	p.AppendBool(false)                                  // Reduced Debug Info
	p.AppendBool(true)                                   // Enable respawn screen
	p.AppendBool(false)                                  // Do limited crafting (unused)
	p.AppendVarint(0)                                    // Dimension Type (index into minecraft:dimension_type => overworld)
	p.AppendString("minecraft:overworld")                // Dimension Name
	p.AppendLong(0)                                      // Hashed seed
	p.AppendByte(0)                                      // Game mode (unsigned byte)
	p.AppendByte(0)                                      // Previous Game mode (byte)
	p.AppendBool(false)                                  // Is Debug
	p.AppendBool(false)                                  // Is Flat
	p.AppendBool(false)                                  // Has death location
	p.AppendVarint(0)                                    // Portal cooldown
	p.AppendVarint(50)                                   // Sea level
	p.AppendBool(false)                                  // Online mode
	p.AppendBool(false)                                  // Enforces Secure Chat
	ms.logger.Infow("sending play login", "bytes", p.Len())
	return c.writePacket(p)
}

type TeleportFlags int32

const (
	RelX           TeleportFlags = 1 << iota // 0x0001
	RelY                                     // 0x0002
	RelZ                                     // 0x0004
	RelYaw                                   // 0x0008
	RelPitch                                 // 0x0010
	RelVelX                                  // 0x0020
	RelVelY                                  // 0x0040
	RelVelZ                                  // 0x0080
	RotateVelocity                           // 0x0100
)

func (ms *MinecraftServer) sendPlayerPosition(c *connection) error {
	p := Packet{PacketId: PlayCbPlayerPosition}
	p.AppendVarint(0) // Teleport ID
	p.AppendDouble(0) // X
	p.AppendDouble(0) // Y
	p.AppendDouble(0) // Z
	p.AppendDouble(0) // Velocity X
	p.AppendDouble(0) // Velocity Y
	p.AppendDouble(0) // Velocity Z
	p.AppendFloat(0)  // Yaw
	p.AppendFloat(0)  // Pitch
	p.AppendInt(0)    // Relatives (teleport flags bitfield)
	return c.writePacket(p)
}

// readAcceptTeleportation waits for the client to confirm the teleport sent by
// sendPlayerPosition. The client is free to send other packets first
// (client_information, custom_payload, client_tick_end, ...), so skip anything
// that is not the confirmation rather than treating it as an error.
func (ms *MinecraftServer) readAcceptTeleportation(c *connection) error {
	for {
		packetID, body, err := c.readPacket()
		if err != nil {
			return fmt.Errorf("read packet: %w", err)
		}
		if packetID != PlaySbAcceptTeleportation {
			ms.logger.Debugw("waiting for teleport confirm, skipping packet",
				"id", fmt.Sprintf("0x%02X", packetID))
			continue
		}

		TeleportID, err := binary.ReadUvarint(body)
		if err != nil {
			return err
		}
		if TeleportID != 0 {
			return fmt.Errorf("wrong teleport id expected 0 and got %v", TeleportID)
		}
		return nil
		// for 26.3:
		// var X float64
		// if err := binary.Read(body, binary.BigEndian, &X); err != nil {
		// 	return err
		// }
		// var Y float64
		// if err := binary.Read(body, binary.BigEndian, &Y); err != nil {
		// 	return err
		// }
		// var Z float64
		// if err := binary.Read(body, binary.BigEndian, &Z); err != nil {
		// 	return err
		// }
		// var yaw float32
		// if err := binary.Read(body, binary.BigEndian, &yaw); err != nil {
		// 	return err
		// }
		// var pitch float32
		// if err := binary.Read(body, binary.BigEndian, &pitch); err != nil {
		// 	return err
		// }
		// return nil
	}
}

// PlayerInfoAction selects which per-entry fields the Player Info Update packet
// carries. The wire format is an EnumSet over the 8 actions below, encoded as a
// fixed bitset of ceil(8/8) = 1 byte with ADD_PLAYER as the least significant bit.
type PlayerInfoAction byte

const (
	AddPlayer         PlayerInfoAction = 1 << iota // 0x01
	InitializeChat                                 // 0x02
	UpdateGameMode                                 // 0x04
	UpdateListed                                   // 0x08
	UpdateLatency                                  // 0x10
	UpdateDisplayName                              // 0x20
	UpdateListOrder                                // 0x40
	UpdateHat                                      // 0x80
)

func (ms *MinecraftServer) sendPlayerInfoUpdate(c *connection) error {
	actions := AddPlayer | UpdateGameMode | UpdateListed | UpdateLatency

	p := Packet{PacketId: PlayCbPlayerInfoUpdate}
	p.AppendByte(byte(actions))
	p.AppendVarint(1) // Number of entries

	p.AppendUUID(c.uuid)   // Profile ID
	p.AppendString(c.name) // AddPlayer: name
	p.AppendVarint(0)      // AddPlayer: signed profile properties (none)
	p.AppendVarint(0)      // UpdateGameMode: survival
	p.AppendBool(true)     // UpdateListed: show in the tab list
	p.AppendVarint(0)      // UpdateLatency: ping in ms

	ms.logger.Infow("sending player info update",
		"name", c.name,
		"actions", fmt.Sprintf("0x%02X", byte(actions)),
		"bytes", p.Len(),
	)
	return c.writePacket(p)
}

func (ms *MinecraftServer) handleStatus(c *connection) {
	if c.state != stateStatus {
		ms.logger.Errorw("Wrong state, expected status", "state", c.state)
		return
	}
	for {
		packetID, p, err := c.readPacket()
		if err != nil {
			return
		}
		switch packetID {

		case StatusSbStatusRequest:
			status := `{
				"version": {"name": "26.2", "protocol": 776},
				"players": {"max": -1, "online": 0},
				"description": {"text": "Hello from Go!"}
			}`
			packet := Packet{
				PacketId: StatusCbStatusResponse,
			}
			packet.AppendString(status)
			if err := c.writePacket(packet); err != nil {
				return
			}

		case StatusSbPingRequest:
			var nonce int64
			if err := binary.Read(p, binary.BigEndian, &nonce); err != nil {
				return
			}
			packet := Packet{
				PacketId: StatusCbPongResponse,
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
