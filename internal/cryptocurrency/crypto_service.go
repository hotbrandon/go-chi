package cryptocurrency

import (
	"context"

	"github.com/hotbrandon/go-chi/internal/repository"
)

type ICryptoService interface {
	ListTransactions(ctx context.Context) ([]repository.Transaction, error)
}

type CryptoService struct {
	// repository
	repo repository.Querier
}

func NewCryptoService(repo repository.Querier) *CryptoService {
	return &CryptoService{
		repo: repo,
	}
}

func (c *CryptoService) ListTransactions(ctx context.Context) ([]repository.Transaction, error) {
	// query := "SELECT * FROM transactions"
	// _, err := db.QueryContext(ctx, query)
	// if err != nil {
	// 	return err
	// }

	return c.repo.ListTransactions(ctx)
}
