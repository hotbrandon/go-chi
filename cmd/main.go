package main

import (
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/sijms/go-ora/v2"
)

type config struct {
	appAddr string
	db      dbConfig
}

type dbConfig struct {
	oracleDSNs map[string]string
}

type databaseStatus struct {
	OracleDatabases map[string]bool `json:"oracle_databases"`
}

type application struct {
	cfg      config
	db       map[string]*sql.DB
	dbStatus databaseStatus
}

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
	dsns := loadOracleDSNs()

	app := application{
		cfg: config{
			appAddr: appAddrEnv,
			db: dbConfig{
				oracleDSNs: dsns,
			},
		},
		db: make(map[string]*sql.DB),
	}

	dbStatus := databaseStatus{
		OracleDatabases: make(map[string]bool),
	}
	app.dbStatus = dbStatus

	for databaseId, dsn := range app.cfg.db.oracleDSNs {
		db, err := openDatabase("oracle", dsn, databaseId, false)
		if err != nil {
			slog.Warn("skipping database due to connection error", "database", databaseId)
			dbStatus.OracleDatabases[databaseId] = false
			continue
		}
		defer db.Close()
		app.db[databaseId] = db
		dbStatus.OracleDatabases[databaseId] = true
	}

	slog.Info("database initialization complete",
		"oracle_databases", dbStatus.OracleDatabases)

	if err := app.serve(); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}

func loadOracleDSNs() map[string]string {
	dsns := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		if strings.HasPrefix(key, "ORA_") && strings.HasSuffix(key, "_DSN") {
			slog.Info("found Oracle DSN:", "env", key, "dsn", value)
			databaseId := strings.ToUpper(strings.TrimSuffix(strings.TrimPrefix(key, "ORA_"), "_DSN"))
			dsns[databaseId] = value
		}
	}

	return dsns
}

func openDatabase(driver, dsn, databaseId string, required bool) (*sql.DB, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	var db *sql.DB
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sql.Open(driver, dsn)
		if err != nil {
			if required {
				slog.Error("failed to open database",
					"name", databaseId,
					"attempt", attempt,
					"error", err)
			} else {
				slog.Warn("failed to open database",
					"name", databaseId,
					"attempt", attempt,
					"error", err)
			}

			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, err
		}

		// Configure connection pool
		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(10 * time.Minute)

		// Verify connection
		err = db.Ping()
		if err != nil {
			if required {
				slog.Error("failed to ping database",
					"databaseId", databaseId,
					"attempt", attempt,
					"error", err)
			} else {
				slog.Warn("failed to ping database",
					"databaseId", databaseId,
					"attempt", attempt,
					"error", err)
			}

			db.Close()

			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, err
		}

		// Success!
		slog.Info("database connected",
			"databaseId", databaseId,
			"attempt", attempt)
		return db, nil
	}

	return nil, err
}
