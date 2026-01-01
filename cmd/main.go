package main

import (
	"database/sql"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/sijms/go-ora/v2"
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
			oracle_dsn: map[database_id]string{
				"lab_dsn": os.Getenv("LAB_DSN"),
			},
		},
		db: make(map[database_id]*sql.DB),
	}

	lab_db, err := sql.Open("oracle", app.cfg.oracle_dsn["lab_dsn"])
	if err != nil {
		slog.Error("failed to connect to lab_dsn", "error", err)
		os.Exit(1)
	}
	if err := lab_db.Ping(); err != nil {
		slog.Error("failed to ping lab_dsn", "error", err)
		os.Exit(1)
	}
	defer lab_db.Close()

	app.db["lab_db"] = lab_db

	if err := app.serve(); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
