package transaction

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"go-api/internal/domain/account"
)

type PixTransaction struct {
	ID                 string
	PayerAccountID     string
	ReceiverKey        string
	ReceiverAccountID  string
	Amount             account.Money
	Status             TransactionStatus
	Description        string
	InitiatedAt        time.Time
	CompletedAt        *time.Time
	FailureReason      *string
}

func NewTransaction(payerAccountID, receiverKey, description string, amount account.Money) (*PixTransaction, error) {
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
	}

	return &PixTransaction{
		ID:             uuid.NewString(),
		PayerAccountID: payerAccountID,
		ReceiverKey:    receiverKey,
		Amount:         amount,
		Status:         Pending,
		Description:    strings.TrimSpace(description),
		InitiatedAt:    time.Now(),
	}, nil
}

func (t *PixTransaction) StartProcessing() error {
	if t.Status != Pending {
		return ErrTransactionAlreadyProcessed
	}
	t.Status = Processing
	return nil
}

func (t *PixTransaction) Complete(receiverAccountID string) error {
	if t.Status != Processing {
		return ErrInvalidTransactionStatus
	}
	now := time.Now()
	t.Status = Completed
	t.ReceiverAccountID = receiverAccountID
	t.CompletedAt = &now
	return nil
}

func (t *PixTransaction) Fail(reason string) error {
	if t.Status != Processing && t.Status != Pending {
		return ErrInvalidTransactionStatus
	}
	now := time.Now()
	t.Status = Failed
	t.CompletedAt = &now
	t.FailureReason = &reason
	return nil
}

func (t *PixTransaction) IsFinal() bool {
	return t.Status == Completed || t.Status == Failed
}
