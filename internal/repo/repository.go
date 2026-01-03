package repo

import (
	"context"
)

// type CryptoRepository interface {
// 	ListTransactions(ctx context.Context) ([]Transaction, error)
// }

func (r *Repository) ListTransactions(ctx context.Context, page, pageSize int) ([]Transaction, error) {
	var transactions []Transaction

	startRow := (page - 1) * pageSize
	endRow := page * pageSize

	// This query uses ROWNUM for pagination, which is compatible with Oracle 11gR2.
	// ORDER BY is crucial for stable pagination results.
	query := `
		SELECT transactions_seq, coin_symbol, transaction_type, quantity, price_per_unit, total_cost, transaction_date, exchange, notes, created_at
		FROM (
			SELECT t.*, ROWNUM rnum
			FROM (
				SELECT
					transactions_seq,
					coin_symbol,
					transaction_type,
					quantity,
					price_per_unit,
					total_cost,
					TO_CHAR(transaction_date, 'YYYY-MM-DD"T"HH24:MI:SS') AS transaction_date,
					exchange,
					notes,
					TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS') AS created_at
				FROM transactions
				ORDER BY transaction_date DESC, transactions_seq DESC
			) t
			WHERE ROWNUM <= :1
		)
		WHERE rnum > :2`

	rows, err := r.db.QueryContext(ctx, query, endRow, startRow)
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

func (r *Repository) CreateTransaction(ctx context.Context, t Transaction) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO TRANSACTIONS (
			TRANSACTIONS_SEQ,
			COIN_SYMBOL,
			TRANSACTION_TYPE,
			QUANTITY,
			PRICE_PER_UNIT,
			TOTAL_COST,
			TRANSACTION_DATE,
			EXCHANGE,
			NOTES
		) VALUES (
			TRANSACTIONS_SEQ.NEXTVAL, :1, :2, :3, :4, :5, TO_DATE(:6, 'YYYY-MM-DD'), :7, :8
		)`,
		t.CoinSymbol, t.TransactionType, t.Quantity, t.PricePerUnit, t.TotalCost, t.TransactionDate, t.Exchange, t.Notes)

	return err
}
