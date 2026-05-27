package kafka

import (
	"context"
	"encoding/json"

	kafkaout "go-api/internal/adapters/out/kafka"
	"go-api/internal/infrastructure/logger"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// ResultLogger consumes pix.payment.completed and pix.payment.failed,
// logging the outcome to demonstrate the end-to-end Kafka pipeline.
type ResultLogger struct {
	completedReader *kafka.Reader
	failedReader    *kafka.Reader
}

func NewResultLogger(completedReader, failedReader *kafka.Reader) *ResultLogger {
	return &ResultLogger{
		completedReader: completedReader,
		failedReader:    failedReader,
	}
}

func (r *ResultLogger) Start(ctx context.Context) {
	go r.consume(ctx, r.completedReader, "completed")
	go r.consume(ctx, r.failedReader, "failed")
}

func (r *ResultLogger) consume(ctx context.Context, reader *kafka.Reader, outcome string) {
	log := logger.Log.With(zap.String("consumer", "result.logger"), zap.String("outcome", outcome))
	log.Info("result logger started", zap.String("topic", reader.Config().Topic))

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Info("result logger stopped", zap.String("outcome", outcome))
				return
			}
			log.Error("failed to read message", zap.Error(err))
			continue
		}

		var event kafkaout.PaymentEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Error("failed to unmarshal event", zap.Error(err))
			continue
		}

		fields := []zap.Field{
			zap.String("transaction_id", event.TransactionID),
			zap.String("payer_account_id", event.PayerAccountID),
			zap.String("receiver_key", event.ReceiverKey),
			zap.Int64("amount_cents", event.AmountCents),
			zap.Time("occurred_at", event.OccurredAt),
		}

		if event.FailureReason != nil {
			fields = append(fields, zap.String("failure_reason", *event.FailureReason))
		}

		if outcome == "completed" {
			log.Info("PIX payment completed successfully", fields...)
		} else {
			log.Warn("PIX payment failed", fields...)
		}
	}
}

func (r *ResultLogger) Close() {
	_ = r.completedReader.Close()
	_ = r.failedReader.Close()
}
