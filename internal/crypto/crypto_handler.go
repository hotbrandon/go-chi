package cryptocurrency

import (
	"encoding/json"
	"log/slog"
	"net/http"

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

func (h *CryptoHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := h.repo.ListTransactions(r.Context())
	if err != nil {
		slog.Error("ListTransactions", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}
