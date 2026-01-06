// internal/handlers/crypto_integration_test.go
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hotbrandon/go-chi/internal/handlers"
	"github.com/hotbrandon/go-chi/internal/repo"
	_ "github.com/sijms/go-ora/v2"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set, skipping integration test")
	}

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	return db
}

// cleanupTestData removes test data after tests
func cleanupTestData(t *testing.T, db *sql.DB) {
	_, err := db.Exec("DELETE FROM TRANSACTIONS WHERE EXCHANGE = 'TEST_EXCHANGE'")
	if err != nil {
		t.Logf("warning: failed to cleanup test data: %v", err)
	}
}

func TestIntegration_CreateAndListTransactions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repository := repo.New(db)
	handler := handlers.NewCryptoHandlers()

	// Test 1: Create a transaction
	t.Run("create transaction", func(t *testing.T) {
		payload := handlers.CreateTransactionRequest{
			CoinSymbol:      "BTC",
			TransactionType: "BUY",
			Quantity:        1.5,
			PricePerUnit:    45000.00,
			TotalCost:       67500.00,
			TransactionDate: "2024-01-20",
			Exchange:        "TEST_EXCHANGE", // Marker for cleanup
			Notes:           "Integration test",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/crypto/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), handlers.RepoContextKey, repository)
		ctx = context.WithValue(ctx, handlers.DBIDContextKey, "test_db")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.CreateTransaction(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", 
				http.StatusCreated, w.Code, w.Body.String())
		}
	})

	// Test 2: List transactions (should include the one we just created)
	t.Run("list transactions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/crypto/transactions?page=1&page_size=10", nil)
		
		ctx := context.WithValue(req.Context(), handlers.RepoContextKey, repository)
		ctx = context.WithValue(ctx, handlers.DBIDContextKey, "test_db")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ListTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		transactions := response["transactions"].([]interface{})
		if len(transactions) == 0 {
			t.Error("expected at least 1 transaction")
		}
	})
}

// ============================================================================
// cmd/api_test.go - Testing full API with middleware
// ============================================================================

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hotbrandon/go-chi/internal/repo"
)

// Mock database connection for testing middleware
type mockDB struct{}

func (m *mockDB) PingContext(ctx context.Context) error {
	return nil
}

func TestDatabaseMiddleware_NotFound(t *testing.T) {
	app := &application{
		cfg: config{
			databases: map[string]DatabaseConfig{
				"sales": {},
			},
		},
	}

	handler := app.databaseMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/nonexistent/crypto/transactions", nil)
	// Mock chi URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("database_id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)

	if response["code"] != "DB_NOT_FOUND" {
		t.Errorf("expected error code DB_NOT_FOUND, got %s", response["code"])
	}
}

func TestHealthCheckHandler(t *testing.T) {
	app := &application{}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	app.healthCheckHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status ok, got %v", response["status"])
	}
}

// ============================================================================
// Makefile or test commands
// ============================================================================

/*
# Makefile

.PHONY: test test-unit test-integration test-coverage

# Run all tests (unit only, fast)
test:
	go test ./...

# Run unit tests with verbose output
test-unit:
	go test -v ./...

# Run integration tests (requires database)
test-integration:
	TEST_DB_DSN="oracle://user:pass@host:1521/sid" go test -v -tags=integration ./...

# Run with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
test-handlers:
	go test -v ./internal/handlers/...

test-repo:
	go test -v ./internal/repo/...

# Watch mode (requires entr or similar)
test-watch:
	find . -name "*.go" | entr -c go test ./...
*/

// ============================================================================
// Run tests:
// ============================================================================

/*
# Run all unit tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package
go test ./internal/handlers/...

# Run specific test function
go test -v -run TestCreateTransaction_Success ./internal/handlers/...

# Run with coverage
go test -cover ./...

# Run integration tests (with build tag)
TEST_DB_DSN="oracle://..." go test -v -tags=integration ./...

# Run tests in parallel
go test -v -parallel 4 ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmark tests
go test -bench=. ./...
*/