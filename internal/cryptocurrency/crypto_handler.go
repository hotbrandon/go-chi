package cryptocurrency

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// If timestamps can be NULL, use:
// TransactionDate sql.NullTime `json:"transaction_date"`
// CreatedAt       sql.NullTime `json:"created_at"`
type Transaction struct {
	TransactionsSeq int       `json:"transactions_seq"`
	CoinSymbol      string    `json:"coin_symbol"`
	TransactionType string    `json:"transaction_type"`
	Quantity        float64   `json:"quantity"`
	PricePerUnit    float64   `json:"price_per_unit"`
	TotalCost       float64   `json:"total_cost"`
	TransactionDate time.Time `json:"transaction_date"` // Changed
	Exchange        string    `json:"exchange"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"` // Changed
}

type CryptoHandler struct {
	svc *CryptoService
}

func NewCryptoHandler(svc *CryptoService) *CryptoHandler {
	return &CryptoHandler{
		svc,
	}
}

func (h *CryptoHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	err := h.svc.GetTransactions(r.Context())
	if err != nil {
		slog.Error("GetTransactions", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	transactions := struct {
		Transactions []Transaction `json:"transactions"`
	}{}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}
