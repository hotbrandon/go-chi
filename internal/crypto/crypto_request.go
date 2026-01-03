package crypto

// omitEmpty only applies when marshaling to JSON; it has no effect when unmarshaling
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
