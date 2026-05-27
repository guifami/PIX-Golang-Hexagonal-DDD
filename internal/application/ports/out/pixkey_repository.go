package out

import (
	"context"

	"go-api/internal/domain/pixkey"
)

type PixKeyRepository interface {
	Create(ctx context.Context, key *pixkey.PixKey) error
	FindByValue(ctx context.Context, keyValue string) (*pixkey.PixKey, error)
	FindByAccountID(ctx context.Context, accountID string) ([]pixkey.PixKey, error)
	CountByAccountID(ctx context.Context, accountID string) (int, error)
	Delete(ctx context.Context, keyValue string) error
}
