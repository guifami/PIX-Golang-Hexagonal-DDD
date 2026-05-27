package out

import (
	"context"

	"go-api/internal/domain/transaction"
)

type EventPublisher interface {
	PublishPaymentInitiated(ctx context.Context, tx *transaction.PixTransaction) error
	PublishPaymentCompleted(ctx context.Context, tx *transaction.PixTransaction) error
	PublishPaymentFailed(ctx context.Context, tx *transaction.PixTransaction) error
}
