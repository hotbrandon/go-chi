package cryptocurrency

import (
	"encoding/json"
	"net/http"
)

type Transaction struct {
	transactions_seq int
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

	transactions := struct {
		Transactions []Transaction `json:"transactions"`
	}{}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}
