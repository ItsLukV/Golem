package server

import (
	"fmt"
	"math/bits"
)

func (ms *MinecraftServer) OnJoin(fn func(*Player) error) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onJoin = append(ms.onJoin, fn)
}

func (ms *MinecraftServer) OnMove(fn func(p *Player, prev PlayerPos) error) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onMove = append(ms.onMove, fn)
}

func (ms *MinecraftServer) OnLeave(fn func(*Player)) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onLeave = append(ms.onLeave, fn)
}

func (ms *MinecraftServer) OnUseItem(fn func(*Player) error) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onUseItem = append(ms.onUseItem, fn)
}

type InputType int

const (
	Forward InputType = 1 << iota
	Backward
	Left
	Right
	Jump
	Sneak
	Sprint
)

func (i InputType) String() string {
	switch i {
	case Forward:
		return "Forward"
	case Backward:
		return "Backward"
	case Left:
		return "Left"
	case Right:
		return "Right"
	case Jump:
		return "Jump"
	case Sneak:
		return "Sneak"
	case Sprint:
		return "Sprint"
	}
	return fmt.Sprintf("Unknown Input Type: %v", int(i))
}

func (i InputType) Has(flag InputType) bool {
	return i&flag != 0
}

func (i InputType) Flags() []InputType {
	b := byte(i)
	flags := make([]InputType, 0, bits.OnesCount8(b))
	for ; b != 0; b &= b - 1 {
		flags = append(flags, InputType(1<<bits.TrailingZeros8(b)))
	}
	return flags
}

func (ms *MinecraftServer) OnPlayerInput(fn func(*Player, []InputType) error) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onPlayerInput = append(ms.onPlayerInput, fn)
}

func (ms *MinecraftServer) fireJoin(p *Player) error {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onJoin {
		if err := fn(p); err != nil {
			return err
		}
	}
	return nil
}

func (ms *MinecraftServer) fireMove(p *Player, prev PlayerPos) error {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onMove {
		if err := fn(p, prev); err != nil {
			return err
		}
	}
	return nil
}

func (ms *MinecraftServer) fireLeave(p *Player) {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onLeave {
		fn(p)
	}
}

func (ms *MinecraftServer) fireOnUseItem(p *Player) error {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onUseItem {
		if err := fn(p); err != nil {
			return err
		}
	}
	return nil
}

func (ms *MinecraftServer) fireOnPlayerInput(p *Player, input []InputType) error {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onPlayerInput {
		if err := fn(p, input); err != nil {
			return err
		}
	}
	return nil
}
