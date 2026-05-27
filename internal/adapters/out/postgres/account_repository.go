package postgres

import (
	"context"
	"database/sql"
	"errors"

	portout "go-api/internal/application/ports/out"
	"go-api/internal/domain/account"
	"go-api/internal/infrastructure/logger"

	"go.uber.org/zap"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) portout.AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, acc *account.Account) error {
	log := logger.FromContext(ctx).With(zap.String("repo", "account.create"))

	const q = `
		INSERT INTO accounts (id, owner_name, cpf, balance_cents, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, q,
		acc.ID, acc.OwnerName, acc.CPF.String(),
		acc.Balance.Cents(), string(acc.Status),
		acc.CreatedAt, acc.UpdatedAt,
	)
	if err != nil {
		log.Error("insert failed", zap.Error(err))
		return err
	}
	return nil
}

func (r *AccountRepository) FindByID(ctx context.Context, id string) (*account.Account, error) {
	const q = `
		SELECT id, owner_name, cpf, balance_cents, status, created_at, updated_at
		FROM accounts WHERE id = $1`

	return r.scanAccount(r.db.QueryRowContext(ctx, q, id))
}

func (r *AccountRepository) FindByCPF(ctx context.Context, cpf string) (*account.Account, error) {
	const q = `
		SELECT id, owner_name, cpf, balance_cents, status, created_at, updated_at
		FROM accounts WHERE cpf = $1`

	return r.scanAccount(r.db.QueryRowContext(ctx, q, cpf))
}

func (r *AccountRepository) Update(ctx context.Context, acc *account.Account) error {
	log := logger.FromContext(ctx).With(zap.String("repo", "account.update"), zap.String("id", acc.ID))

	const q = `
		UPDATE accounts
		SET balance_cents = $1, status = $2, updated_at = $3
		WHERE id = $4`

	result, err := r.db.ExecContext(ctx, q,
		acc.Balance.Cents(), string(acc.Status), acc.UpdatedAt, acc.ID,
	)
	if err != nil {
		log.Error("update failed", zap.Error(err))
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return account.ErrAccountNotFound
	}
	return nil
}

func (r *AccountRepository) scanAccount(row *sql.Row) (*account.Account, error) {
	var acc account.Account
	var cpfStr string
	var balanceCents int64
	var statusStr string

	err := row.Scan(
		&acc.ID, &acc.OwnerName, &cpfStr,
		&balanceCents, &statusStr,
		&acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, account.ErrAccountNotFound
		}
		return nil, err
	}

	acc.CPF = account.CPF(cpfStr)
	acc.Balance = account.Money(balanceCents)
	acc.Status = account.AccountStatus(statusStr)
	return &acc, nil
}
