package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	reg "github.com/itslukv/minecraftserver/server/registries"
	"go.uber.org/zap"
)

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

	player *Player
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

type Player struct {
	EntityID int32
	name     string
	uuid     uuid.UUID

	mu  sync.RWMutex
	pos PlayerPos

	conn         *connection
	nextTeleport atomic.Int32
}

type PlayerPos struct {
	X, Y, Z    float64
	Yaw, Pitch float32
	OnGround   bool
}

type MinecraftServer struct {
	Logger     *zap.SugaredLogger
	registries []reg.Registry
	tags       []reg.TagRegistry

	mu      sync.RWMutex
	players map[uuid.UUID]*Player
	nextEID atomic.Int32

	hmu       sync.RWMutex
	onJoin    []func(*Player) error
	onMove    []func(*Player, PlayerPos) error
	onLeave   []func(*Player)
	onUseItem []func(*Player) error
}

func (ms *MinecraftServer) registerPlayer(c *connection, name string, id uuid.UUID) *Player {
	p := &Player{
		EntityID: ms.nextEID.Add(1),
		name:     name,
		uuid:     id,
		conn:     c,
	}
	ms.mu.Lock()
	ms.players[id] = p
	online := len(ms.players)
	ms.mu.Unlock()
	ms.Logger.Infow("player joined", "name", name, "entity", p.EntityID, "online", online)
	return p
}

func (ms *MinecraftServer) unregisterPlayer(p *Player) {
	ms.mu.Lock()
	delete(ms.players, p.uuid)
	online := len(ms.players)
	ms.mu.Unlock()
	ms.Logger.Infow("player left", "name", p.name, "online", online)
}

func (ms *MinecraftServer) Players() []*Player {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	out := make([]*Player, 0, len(ms.players))
	for _, p := range ms.players {
		out = append(out, p)
	}
	return out
}

func NewServer(log *zap.SugaredLogger, regs []reg.Registry, tags []reg.TagRegistry) *MinecraftServer {
	return &MinecraftServer{
		Logger:     log,
		registries: regs,
		tags:       tags,
		players:    map[uuid.UUID]*Player{},
	}
}

