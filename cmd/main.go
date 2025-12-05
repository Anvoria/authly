package main

import (
	"log/slog"
	"os"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/server"
)

func main() {
	envConfig := config.LoadEnv()

	cfg, err := config.Load(envConfig.ConfigPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	if err := server.Start(cfg); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
