//go:generate go run ./internal/gen
package main

import (
	"log"
	"math"
	"net"

	server "github.com/itslukv/minecraftserver/internal"
	reg "github.com/itslukv/minecraftserver/internal/registries"

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

	ms := server.NewServer(logger, registries, tags)

	ms.OnJoin(func(p *server.Player) error {
		pos := p.Pos()
		pos.X = 8
		pos.Z = 8
		return p.Teleport(pos)
	})

	ms.OnUseItem(func(p *server.Player) error {
		pos := p.Pos()
		dx, _, dz := lookVector(pos.Yaw, pos.Pitch)
		const l = 5
		pos.X += dx * l
		pos.Z += dz * l
		return p.Teleport(pos)
	})

	ms.OnMove(func(p *server.Player, prev server.PlayerPos) error {
		pos := p.Pos()
		if pos.Y-prev.Y > 0 && !pos.OnGround {
			pos.Y += 1
			return p.Teleport(pos)
		}
		return nil
	})

	ms.OnPlayerInput(func(p *server.Player, it []server.InputType) error {
		logger.Infof("Player Input: %v", it)
		return nil
	})

	logger.Infoln("Server listening on :25565")
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Infoln("accept error:", err)
			continue
		}

		go ms.HandleConnection(conn)
	}
}

func lookVector(yaw, pitch float32) (dx, dy, dz float64) {
	y := float64(yaw) * math.Pi / 180
	p := float64(pitch) * math.Pi / 180

	return -math.Sin(y) * math.Cos(p), -math.Sin(p), math.Cos(y) * math.Cos(p)
}
