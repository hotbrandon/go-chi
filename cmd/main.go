package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
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
	cfg            config
	dbs            map[string]*sql.DB
	dbMutex        sync.RWMutex
	failedDBs      map[string]time.Time // Track when DB last failed
	failedDBsMutex sync.RWMutex
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
		dbs:       make(map[string]*sql.DB),
		failedDBs: make(map[string]time.Time),
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

	slog.Info("database initialization complete",
		"configured", len(dbConfigs),
		"connected", successCount)

	// Add cleanup here, BEFORE app.serve()
	defer func() {
		app.dbMutex.Lock()
		defer app.dbMutex.Unlock()
		for dbID, db := range app.dbs {
			slog.Info("closing database connection", "database_id", dbID)
			db.Close()
		}
	}()

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
	db, err := sql.Open(driver, dsn)
	if err != nil {
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
		db.Close()
		return nil, err
	}

	slog.Info("database connection verified", "database", databaseId)
	return db, nil
}

// Add lazy connection method
// attemptConnection tries to connect to a database if not already connected
// Returns the connection or nil if it fails (with appropriate backoff)
func (app *application) getOrConnectDB(dbID string) (*sql.DB, error) {
	// First, check if we already have a healthy connection
	app.dbMutex.RLock()
	db, exists := app.dbs[dbID]
	app.dbMutex.RUnlock()

	if exists {
		// Quick ping to verify it's still healthy
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := db.PingContext(ctx)
		cancel()

		if err == nil {
			return db, nil // Connection is good
		}

		// Connection is bad, remove it
		slog.Warn("existing database connection is unhealthy, will reconnect",
			"database_id", dbID,
			"error", err)

		app.dbMutex.Lock()
		delete(app.dbs, dbID)
		db.Close()
		app.dbMutex.Unlock()
	}

	// Check if we recently failed (implement backoff)
	app.failedDBsMutex.RLock()
	lastFailed, recentlyFailed := app.failedDBs[dbID]
	app.failedDBsMutex.RUnlock()

	if recentlyFailed {
		// Don't retry for 30 seconds after last failure
		if time.Since(lastFailed) < 30*time.Second {
			return nil, fmt.Errorf("database recently failed, retry after %v",
				30*time.Second-time.Since(lastFailed))
		}
	}

	// Get the config for this database
	dbConfig, exists := app.cfg.databases[dbID]
	if !exists {
		return nil, fmt.Errorf("database configuration not found")
	}

	// Attempt to connect
	slog.Info("attempting to connect to database", "database_id", dbID)
	dsn := dbConfig.BuildDSN()
	db, err := openDatabase("oracle", dsn, dbID)

	if err != nil {
		// Mark as failed
		app.failedDBsMutex.Lock()
		app.failedDBs[dbID] = time.Now()
		app.failedDBsMutex.Unlock()

		slog.Warn("failed to connect to database",
			"database_id", dbID,
			"error", err)
		return nil, err
	}

	// Success! Store the connection and clear failure record
	app.dbMutex.Lock()
	app.dbs[dbID] = db
	app.dbMutex.Unlock()

	app.failedDBsMutex.Lock()
	delete(app.failedDBs, dbID)
	app.failedDBsMutex.Unlock()

	slog.Info("database connected successfully via lazy connection",
		"database_id", dbID)

	return db, nil
}