func (ms *MinecraftServer) HandleConnection(conn net.Conn) {
	defer conn.Close()
	c := &connection{
		conn:  conn,
		r:     bufio.NewReader(conn),
		state: stateHandshaking,
	}
	defer func() {
		if c.player != nil {
			ms.unregisterPlayer(c.player)
			ms.fireLeave(c.player)
		}
	}()

	_, p, err := c.readPacket()
	if err != nil {
		if err != io.EOF {
			ms.Logger.Errorln("read packet:", err)
		}
		return
	}

	protoVersion, _ := binary.ReadUvarint(p)
	if protoVersion != 776 {
		ms.Logger.Errorf("wrong proto version. Expected: 776. Got: %v", protoVersion)
		return
	}
	address, err := readString(p)
	if err != nil {
		ms.Logger.Errorln("read address:", err)
		return
	}
	var port uint16
	binary.Read(p, binary.BigEndian, &port)
	intent, _ := binary.ReadUvarint(p)

	ms.Logger.Infow("handshake",
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
			ms.Logger.Errorln("handshake error:", err)
			return
		}
		if err := ms.handleConfiguration(c); err != nil {
			ms.Logger.Errorw("configuration failed", "err", err)
			return
		}
		if err := ms.handlePlay(c); err != nil {
			ms.Logger.Errorw("play failed", "err", err)
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
	if ackID != LoginSbLoginAcknowledged {
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
	name, err := readString(body)
	if err != nil {
		ms.Logger.Errorln("read name:", err)
		return nil, nil, err
	}
	uuid, err := readUUID(body)
	if err != nil {
		ms.Logger.Errorln("read uuid:", err)
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
			ms.Logger.Infoln("configuration finished")
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
	ms.Logger.Infow("sending tags", "registries", len(ms.tags), "bytes", p.Len())
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
	if err := ms.fireJoin(c.player); err != nil {
		return fmt.Errorf("join rejected: %w", err)
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
	ms.Logger.Infow("sent chunk batch", "chunks", count)
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

	for {
		id, body, err := c.readPacket()
		if err != nil {
			if err == io.EOF {
				ms.Logger.Infow("client disconnected", "name", c.player.name)
				return nil
			}
			return fmt.Errorf("read packet: %w", err)
		}

		switch id {
		case PlaySbKeepAlive:
			var token int64
			if err := binary.Read(body, binary.BigEndian, &token); err != nil {
				return fmt.Errorf("keep alive: %w", err)
			}
		case PlaySbAcceptTeleportation:
			// Confirms a Teleport
		case PlaySbMovePlayerPos, PlaySbMovePlayerPosRot,
			PlaySbMovePlayerRot, PlaySbMovePlayerStatusOnly:
			pos := c.player.Pos()
			hasPos := id == PlaySbMovePlayerPos || id == PlaySbMovePlayerPosRot
			hasRot := id == PlaySbMovePlayerPosRot || id == PlaySbMovePlayerRot
			if err := readMovement(body, &pos, hasPos, hasRot); err != nil {
				ms.Logger.Errorw("movement",
					"id", fmt.Sprintf("0x%02X", id), "err", err)
				continue
			}
			prev := c.player.Pos()
			c.player.setPos(pos)
			if err := ms.fireMove(c.player, prev); err != nil {
				ms.Logger.Debugw("move rejected", "name", c.player.name, "err", err)
				if err := c.player.Teleport(prev); err != nil {
					return fmt.Errorf("rubber-band: %w", err)
				}
			}
		case PlaySbClientTickEnd:
			continue
		case PlaySbUseItemOn, PlaySbUseItem:
			if err := ms.fireOnUseItem(c.player); err != nil {
				ms.Logger.Debugw("failed to use item", "name", c.player.name, "err", err)
			}
		default:
			ms.Logger.Debugw("unhandled play packet", "id", fmt.Sprintf("0x%02X", id))
		}
	}
}

func readMovement(body *bytes.Reader, pos *PlayerPos, hasPos, hasRot bool) error {
	if hasPos {
		for _, f := range []*float64{&pos.X, &pos.Y, &pos.Z} {
			if err := binary.Read(body, binary.BigEndian, f); err != nil {
				return err
			}
		}
	}
	if hasRot {
		for _, f := range []*float32{&pos.Yaw, &pos.Pitch} {
			if err := binary.Read(body, binary.BigEndian, f); err != nil {
				return err
			}
		}
	}
	var flags byte
	if err := binary.Read(body, binary.BigEndian, &flags); err != nil {
		return err
	}
	pos.OnGround = flags&0x01 != 0
	return nil
}

func (ms *MinecraftServer) sendKeepAlives(c *connection, stop <-chan struct{}) {
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
				ms.Logger.Errorw("keep alive failed", "err", err)
				return
			}
		}
	}
}

func (ms *MinecraftServer) sendLogin(c *connection) error {
	p := Packet{
		PacketId: PlayCbLogin,
	}
	p.AppendInt(c.player.EntityID)                       // Entity ID
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
	ms.Logger.Infow("sending play login", "bytes", p.Len())
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

func (ms *MinecraftServer) readAcceptTeleportation(c *connection) error {
	for {
		packetID, body, err := c.readPacket()
		if err != nil {
			return fmt.Errorf("read packet: %w", err)
		}
		if packetID != PlaySbAcceptTeleportation {
			ms.Logger.Debugw("waiting for teleport confirm, skipping packet",
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

	p.AppendUUID(c.player.uuid)   // Profile ID
	p.AppendString(c.player.name) // AddPlayer: name
	p.AppendVarint(0)             // AddPlayer: signed profile properties (none)
	p.AppendVarint(0)             // UpdateGameMode: survival
	p.AppendBool(true)            // UpdateListed: show in the tab list
	p.AppendVarint(0)             // UpdateLatency: ping in ms

	ms.Logger.Infow("sending player info update",
		"name", c.player.name,
		"actions", fmt.Sprintf("0x%02X", byte(actions)),
		"bytes", p.Len(),
	)
	return c.writePacket(p)
}

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

		case StatusSbStatusRequest:
			status := fmt.Sprintf(`{
				"version": {"name": "26.2", "protocol": 776},
				"players": {"max": %v, "online": %v},
				"description": {"text": "Hello from Go!"}
			}`, len(ms.players)+1, len(ms.players))
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
