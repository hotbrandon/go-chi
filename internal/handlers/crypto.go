package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/hotbrandon/go-chi/internal/repo"
)

type CryptoHandlers struct {
}

func NewCryptoHandlers() *CryptoHandlers {
	return &CryptoHandlers{}
}

type CreateTransactionRequest struct {
	CoinSymbol      string  `json:"coin_symbol"`
	TransactionType string  `json:"transaction_type"`
	Quantity        float64 `json:"quantity"`
	PricePerUnit    float64 `json:"price_per_unit"`
	TotalCost       float64 `json:"total_cost"`
	TransactionDate string  `json:"transaction_date"` // YYYY-MM-DD
	Exchange        string  `json:"exchange"`
	Notes           string  `json:"notes,omitempty"`
}

func (h *CryptoHandlers) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	repository := MustGetRepo(r.Context())
	dbID, _ := GetDBID(r.Context())

	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	slog.Info("creating transaction",
		"database_id", dbID,
		"coin", req.CoinSymbol)

	if _, err := time.Parse(time.DateOnly, req.TransactionDate); err != nil {
		http.Error(w, "Invalid transaction_date format (expected YYYY-MM-DD)", http.StatusBadRequest)
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

	if err := repository.CreateTransaction(r.Context(), t); err != nil {
		slog.Error("failed to create transaction", "error", err)
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
	})
}

func (h *CryptoHandlers) ListTransactions(w http.ResponseWriter, r *http.Request) {
	repository := MustGetRepo(r.Context())
	dbID, _ := GetDBID(r.Context())

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

	slog.Info("listing transactions",
		"database_id", dbID,
		"page", page,
		"page_size", pageSize)

	transactions, err := repository.ListTransactions(r.Context(), page, pageSize)
	if err != nil {
		slog.Error("failed to list transactions", "error", err)
		http.Error(w, "Failed to list transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transactions": transactions,
		"page":         page,
		"page_size":    pageSize,
	})
}

func (h *CryptoHandlers) GetTransaction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "Not implemented yet",
	})
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
