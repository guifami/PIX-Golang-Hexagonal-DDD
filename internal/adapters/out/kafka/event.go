package kafka

import "time"

const (
	TopicPaymentInitiated = "pix.payment.initiated"
	TopicPaymentCompleted = "pix.payment.completed"
	TopicPaymentFailed    = "pix.payment.failed"
)

// PaymentEvent is the message schema published to all PIX payment topics.
type PaymentEvent struct {
	TransactionID  string    `json:"transaction_id"`
	PayerAccountID string    `json:"payer_account_id"`
	ReceiverKey    string    `json:"receiver_key"`
	AmountCents    int64     `json:"amount_cents"`
	Status         string    `json:"status"`
	OccurredAt     time.Time `json:"occurred_at"`
	FailureReason  *string   `json:"failure_reason,omitempty"`
}
