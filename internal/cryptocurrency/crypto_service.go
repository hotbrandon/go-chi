package cryptocurrency

import "context"

type ICryptoService interface {
	GetTransactions(ctx context.Context) error
}

type CryptoService struct {
}

func NewCryptoService() *CryptoService {
	return &CryptoService{}
}

func (c *CryptoService) GetTransactions(ctx context.Context) error {
	return nil
}
