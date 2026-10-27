# Golem

A Minecraft Java Edition server written from scratch in Go, targeting
version 26.2 (protocol 776).

## Running

```sh
go run ./cmd/golem
```

The server listens on `:25565`. With Nix: `nix run` or `nix build`.

## Hooks

Game logic is plugged in through hooks on the server:

```go
ms := server.NewServer(logger, registries, tags)

ms.OnJoin(func(p *server.Player) error { ... })
ms.OnMove(func(p *server.Player, prev server.PlayerPos) error { ... })
ms.OnUseItem(func(p *server.Player) error { ... })
ms.OnPlayerInput(func(p *server.Player, in []server.InputType) error { ... })
ms.OnLeave(func(p *server.Player) { ... })
```

Returning an error from `OnMove` rubber-bands the player back to `prev`.
See `cmd/golem/main.go` for examples.

## Layout

```
cmd/golem/            entry point and example hooks
server/               public API: server, players, hooks, one file per protocol phase
internal/protocol/    packet encoding/decoding; packetid/ is generated
internal/world/       chunk section, paletted container and light encoding
internal/block/       block state IDs (generated)
internal/registry/    loads synced registries and tags from data/
internal/gen/         code generator for block/ and protocol/packetid/
data/                 vanilla data from https://github.com/mj41/mc26-data (mc-26.2)
```

## Regenerating code

After updating `data/minecraft/blocks.json` or `packets.json`:

```sh
go generate ./...
```
