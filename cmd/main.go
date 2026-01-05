package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/sijms/go-ora/v2"
)

// DatabaseConfig holds configuration for a single database
type DatabaseConfig struct {
	ID       string
	Host     string
	Port     int
	SID      string
	User     string
	Password string
}

// BuildDSN creates a connection string from config
func (dc DatabaseConfig) BuildDSN() string {
	// Format: oracle://user:password@host:port/service_name
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		dc.User,
		dc.Password,
		dc.Host,
		dc.Port,
		dc.SID)
}

type config struct {
	appAddr   string
	databases map[string]DatabaseConfig
}

type application struct {
	cfg config
	dbs map[string]*sql.DB
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

	// Load database configurations from environment
	dbConfigs := loadDatabaseConfigs()
	if len(dbConfigs) == 0 {
		slog.Error("no database configurations found")
		os.Exit(1)
	}

	app := application{
		cfg: config{
			appAddr:   appAddrEnv,
			databases: dbConfigs,
		},
		dbs: make(map[string]*sql.DB),
	}

	// Connect to all configured databases
	successCount := 0
	for dbID, dbConfig := range app.cfg.databases {
		dsn := dbConfig.BuildDSN()
		db, err := openDatabase("oracle", dsn, dbID)
		if err != nil {
			slog.Error("failed to connect to database",
				"database_id", dbID,
				"error", err)
			// Option 1: Fail completely if any database fails
			// os.Exit(1)

			// Option 2: Continue with available databases
			continue
		}
		app.dbs[dbID] = db
		successCount++
		slog.Info("database connected successfully",
			"database_id", dbID,
			"host", dbConfig.Host,
			"sid", dbConfig.SID)
	}

	if successCount == 0 {
		slog.Error("no databases available, cannot start server")
		os.Exit(1)
	}

	slog.Info("database initialization complete",
		"configured", len(dbConfigs),
		"connected", successCount)

	if err := app.serve(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// loadDatabaseConfigs reads database configurations from environment variables
// Supports two patterns:
// 1. Full DSN: ORA_<ID>_DSN=oracle://user:pass@host:port/sid
// 2. Separate components: ORA_<ID>_HOST, ORA_<ID>_PORT, etc.
func loadDatabaseConfigs() map[string]DatabaseConfig {
	configs := make(map[string]DatabaseConfig)
	foundIDs := make(map[string]bool)

	// First pass: find all database IDs
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]

		// Look for ORA_* variables
		if strings.HasPrefix(key, "ORA_") {
			// Extract database ID (e.g., ORA_SALES_HOST -> SALES)
			remaining := strings.TrimPrefix(key, "ORA_")
			parts := strings.Split(remaining, "_")
			if len(parts) >= 2 {
				// Database ID is everything except the last part
				dbID := strings.Join(parts[:len(parts)-1], "_")
				foundIDs[dbID] = true
			}
		}
	}

	// Second pass: build configs for each database ID
	for dbID := range foundIDs {
		// Try full DSN first (easier)
		dsnKey := fmt.Sprintf("ORA_%s_DSN", dbID)
		if dsn := os.Getenv(dsnKey); dsn != "" {
			// For DSN string, we just store it and use it directly
			// You could parse it if needed, but simpler to just use it
			slog.Info("found database DSN", "database_id", dbID)
			configs[strings.ToLower(dbID)] = DatabaseConfig{
				ID: strings.ToLower(dbID),
				// Store as a marker that we should use the DSN directly
			}
			continue
		}

		// Try component-based config
		hostKey := fmt.Sprintf("ORA_%s_HOST", dbID)
		portKey := fmt.Sprintf("ORA_%s_PORT", dbID)
		sidKey := fmt.Sprintf("ORA_%s_SID", dbID)
		userKey := fmt.Sprintf("ORA_%s_USER", dbID)
		passKey := fmt.Sprintf("ORA_%s_PASSWORD", dbID)

		host := os.Getenv(hostKey)
		portStr := os.Getenv(portKey)
		sid := os.Getenv(sidKey)
		user := os.Getenv(userKey)
		password := os.Getenv(passKey)

		// Validate required fields
		if host == "" || sid == "" || user == "" || password == "" {
			slog.Warn("incomplete database configuration, skipping",
				"database_id", dbID,
				"has_host", host != "",
				"has_sid", sid != "",
				"has_user", user != "",
				"has_password", password != "")
			continue
		}

		port := 1521 // default Oracle port
		if portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}

		configs[strings.ToLower(dbID)] = DatabaseConfig{
			ID:       strings.ToLower(dbID),
			Host:     host,
			Port:     port,
			SID:      sid,
			User:     user,
			Password: password,
		}

		slog.Info("found database configuration",
			"database_id", dbID,
			"host", host,
			"port", port,
			"sid", sid)
	}

	return configs
}

// Alternative: load from DSN strings only (simpler)
// func loadDatabaseConfigsFromDSN() map[string]string {
// 	dsns := make(map[string]string)

// 	for _, env := range os.Environ() {
// 		parts := strings.SplitN(env, "=", 2)
// 		if len(parts) != 2 {
// 			continue
// 		}
// 		key := parts[0]
// 		value := parts[1]

// 		if strings.HasPrefix(key, "ORA_") && strings.HasSuffix(key, "_DSN") {
// 			dbID := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(key, "ORA_"), "_DSN"))
// 			dsns[dbID] = value
// 			slog.Info("found database DSN", "database_id", dbID)
// 		}
// 	}

// 	return dsns
// }

func openDatabase(driver, dsn, databaseId string) (*sql.DB, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	var db *sql.DB
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sql.Open(driver, dsn)
		if err != nil {
			slog.Warn("failed to open database",
				"database", databaseId,
				"attempt", attempt,
				"error", err)

			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, err
		}

		// Configure connection pool for production
		db.SetMaxOpenConns(25)                 // Max concurrent connections
		db.SetMaxIdleConns(5)                  // Idle connections to keep
		db.SetConnMaxLifetime(5 * time.Minute) // Recycle connections
		db.SetConnMaxIdleTime(2 * time.Minute) // Close idle connections

		// Verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = db.PingContext(ctx)
		if err != nil {
			slog.Warn("failed to ping database",
				"database", databaseId,
				"attempt", attempt,
				"error", err)

			db.Close()

			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, err
		}

		slog.Info("database connection verified",
			"database", databaseId,
			"attempt", attempt)
		return db, nil
	}

	return nil, err
}
