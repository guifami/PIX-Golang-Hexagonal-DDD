package out

import (
	"context"

	"go-api/internal/domain/transaction"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx *transaction.PixTransaction) error
	FindByID(ctx context.Context, id string) (*transaction.PixTransaction, error)
	FindByPayerAccountID(ctx context.Context, accountID string) ([]transaction.PixTransaction, error)
	Update(ctx context.Context, tx *transaction.PixTransaction) error
}
