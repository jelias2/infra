package main

import (
	ohm "github.com/ethereum-optimism/infra/ophostmanager"
	"github.com/ethereum/go-ethereum/log"
	"log/slog"
	"os"
)

func main() {
	SetLogLevel(slog.LevelInfo)
	log.Info("Starting server")
	srv, err := ohm.NewServer()
	if err != nil {
		log.Crit("Error intializing server. Exiting...",
			"error", err.Error(),
		)
	}
	srv.Start()
}

func SetLogLevel(logLevel slog.Leveler) {
	log.SetDefault(log.NewLogger(slog.NewJSONHandler(
		os.Stdout, &slog.HandlerOptions{Level: logLevel})))
}
