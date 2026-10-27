// generate.go
package main

//go:generate go run .

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func main() {
	root, err := moduleRoot()
	if err != nil {
		panic(err)
	}
	if err := os.Chdir(root); err != nil {
		panic(err)
	}

	if err := genBlocks("./data/minecraft/blocks.json", "./internal/block"); err != nil {
		panic(err)
	}
	if err := genPackets("./data/minecraft/packets.json", "./internal/protocol/packetid"); err != nil {
		panic(err)
	}
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found")
		}
		dir = parent
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
