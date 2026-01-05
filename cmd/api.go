package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	cryptocurrency "github.com/hotbrandon/go-chi/internal/crypto"
	"github.com/hotbrandon/go-chi/internal/repo"
)

type contextKey string

const dbContextKey contextKey = "database"
const dbIDContextKey contextKey = "database_id"

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health checks (no authentication needed)
	r.Get("/health", app.healthCheckHandler)
	r.Get("/health/readiness", app.readinessCheckHandler)

	// List available databases (useful for frontends)
	r.Get("/databases", app.listDatabasesHandler)

	// API routes - database ID in path
	r.Route("/api/{database_id}", func(r chi.Router) {
		r.Use(app.databaseMiddleware)

		// Crypto/transaction endpoints
		r.Route("/crypto", func(r chi.Router) {
			r.Get("/transactions", app.listTransactionsHandler)
			r.Post("/transactions", app.createTransactionHandler)
			r.Get("/transactions/{id}", app.getTransactionHandler)
			// Add more as needed
		})

		// You can add other resource types here
		// r.Route("/users", func(r chi.Router) { ... })
		// r.Route("/reports", func(r chi.Router) { ... })
	})

	return r
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         app.cfg.appAddr,
		Handler:      app.mount(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("server starting",
		"address", app.cfg.appAddr,
		"databases", len(app.dbs))
	return srv.ListenAndServe()
}

// Health check - simple liveness probe
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Readiness check - tests all database connections
func (app *application) readinessCheckHandler(w http.ResponseWriter, r *http.Request) {
	type DatabaseHealth struct {
		ID        string `json:"id"`
		Available bool   `json:"available"`
		Latency   string `json:"latency,omitempty"`
		Error     string `json:"error,omitempty"`
	}

	type ReadinessResponse struct {
		Status     string           `json:"status"` // "ready", "degraded", "not_ready"
		Timestamp  string           `json:"timestamp"`
		Databases  []DatabaseHealth `json:"databases"`
		TotalDBs   int              `json:"total_databases"`
		HealthyDBs int              `json:"healthy_databases"`
	}

	health := ReadinessResponse{
		Status:    "ready",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TotalDBs:  len(app.dbs),
	}

	healthyCount := 0

	for dbID, db := range app.dbs {
		start := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		err := db.PingContext(ctx)
		cancel()
		latency := time.Since(start)

		dbHealth := DatabaseHealth{
			ID:        dbID,
			Available: err == nil,
		}

		if err == nil {
			dbHealth.Latency = latency.String()
			healthyCount++
		} else {
			dbHealth.Error = err.Error()
		}

		health.Databases = append(health.Databases, dbHealth)
	}

	health.HealthyDBs = healthyCount

	// Determine overall status
	if healthyCount == 0 {
		health.Status = "not_ready"
	} else if healthyCount < len(app.dbs) {
		health.Status = "degraded"
	}

	statusCode := http.StatusOK
	if health.Status == "not_ready" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// List available databases - useful for frontend to build UI
func (app *application) listDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	type DatabaseInfo struct {
		ID        string `json:"id"`
		Available bool   `json:"available"`
	}

	databases := make([]DatabaseInfo, 0, len(app.dbs))
	for dbID := range app.dbs {
		databases = append(databases, DatabaseInfo{
			ID:        dbID,
			Available: true,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"databases": databases,
		"count":     len(databases),
	})
}

// Middleware: Inject database connection into request context
func (app *application) databaseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbID := strings.ToLower(chi.URLParam(r, "database_id"))

		// Check if database config exists
		if _, exists := app.cfg.databases[dbID]; !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Database Not Found",
				"message": "The specified database does not exist or is not configured",
				"code":    "DB_NOT_FOUND",
			})
			return
		}

		// Get or connect to database (lazy connection)
		db, err := app.getOrConnectDB(dbID)
		if err != nil {
			slog.Warn("database unavailable during request",
				"database_id", dbID,
				"error", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Database Unavailable",
				"message": "The database is temporarily unavailable. Please try again later.",
				"code":    "DB_UNAVAILABLE",
			})
			return
		}

		// Add both database connection and ID to context
		ctx := context.WithValue(r.Context(), dbContextKey, db)
		ctx = context.WithValue(ctx, dbIDContextKey, dbID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Handler: Create transaction
func (app *application) createTransactionHandler(w http.ResponseWriter, r *http.Request) {
	db := r.Context().Value(dbContextKey).(*sql.DB)
	dbID := r.Context().Value(dbIDContextKey).(string)

	cryptoRepo := repo.New(db)
	cryptoHandler := cryptocurrency.NewCryptoHandler(cryptoRepo)

	slog.Info("creating transaction", "database_id", dbID)
	cryptoHandler.CreateTransaction(w, r)
}

// Handler: List transactions
func (app *application) listTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	db := r.Context().Value(dbContextKey).(*sql.DB)
	dbID := r.Context().Value(dbIDContextKey).(string)

	cryptoRepo := repo.New(db)
	cryptoHandler := cryptocurrency.NewCryptoHandler(cryptoRepo)

	slog.Info("listing transactions", "database_id", dbID)
	cryptoHandler.ListTransactions(w, r)
}

// Handler: Get single transaction (example of additional endpoint)
func (app *application) getTransactionHandler(w http.ResponseWriter, r *http.Request) {
	// Example implementation
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "Not implemented yet",
	})
}
