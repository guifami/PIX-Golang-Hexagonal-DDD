package out

import (
	"context"

	"go-api/internal/domain/account"
)

type AccountRepository interface {
	Create(ctx context.Context, acc *account.Account) error
	FindByID(ctx context.Context, id string) (*account.Account, error)
	FindByCPF(ctx context.Context, cpf string) (*account.Account, error)
	Update(ctx context.Context, acc *account.Account) error
}
