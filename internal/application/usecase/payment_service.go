package usecase

import (
	"context"

	portin "go-api/internal/application/ports/in"
	portout "go-api/internal/application/ports/out"
	"go-api/internal/domain/account"
	"go-api/internal/domain/transaction"
	"go-api/internal/infrastructure/logger"

	"go.uber.org/zap"
)

type PaymentService struct {
	txRepo      portout.TransactionRepository
	accountRepo portout.AccountRepository
	keyRepo     portout.PixKeyRepository
	publisher   portout.EventPublisher
}

func NewPaymentService(
	txRepo portout.TransactionRepository,
	accountRepo portout.AccountRepository,
	keyRepo portout.PixKeyRepository,
	publisher portout.EventPublisher,
) portin.PaymentUseCase {
	return &PaymentService{
		txRepo:      txRepo,
		accountRepo: accountRepo,
		keyRepo:     keyRepo,
		publisher:   publisher,
	}
}

func (s *PaymentService) Initiate(ctx context.Context, payerAccountID, receiverKey, description string, amountCents int64) (*transaction.PixTransaction, error) {
	log := logger.FromContext(ctx).With(
		zap.String("operation", "payment.initiate"),
		zap.String("payer_account_id", payerAccountID),
		zap.String("receiver_key", receiverKey),
	)

	log.Info("initiating pix payment")

	amount := account.Money(amountCents)
	if !amount.IsPositive() {
		return nil, transaction.ErrInvalidAmount
	}

	payer, err := s.accountRepo.FindByID(ctx, payerAccountID)
	if err != nil {
		log.Info("payer account not found", zap.Error(err))
		return nil, account.ErrAccountNotFound
	}
	if !payer.IsActive() {
		return nil, account.ErrAccountInactive
	}

	receiverKeyEntity, err := s.keyRepo.FindByValue(ctx, receiverKey)
	if err != nil {
		log.Info("receiver key not found", zap.Error(err))
		return nil, err
	}

	if payer.ID == receiverKeyEntity.AccountID {
		return nil, transaction.ErrSameAccount
	}

	if payer.Balance.Cents() < amountCents {
		log.Warn("insufficient funds",
			zap.Int64("balance", payer.Balance.Cents()),
			zap.Int64("amount", amountCents),
		)
		return nil, account.ErrInsufficientFunds
	}

	// amount.IsPositive() is guaranteed above — NewTransaction cannot fail here.
	tx, _ := transaction.NewTransaction(payerAccountID, receiverKey, description, amount)

	if err := s.txRepo.Create(ctx, tx); err != nil {
		log.Error("failed to persist transaction", zap.Error(err))
		return nil, err
	}

	if err := s.publisher.PublishPaymentInitiated(ctx, tx); err != nil {
		log.Error("failed to publish payment initiated event", zap.Error(err))
	}

	log.Info("payment initiated", zap.String("transaction_id", tx.ID))
	return tx, nil
}

func (s *PaymentService) Process(ctx context.Context, transactionID string) error {
	log := logger.FromContext(ctx).With(
		zap.String("operation", "payment.process"),
		zap.String("transaction_id", transactionID),
	)

	log.Info("processing pix payment")

	tx, err := s.txRepo.FindByID(ctx, transactionID)
	if err != nil {
		log.Info("transaction not found", zap.Error(err))
		return err
	}

	if err := tx.StartProcessing(); err != nil {
		log.Warn("transaction already processed", zap.Error(err))
		return err
	}

	if err := s.txRepo.Update(ctx, tx); err != nil {
		log.Error("failed to update transaction to processing", zap.Error(err))
		return err
	}

	receiverKey, err := s.keyRepo.FindByValue(ctx, tx.ReceiverKey)
	if err != nil {
		reason := "receiver key not found"
		_ = s.failTransaction(ctx, tx, reason, log)
		return err
	}

	payer, err := s.accountRepo.FindByID(ctx, tx.PayerAccountID)
	if err != nil {
		reason := "payer account not found"
		_ = s.failTransaction(ctx, tx, reason, log)
		return err
	}

	receiver, err := s.accountRepo.FindByID(ctx, receiverKey.AccountID)
	if err != nil {
		reason := "receiver account not found"
		_ = s.failTransaction(ctx, tx, reason, log)
		return err
	}

	if err := payer.Debit(tx.Amount); err != nil {
		reason := "insufficient funds at processing time"
		_ = s.failTransaction(ctx, tx, reason, log)
		return err
	}

	if err := receiver.Credit(tx.Amount); err != nil {
		reason := "failed to credit receiver"
		_ = s.failTransaction(ctx, tx, reason, log)
		return err
	}

	if err := s.accountRepo.Update(ctx, payer); err != nil {
		log.Error("failed to debit payer", zap.Error(err))
		_ = s.failTransaction(ctx, tx, "database error on debit", log)
		return err
	}

	if err := s.accountRepo.Update(ctx, receiver); err != nil {
		log.Error("failed to credit receiver", zap.Error(err))
		_ = s.failTransaction(ctx, tx, "database error on credit", log)
		return err
	}

	if err := tx.Complete(receiver.ID); err != nil {
		log.Error("failed to complete transaction", zap.Error(err))
		return err
	}

	if err := s.txRepo.Update(ctx, tx); err != nil {
		log.Error("failed to persist completed transaction", zap.Error(err))
		return err
	}

	if err := s.publisher.PublishPaymentCompleted(ctx, tx); err != nil {
		log.Error("failed to publish payment completed event", zap.Error(err))
	}

	log.Info("payment processed successfully",
		zap.String("payer_id", payer.ID),
		zap.String("receiver_id", receiver.ID),
		zap.Int64("amount_cents", tx.Amount.Cents()),
	)
	return nil
}

func (s *PaymentService) GetByID(ctx context.Context, id string) (*transaction.PixTransaction, error) {
	log := logger.FromContext(ctx).With(zap.String("operation", "payment.get"), zap.String("transaction_id", id))

	tx, err := s.txRepo.FindByID(ctx, id)
	if err != nil {
		log.Info("transaction not found", zap.Error(err))
		return nil, err
	}

	return tx, nil
}

func (s *PaymentService) ListByAccount(ctx context.Context, accountID string) ([]transaction.PixTransaction, error) {
	log := logger.FromContext(ctx).With(zap.String("operation", "payment.list"), zap.String("account_id", accountID))

	txs, err := s.txRepo.FindByPayerAccountID(ctx, accountID)
	if err != nil {
		log.Error("failed to list transactions", zap.Error(err))
		return nil, err
	}

	log.Info("transactions listed", zap.Int("count", len(txs)))
	return txs, nil
}

func (s *PaymentService) failTransaction(ctx context.Context, tx *transaction.PixTransaction, reason string, log *zap.Logger) error {
	_ = tx.Fail(reason)
	if err := s.txRepo.Update(ctx, tx); err != nil {
		log.Error("failed to persist failed transaction", zap.Error(err))
		return err
	}
	if err := s.publisher.PublishPaymentFailed(ctx, tx); err != nil {
		log.Error("failed to publish payment failed event", zap.Error(err))
	}
	return nil
}
