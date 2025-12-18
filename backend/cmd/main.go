package main

import (
	"log/slog"
	"os"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/server"
)

// These variables are set at build time using ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	envConfig := config.LoadEnv()

	cfg, err := config.Load(envConfig.ConfigPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Override version from config with build-time version if available
	if Version != "dev" {
		cfg.App.Version = Version
	}

	// Log version information
	slog.Info("Starting application",
		"name", cfg.App.Name,
		"version", cfg.App.Version,
		"build_time", BuildTime,
		"git_commit", GitCommit,
	)

	if err := server.Start(cfg); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
