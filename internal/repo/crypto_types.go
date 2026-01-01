package repo

import (
	"database/sql"
	"time"
)

// If timestamps can be NULL, use:
// TransactionDate sql.NullTime `json:"transaction_date"`
// CreatedAt       sql.NullTime `json:"created_at"`
type Transaction struct {
	TransactionsSeq int            `json:"transactions_seq"`
	CoinSymbol      string         `json:"coin_symbol"`
	TransactionType string         `json:"transaction_type"`
	Quantity        float64        `json:"quantity"`
	PricePerUnit    float64        `json:"price_per_unit"`
	TotalCost       float64        `json:"total_cost"`
	TransactionDate time.Time      `json:"transaction_date"`
	Exchange        string         `json:"exchange"`
	Notes           sql.NullString `json:"notes"`
	CreatedAt       time.Time      `json:"created_at"`
}
