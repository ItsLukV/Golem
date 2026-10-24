package main

import (
	"log"
	"net"

	"github.com/itslukv/minecraftserver/server"
	reg "github.com/itslukv/minecraftserver/server/registries"

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
	ms := server.NewServer(logger, registries, tags)
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Infoln("accept error:", err)
			continue
		}

		go ms.HandleConnection(conn)
	}
}
