```go
package crypto

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/hotbrandon/go-chi/internal/repo"
)

type CryptoHandler struct {
	repo *repo.Repository
}

func NewCryptoHandler(repo *repo.Repository) *CryptoHandler {
	return &CryptoHandler{
		repo,
	}
}

func (h *CryptoHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid Request",
			Message: "Unable to parse request body. Please check your JSON format.",
			Code:    "INVALID_JSON",
		})
		return
	}

	// Validate transaction date format
	if _, err := time.Parse(time.DateOnly, req.TransactionDate); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid Date Format",
			Message: "Transaction date must be in YYYY-MM-DD format.",
			Code:    "INVALID_DATE_FORMAT",
		})
		return
	}

	// Validate required fields
	if req.CoinSymbol == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Missing Required Field",
			Message: "Coin symbol is required.",
			Code:    "MISSING_COIN_SYMBOL",
		})
		return
	}

	if req.TransactionType == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Missing Required Field",
			Message: "Transaction type is required.",
			Code:    "MISSING_TRANSACTION_TYPE",
		})
		return
	}

	if req.Quantity <= 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid Value",
			Message: "Quantity must be greater than zero.",
			Code:    "INVALID_QUANTITY",
		})
		return
	}

	t := repo.Transaction{
		CoinSymbol:      req.CoinSymbol,
		TransactionType: req.TransactionType,
		Quantity:        req.Quantity,
		PricePerUnit:    req.PricePerUnit,
		TotalCost:       req.TotalCost,
		TransactionDate: req.TransactionDate,
		Exchange:        req.Exchange,
		Notes:           stringToPtr(req.Notes),
	}

	if err := h.repo.CreateTransaction(r.Context(), t); err != nil {
		handleDatabaseError(w, err, "CreateTransaction")
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Transaction created successfully",
	})
}

func (h *CryptoHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	transactions, err := h.repo.ListTransactions(r.Context(), page, pageSize)
	if err != nil {
		handleDatabaseError(w, err, "ListTransactions")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": transactions,
		"page": page,
		"page_size": pageSize,
	})
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func writeJSONError(w http.ResponseWriter, statusCode int, errResp ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errResp)
}

// Import the classifyDatabaseError and handleDatabaseError functions from your main package
// or move them to a shared package like internal/errors
```