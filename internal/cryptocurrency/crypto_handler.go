package cryptocurrency

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type CryptoHandler struct {
	svc ICryptoService
}

func NewCryptoHandler(svc ICryptoService) *CryptoHandler {
	return &CryptoHandler{
		svc,
	}
}

func (h *CryptoHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := h.svc.ListTransactions(r.Context())
	if err != nil {
		slog.Error("GetTransactions", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}
