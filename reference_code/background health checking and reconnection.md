# Add background health checking and reconnection

Create a background goroutine that periodically checks failed databases and attempts to reconnect:

```go
// Add to application struct
type application struct {
    cfg       config
    dbs       map[string]*sql.DB
    dbMutex   sync.RWMutex  // Protect concurrent access to dbs map
}

// Add this function to cmd/api.go
func (app *application) startDatabaseHealthChecker(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            app.checkAndReconnectDatabases()
        }
    }
}

func (app *application) checkAndReconnectDatabases() {
    // Check existing connections
    app.dbMutex.RLock()
    existingDBs := make(map[string]*sql.DB)
    for id, db := range app.dbs {
        existingDBs[id] = db
    }
    app.dbMutex.RUnlock()

    for dbID, db := range existingDBs {
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        err := db.PingContext(ctx)
        cancel()
        
        if err != nil {
            slog.Warn("database unhealthy, will retry on next check",
                "database_id", dbID,
                "error", err)
        }
    }

    // Try to connect to any missing databases
    for dbID, dbConfig := range app.cfg.databases {
        app.dbMutex.RLock()
        _, exists := app.dbs[dbID]
        app.dbMutex.RUnlock()
        
        if !exists {
            dsn := dbConfig.BuildDSN()
            db, err := openDatabase("oracle", dsn, dbID)
            if err == nil {
                app.dbMutex.Lock()
                app.dbs[dbID] = db
                app.dbMutex.Unlock()
                
                slog.Info("database reconnected successfully",
                    "database_id", dbID)
            }
        }
    }
}
```

Update main() to start health checker

```go
func main() {
    // ... existing code ...

    if successCount == 0 {
        slog.Warn("no databases available at startup, will retry in background")
        // Don't exit! Continue anyway
    }

    slog.Info("database initialization complete",
        "configured", len(dbConfigs),
        "connected", successCount)

    // Start background health checker
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go app.startDatabaseHealthChecker(ctx)

    if err := app.serve(); err != nil {
        slog.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

Protect database access in middleware
Update your middleware to use the mutex:

```go
func (app *application) databaseMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        dbID := strings.ToLower(chi.URLParam(r, "database_id"))

        app.dbMutex.RLock()
        db, exists := app.dbs[dbID]
        app.dbMutex.RUnlock()
        
        // ... rest of your existing middleware code ...
    })
}
```
