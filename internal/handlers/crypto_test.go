// internal/handlers/crypto_test.go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hotbrandon/go-chi/internal/handlers"
	"github.com/hotbrandon/go-chi/internal/repo"
)

// ============================================================================
// Mock Repository - Implements repo methods for testing
// ============================================================================

type MockRepository struct {
	// Store test data
	transactions []repo.Transaction
	createError  error
	listError    error
}

func (m *MockRepository) CreateTransaction(ctx context.Context, t repo.Transaction) error {
	if m.createError != nil {
		return m.createError
	}

	// Simulate auto-increment
	t.TransactionsSeq = len(m.transactions) + 1
	t.CreatedAt = "2024-01-15T10:30:00"
	m.transactions = append(m.transactions, t)
	return nil
}

func (m *MockRepository) ListTransactions(ctx context.Context, page, pageSize int) ([]repo.Transaction, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.transactions, nil
}

// ============================================================================
// Test Helpers
// ============================================================================

// setupRequest creates a request with mock repository in context
func setupRequest(method, url string, body interface{}) (*http.Request, *MockRepository) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	mockRepo := &MockRepository{
		transactions: []repo.Transaction{},
	}

	// Inject mock repository into context
	ctx := context.WithValue(req.Context(), handlers.RepoContextKey, mockRepo)
	ctx = context.WithValue(ctx, handlers.DBIDContextKey, "test_db")
	req = req.WithContext(ctx)

	return req, mockRepo
}

// ============================================================================
// CreateTransaction Tests
// ============================================================================

func TestCreateTransaction_Success(t *testing.T) {
	// Arrange: Set up request payload
	payload := handlers.CreateTransactionRequest{
		CoinSymbol:      "BTC",
		TransactionType: "BUY",
		Quantity:        0.5,
		PricePerUnit:    50000.00,
		TotalCost:       25000.00,
		TransactionDate: "2024-01-15",
		Exchange:        "Coinbase",
		Notes:           "First purchase",
	}

	req, mockRepo := setupRequest("POST", "/crypto/transactions", payload)
	w := httptest.NewRecorder()

	// Act: Call the handler
	handler := handlers.NewCryptoHandlers()
	handler.CreateTransaction(w, req)

	// Assert: Check response
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	// Verify transaction was saved
	if len(mockRepo.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(mockRepo.transactions))
	}

	savedTx := mockRepo.transactions[0]
	if savedTx.CoinSymbol != "BTC" {
		t.Errorf("expected coin_symbol BTC, got %s", savedTx.CoinSymbol)
	}
	if savedTx.Quantity != 0.5 {
		t.Errorf("expected quantity 0.5, got %f", savedTx.Quantity)
	}
}

