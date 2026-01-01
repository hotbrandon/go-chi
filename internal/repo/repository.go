package repo

import (
	"context"
)

// type CryptoRepository interface {
// 	ListTransactions(ctx context.Context) ([]Transaction, error)
// }

func (r *Repository) ListTransactions(ctx context.Context) ([]Transaction, error) {
	var transactions []Transaction

	rows, err := r.db.QueryContext(ctx, "SELECT transactions_seq, coin_symbol, transaction_type, quantity, price_per_unit, total_cost, transaction_date, exchange, notes, created_at FROM transactions")
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
