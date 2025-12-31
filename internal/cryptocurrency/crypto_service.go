package cryptocurrency

import "context"

type ICryptoService interface {
	GetTransactions(ctx context.Context) error
}

type CryptoService struct {
	// repository
}

func NewCryptoService() *CryptoService {
	return &CryptoService{}
}

func (c *CryptoService) GetTransactions(ctx context.Context) error {
	// query := "SELECT * FROM transactions"
	// _, err := db.QueryContext(ctx, query)
	// if err != nil {
	// 	return err
	// }

	return nil
}