func TestCreateTransaction_InvalidJSON(t *testing.T) {
	// Arrange: Invalid JSON payload
	req := httptest.NewRequest("POST", "/crypto/transactions",
		bytes.NewBufferString(`{"invalid json`))
	req.Header.Set("Content-Type", "application/json")

	mockRepo := &MockRepository{}
	ctx := context.WithValue(req.Context(), handlers.RepoContextKey, mockRepo)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	handler := handlers.NewCryptoHandlers()
	handler.CreateTransaction(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateTransaction_InvalidDate(t *testing.T) {
	// Arrange
	payload := handlers.CreateTransactionRequest{
		CoinSymbol:      "BTC",
		TransactionType: "BUY",
		Quantity:        0.5,
		PricePerUnit:    50000.00,
		TotalCost:       25000.00,
		TransactionDate: "invalid-date", // Bad format
		Exchange:        "Coinbase",
	}

	req, _ := setupRequest("POST", "/crypto/transactions", payload)
	w := httptest.NewRecorder()

	// Act
	handler := handlers.NewCryptoHandlers()
	handler.CreateTransaction(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateTransaction_RepositoryError(t *testing.T) {
	// Arrange
	payload := handlers.CreateTransactionRequest{
		CoinSymbol:      "BTC",
		TransactionType: "BUY",
		Quantity:        0.5,
		PricePerUnit:    50000.00,
		TotalCost:       25000.00,
		TransactionDate: "2024-01-15",
		Exchange:        "Coinbase",
	}

	req, mockRepo := setupRequest("POST", "/crypto/transactions", payload)

	// Simulate database error
	mockRepo.createError = context.DeadlineExceeded

	w := httptest.NewRecorder()

	// Act
	handler := handlers.NewCryptoHandlers()
	handler.CreateTransaction(w, req)

	// Assert
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// ============================================================================
// ListTransactions Tests
// ============================================================================

func TestListTransactions_Success(t *testing.T) {
	// Arrange: Pre-populate mock data
	req, mockRepo := setupRequest("GET", "/crypto/transactions?page=1&page_size=10", nil)

	mockRepo.transactions = []repo.Transaction{
		{
			TransactionsSeq: 1,
			CoinSymbol:      "BTC",
			TransactionType: "BUY",
			Quantity:        0.5,
			PricePerUnit:    50000.00,
			TotalCost:       25000.00,
			TransactionDate: "2024-01-15T00:00:00",
			Exchange:        "Coinbase",
			CreatedAt:       "2024-01-15T10:30:00",
		},
		{
			TransactionsSeq: 2,
			CoinSymbol:      "ETH",
			TransactionType: "BUY",
			Quantity:        2.0,
			PricePerUnit:    3000.00,
			TotalCost:       6000.00,
			TransactionDate: "2024-01-16T00:00:00",
			Exchange:        "Binance",
			CreatedAt:       "2024-01-16T14:20:00",
		},
	}

	w := httptest.NewRecorder()

	// Act
	handler := handlers.NewCryptoHandlers()
	handler.ListTransactions(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check response structure
	transactions, ok := response["transactions"].([]interface{})
	if !ok {
		t.Fatal("expected transactions array in response")
	}

	if len(transactions) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(transactions))
	}
}

func TestListTransactions_Pagination(t *testing.T) {
	// Arrange
	req, _ := setupRequest("GET", "/crypto/transactions?page=2&page_size=5", nil)
	w := httptest.NewRecorder()

	// Act
	handler := handlers.NewCryptoHandlers()
	handler.ListTransactions(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	// Check pagination params in response
	if page := response["page"].(float64); page != 2 {
		t.Errorf("expected page 2, got %v", page)
	}
	if pageSize := response["page_size"].(float64); pageSize != 5 {
		t.Errorf("expected page_size 5, got %v", pageSize)
	}
}

func TestListTransactions_DefaultPagination(t *testing.T) {
	// Arrange: No query params
	req, _ := setupRequest("GET", "/crypto/transactions", nil)
	w := httptest.NewRecorder()

	// Act
	handler := handlers.NewCryptoHandlers()
	handler.ListTransactions(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	// Should use defaults
	if page := response["page"].(float64); page != 1 {
		t.Errorf("expected default page 1, got %v", page)
	}
	if pageSize := response["page_size"].(float64); pageSize != 20 {
		t.Errorf("expected default page_size 20, got %v", pageSize)
	}
}

// ============================================================================
// Table-Driven Test Example
// ============================================================================

func TestCreateTransaction_Validation(t *testing.T) {
	tests := []struct {
		name           string
		payload        handlers.CreateTransactionRequest
		expectedStatus int
		description    string
	}{
		{
			name: "valid transaction",
			payload: handlers.CreateTransactionRequest{
				CoinSymbol:      "BTC",
				TransactionType: "BUY",
				Quantity:        0.5,
				PricePerUnit:    50000.00,
				TotalCost:       25000.00,
				TransactionDate: "2024-01-15",
				Exchange:        "Coinbase",
			},
			expectedStatus: http.StatusCreated,
			description:    "should accept valid transaction",
		},
		{
			name: "invalid date format",
			payload: handlers.CreateTransactionRequest{
				CoinSymbol:      "BTC",
				TransactionType: "BUY",
				Quantity:        0.5,
				PricePerUnit:    50000.00,
				TotalCost:       25000.00,
				TransactionDate: "01/15/2024", // Wrong format
				Exchange:        "Coinbase",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "should reject invalid date format",
		},
		{
			name: "missing date",
			payload: handlers.CreateTransactionRequest{
				CoinSymbol:      "BTC",
				TransactionType: "BUY",
				Quantity:        0.5,
				PricePerUnit:    50000.00,
				TotalCost:       25000.00,
				TransactionDate: "", // Empty
				Exchange:        "Coinbase",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "should reject empty date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			req, _ := setupRequest("POST", "/crypto/transactions", tt.payload)
			w := httptest.NewRecorder()

			// Act
			handler := handlers.NewCryptoHandlers()
			handler.CreateTransaction(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d",
					tt.description, tt.expectedStatus, w.Code)
			}
		})
	}
}
