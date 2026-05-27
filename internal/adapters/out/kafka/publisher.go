package kafka

import (
	"context"
	"encoding/json"
	"time"

	portout "go-api/internal/application/ports/out"
	"go-api/internal/domain/transaction"
	"go-api/internal/infrastructure/logger"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// KafkaEventPublisher implements portout.EventPublisher using kafka-go writers.
type KafkaEventPublisher struct {
	initiatedWriter *kafka.Writer
	completedWriter *kafka.Writer
	failedWriter    *kafka.Writer
}

func NewKafkaEventPublisher(broker string) portout.EventPublisher {
	return &KafkaEventPublisher{
		initiatedWriter: newWriter(broker, TopicPaymentInitiated),
		completedWriter: newWriter(broker, TopicPaymentCompleted),
		failedWriter:    newWriter(broker, TopicPaymentFailed),
	}
}

func newWriter(broker, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(broker),
		Topic:                  topic,
		AllowAutoTopicCreation: true,
		Balancer:               &kafka.LeastBytes{},
	}
}

func (p *KafkaEventPublisher) PublishPaymentInitiated(ctx context.Context, tx *transaction.PixTransaction) error {
	return p.publish(ctx, p.initiatedWriter, tx)
}

func (p *KafkaEventPublisher) PublishPaymentCompleted(ctx context.Context, tx *transaction.PixTransaction) error {
	return p.publish(ctx, p.completedWriter, tx)
}

func (p *KafkaEventPublisher) PublishPaymentFailed(ctx context.Context, tx *transaction.PixTransaction) error {
	return p.publish(ctx, p.failedWriter, tx)
}

func (p *KafkaEventPublisher) publish(ctx context.Context, w *kafka.Writer, tx *transaction.PixTransaction) error {
	log := logger.FromContext(ctx).With(
		zap.String("topic", w.Topic),
		zap.String("transaction_id", tx.ID),
	)

	event := PaymentEvent{
		TransactionID:  tx.ID,
		PayerAccountID: tx.PayerAccountID,
		ReceiverKey:    tx.ReceiverKey,
		AmountCents:    tx.Amount.Cents(),
		Status:         string(tx.Status),
		OccurredAt:     time.Now(),
		FailureReason:  tx.FailureReason,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Error("failed to marshal event", zap.Error(err))
		return err
	}

	if err := w.WriteMessages(ctx, kafka.Message{Key: []byte(tx.ID), Value: payload}); err != nil {
		log.Error("failed to write kafka message", zap.Error(err))
		return err
	}

	log.Info("event published", zap.String("status", string(tx.Status)))
	return nil
}

func (p *KafkaEventPublisher) Close() {
	_ = p.initiatedWriter.Close()
	_ = p.completedWriter.Close()
	_ = p.failedWriter.Close()
}
