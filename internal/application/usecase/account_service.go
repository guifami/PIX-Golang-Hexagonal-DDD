package usecase

import (
	"context"

	portin "go-api/internal/application/ports/in"
	portout "go-api/internal/application/ports/out"
	"go-api/internal/domain/account"
	"go-api/internal/infrastructure/logger"

	"go.uber.org/zap"
)

type AccountService struct {
	repo portout.AccountRepository
}

func NewAccountService(repo portout.AccountRepository) portin.AccountUseCase {
	return &AccountService{repo: repo}
}

func (s *AccountService) Create(ctx context.Context, ownerName, cpf string) (*account.Account, error) {
	log := logger.FromContext(ctx).With(zap.String("operation", "account.create"), zap.String("cpf", cpf))

	log.Info("creating account")

	_, err := s.repo.FindByCPF(ctx, cpf)
	if err == nil {
		log.Warn("CPF already in use")
		return nil, account.ErrCPFAlreadyInUse
	}

	acc, err := account.NewAccount(ownerName, cpf)
	if err != nil {
		log.Warn("domain validation failed", zap.Error(err))
		return nil, err
	}

	if err := s.repo.Create(ctx, acc); err != nil {
		log.Error("failed to persist account", zap.Error(err))
		return nil, err
	}

	log.Info("account created", zap.String("account_id", acc.ID))
	return acc, nil
}

func (s *AccountService) GetByID(ctx context.Context, id string) (*account.Account, error) {
	log := logger.FromContext(ctx).With(zap.String("operation", "account.get"), zap.String("account_id", id))

	acc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		log.Info("account not found", zap.Error(err))
		return nil, err
	}

	return acc, nil
}

func (s *AccountService) GetBalance(ctx context.Context, id string) (account.Money, error) {
	log := logger.FromContext(ctx).With(zap.String("operation", "account.balance"), zap.String("account_id", id))

	acc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		log.Info("account not found", zap.Error(err))
		return 0, err
	}

	log.Info("balance retrieved", zap.Int64("balance_cents", acc.Balance.Cents()))
	return acc.Balance, nil
}

func (s *AccountService) Deposit(ctx context.Context, id string, amountCents int64) (*account.Account, error) {
	log := logger.FromContext(ctx).With(zap.String("operation", "account.deposit"), zap.String("account_id", id))

	amount, err := account.NewMoney(amountCents)
	if err != nil {
		return nil, err
	}
	if !amount.IsPositive() {
		return nil, account.ErrInvalidAmount
	}

	acc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		log.Info("account not found", zap.Error(err))
		return nil, err
	}

	if err := acc.Credit(amount); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, acc); err != nil {
		log.Error("failed to update account", zap.Error(err))
		return nil, err
	}

	log.Info("deposit completed", zap.Int64("amount_cents", amountCents))
	return acc, nil
}
