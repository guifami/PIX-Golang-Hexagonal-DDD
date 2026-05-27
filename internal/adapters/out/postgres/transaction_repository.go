package postgres

import (
	"context"
	"database/sql"
	"errors"

	portout "go-api/internal/application/ports/out"
	"go-api/internal/domain/account"
	"go-api/internal/domain/transaction"
	"go-api/internal/infrastructure/logger"

	"go.uber.org/zap"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) portout.TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, tx *transaction.PixTransaction) error {
	log := logger.FromContext(ctx).With(zap.String("repo", "transaction.create"))

	const q = `
		INSERT INTO pix_transactions
			(id, payer_account_id, receiver_key, receiver_account_id, amount_cents, status, description, initiated_at, completed_at, failure_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	var receiverAccountID *string
	if tx.ReceiverAccountID != "" {
		receiverAccountID = &tx.ReceiverAccountID
	}

	_, err := r.db.ExecContext(ctx, q,
		tx.ID, tx.PayerAccountID, tx.ReceiverKey,
		receiverAccountID, tx.Amount.Cents(),
		string(tx.Status), tx.Description,
		tx.InitiatedAt, tx.CompletedAt, tx.FailureReason,
	)
	if err != nil {
		log.Error("insert failed", zap.Error(err))
		return err
	}
	return nil
}

func (r *TransactionRepository) FindByID(ctx context.Context, id string) (*transaction.PixTransaction, error) {
	const q = `
		SELECT id, payer_account_id, receiver_key, receiver_account_id,
		       amount_cents, status, description, initiated_at, completed_at, failure_reason
		FROM pix_transactions WHERE id = $1`

	return r.scanTransaction(r.db.QueryRowContext(ctx, q, id))
}

func (r *TransactionRepository) FindByPayerAccountID(ctx context.Context, accountID string) ([]transaction.PixTransaction, error) {
	const q = `
		SELECT id, payer_account_id, receiver_key, receiver_account_id,
		       amount_cents, status, description, initiated_at, completed_at, failure_reason
		FROM pix_transactions WHERE payer_account_id = $1 ORDER BY initiated_at DESC`

	rows, err := r.db.QueryContext(ctx, q, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []transaction.PixTransaction
	for rows.Next() {
		tx, err := r.scanTransactionRow(rows)
		if err != nil {
			return nil, err
		}
		txs = append(txs, *tx)
	}
	return txs, rows.Err()
}

func (r *TransactionRepository) Update(ctx context.Context, tx *transaction.PixTransaction) error {
	log := logger.FromContext(ctx).With(zap.String("repo", "transaction.update"), zap.String("id", tx.ID))

	const q = `
		UPDATE pix_transactions
		SET receiver_account_id = $1, status = $2, completed_at = $3, failure_reason = $4
		WHERE id = $5`

	var receiverAccountID *string
	if tx.ReceiverAccountID != "" {
		receiverAccountID = &tx.ReceiverAccountID
	}

	result, err := r.db.ExecContext(ctx, q,
		receiverAccountID, string(tx.Status), tx.CompletedAt, tx.FailureReason, tx.ID,
	)
	if err != nil {
		log.Error("update failed", zap.Error(err))
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return transaction.ErrTransactionNotFound
	}
	return nil
}

func (r *TransactionRepository) scanTransaction(row *sql.Row) (*transaction.PixTransaction, error) {
	var tx transaction.PixTransaction
	var amountCents int64
	var statusStr string
	var receiverAccountID sql.NullString

	err := row.Scan(
		&tx.ID, &tx.PayerAccountID, &tx.ReceiverKey, &receiverAccountID,
		&amountCents, &statusStr, &tx.Description,
		&tx.InitiatedAt, &tx.CompletedAt, &tx.FailureReason,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transaction.ErrTransactionNotFound
		}
		return nil, err
	}

	tx.Amount = account.Money(amountCents)
	tx.Status = transaction.TransactionStatus(statusStr)
	if receiverAccountID.Valid {
		tx.ReceiverAccountID = receiverAccountID.String
	}
	return &tx, nil
}

func (r *TransactionRepository) scanTransactionRow(rows *sql.Rows) (*transaction.PixTransaction, error) {
	var tx transaction.PixTransaction
	var amountCents int64
	var statusStr string
	var receiverAccountID sql.NullString

	err := rows.Scan(
		&tx.ID, &tx.PayerAccountID, &tx.ReceiverKey, &receiverAccountID,
		&amountCents, &statusStr, &tx.Description,
		&tx.InitiatedAt, &tx.CompletedAt, &tx.FailureReason,
	)
	if err != nil {
		return nil, err
	}

	tx.Amount = account.Money(amountCents)
	tx.Status = transaction.TransactionStatus(statusStr)
	if receiverAccountID.Valid {
		tx.ReceiverAccountID = receiverAccountID.String
	}
	return &tx, nil
}
