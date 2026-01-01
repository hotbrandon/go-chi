package repository

import (
	"context"
	"database/sql"
	"time"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}

type Querier interface {
	ListTransactions(ctx context.Context) ([]Transaction, error)
}

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
	TransactionDate time.Time      `json:"transaction_date"` // Changed
	Exchange        string         `json:"exchange"`
	Notes           sql.NullString `json:"notes"`
	CreatedAt       time.Time      `json:"created_at"` // Changed
}

func (q *Queries) ListTransactions(ctx context.Context) ([]Transaction, error) {
	var transactions []Transaction

	rows, err := q.db.QueryContext(ctx, "SELECT transactions_seq, coin_symbol, transaction_type, quantity, price_per_unit, total_cost, transaction_date, exchange, notes, created_at FROM transactions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t Transaction
		if err := rows.Scan(
			&t.TransactionsSeq,
			&t.CoinSymbol,
			&t.TransactionType,
			&t.Quantity,
			&t.PricePerUnit,
			&t.TotalCost,
			&t.TransactionDate,
			&t.Exchange,
			&t.Notes,
			&t.CreatedAt); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return transactions, nil
}
