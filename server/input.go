package server

import (
	"fmt"
	"math/bits"
)

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
