package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hotbrandon/go-chi/internal/cryptocurrency"
)

type database_id string
type config struct {
	app_addr   string
	oracle_dsn map[database_id]string
}

type application struct {
	cfg config
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cryptoService := cryptocurrency.NewCryptoService()
	cryptoHandler := cryptocurrency.NewCryptoHandler(cryptoService)

	r.Get("/crypto/transactions", cryptoHandler.GetTransactions)
	return r
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:    app.cfg.app_addr,
		Handler: app.mount(),
	}

	slog.Info("starting server", "address", app.cfg.app_addr)
	return srv.ListenAndServe()
}
