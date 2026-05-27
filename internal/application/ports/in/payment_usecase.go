package in

import (
	"context"

	"go-api/internal/domain/transaction"
)

type PaymentUseCase interface {
	Initiate(ctx context.Context, payerAccountID, receiverKey, description string, amountCents int64) (*transaction.PixTransaction, error)
	Process(ctx context.Context, transactionID string) error
	GetByID(ctx context.Context, id string) (*transaction.PixTransaction, error)
	ListByAccount(ctx context.Context, accountID string) ([]transaction.PixTransaction, error)
}
