package main

import (
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

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Routes
	r.Get("/health", app.healthCheckHandler)
	r.Get("/health/databases", app.databaseStatusHandler)

	// The use of a random string to access app.db is error-prone
	cryptoRepo := repo.New(app.db["LAB"])
	cryptoHandler := cryptocurrency.NewCryptoHandler(cryptoRepo)
	r.Route("/crypto", func(r chi.Router) {
		r.Use(app.requireOracleDb)
		r.Get("/transactions", cryptoHandler.ListTransactions)
		r.Post("/transactions", cryptoHandler.CreateTransaction)
	})
	//
	return r
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:    app.cfg.appAddr,
		Handler: app.mount(),
	}

	slog.Info("starting server", "address", app.cfg.appAddr)
	return srv.ListenAndServe()
}

// Health check endpoint - basic liveness check
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (app *application) databaseStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(app.dbStatus)
}

// Middleware: Check if oracle database is available
func (app *application) requireOracleDb(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		databaseId := strings.ToUpper(r.URL.Query().Get("database_id"))
		available, exists := app.dbStatus.OracleDatabases[databaseId]

		if !exists {
			http.Error(w,
				`{"error":"database not found"}`,
				http.StatusNotFound)
			return
		}

		if !available {
			slog.Warn("request to database endpoint but database unavailable",
				"database_id", databaseId)
			http.Error(w,
				`{"error":"database is temporarily unavailable"}`,
				http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}
