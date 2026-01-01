package repo

import (
	"context"
	"database/sql"
)

// DBTX is an interface that wraps the basic methods of *sql.DB and *sql.Tx
// to allow for using either in the Repository struct.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type Repository struct {
	db DBTX
}

func New(db DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithTx(tx *sql.Tx) *Repository {
	return &Repository{
		db: tx,
	}
}
