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
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	slog.Info("CreateTransaction", "request", req)
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

	if err := h.repo.CreateTransaction(r.Context(), t); err != nil {
		slog.Error("CreateTransaction", "error", err.Error())
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *CryptoHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size"))
	if err != nil || pageSize < 1 {
		pageSize = 20 // Default page size
	}
	if pageSize > 100 {
		pageSize = 100 // Max page size
	}

	transactions, err := h.repo.ListTransactions(r.Context(), page, pageSize)
	if err != nil {
		slog.Error("ListTransactions", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
