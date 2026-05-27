package in

import (
	"context"

	"go-api/internal/domain/account"
)

type AccountUseCase interface {
	Create(ctx context.Context, ownerName, cpf string) (*account.Account, error)
	GetByID(ctx context.Context, id string) (*account.Account, error)
	GetBalance(ctx context.Context, id string) (account.Money, error)
	Deposit(ctx context.Context, id string, amountCents int64) (*account.Account, error)
}
