package kafka

import (
	"context"
	"encoding/json"

	portin "go-api/internal/application/ports/in"
	kafkaout "go-api/internal/adapters/out/kafka"
	"go-api/internal/infrastructure/logger"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type PaymentConsumer struct {
	reader  *kafka.Reader
	useCase portin.PaymentUseCase
}

func NewPaymentConsumer(reader *kafka.Reader, useCase portin.PaymentUseCase) *PaymentConsumer {
	return &PaymentConsumer{reader: reader, useCase: useCase}
}

func (c *PaymentConsumer) Start(ctx context.Context) {
	log := logger.Log.With(zap.String("consumer", "payment.process"))
	log.Info("payment consumer started", zap.String("topic", kafkaout.TopicPaymentInitiated))

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Info("payment consumer stopped")
				return
			}
			log.Error("failed to read message", zap.Error(err))
			continue
		}

		log.Info("message received",
			zap.String("transaction_id", string(msg.Key)),
			zap.Int64("offset", msg.Offset),
		)

		var event kafkaout.PaymentEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Error("failed to unmarshal event", zap.Error(err))
			continue
		}

		processCtx := logger.WithContext(ctx, log.With(zap.String("transaction_id", event.TransactionID)))
		if err := c.useCase.Process(processCtx, event.TransactionID); err != nil {
			log.Error("failed to process payment",
				zap.String("transaction_id", event.TransactionID),
				zap.Error(err),
			)
		}
	}
}

func (c *PaymentConsumer) Close() error {
	return c.reader.Close()
}
