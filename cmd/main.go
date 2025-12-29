package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	appAddrEnv := os.Getenv("APP_ADDR")
	if appAddrEnv == "" {
		slog.Error("APP_ADDR environment variable is not set")
		os.Exit(1)
	}

	app := application{
		cfg: config{
			app_addr: appAddrEnv,
		},
	}

	if err := app.serve(); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
