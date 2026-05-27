package in

import (
	"context"

	"go-api/internal/domain/pixkey"
)

type PixKeyUseCase interface {
	Register(ctx context.Context, accountID, keyType, keyValue string) (*pixkey.PixKey, error)
	GetByValue(ctx context.Context, keyValue string) (*pixkey.PixKey, error)
	Delete(ctx context.Context, keyValue, accountID string) error
	ListByAccount(ctx context.Context, accountID string) ([]pixkey.PixKey, error)
}
