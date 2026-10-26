// generate.go
package main

import (
	"strings"
	"unicode"
)

func main() {
	if err := genBlocks("./data/minecraft/blocks.json", "./internal/block"); err != nil {
		panic(err)
	}
	if err := genPackets("./data/minecraft/packets.json", "./internal/protocol/packetid"); err != nil {
		panic(err)
	}
}

type packet struct {
	ProtocolID int `json:"protocol_id"`
}

func goName(name string) string {
	name = strings.TrimPrefix(name, "minecraft:")

	parts := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var sb strings.Builder
	for _, part := range parts {
		sb.WriteString(strings.ToUpper(part[:1]))
		sb.WriteString(part[1:])
	}
	return sb.String()
}
