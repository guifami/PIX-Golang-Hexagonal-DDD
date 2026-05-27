package usecase

import (
	"context"

	portin "go-api/internal/application/ports/in"
	portout "go-api/internal/application/ports/out"
	"go-api/internal/domain/account"
	"go-api/internal/domain/pixkey"
	"go-api/internal/infrastructure/logger"

	"go.uber.org/zap"
)

type PixKeyService struct {
	keyRepo     portout.PixKeyRepository
	accountRepo portout.AccountRepository
}

func NewPixKeyService(keyRepo portout.PixKeyRepository, accountRepo portout.AccountRepository) portin.PixKeyUseCase {
	return &PixKeyService{keyRepo: keyRepo, accountRepo: accountRepo}
}

func (s *PixKeyService) Register(ctx context.Context, accountID, keyType, keyValue string) (*pixkey.PixKey, error) {
	log := logger.FromContext(ctx).With(
		zap.String("operation", "pixkey.register"),
		zap.String("account_id", accountID),
		zap.String("key_type", keyType),
	)

	log.Info("registering pix key")

	acc, err := s.accountRepo.FindByID(ctx, accountID)
	if err != nil {
		log.Info("account not found", zap.Error(err))
		return nil, account.ErrAccountNotFound
	}
	if !acc.IsActive() {
		return nil, account.ErrAccountInactive
	}

	count, err := s.keyRepo.CountByAccountID(ctx, accountID)
	if err != nil {
		log.Error("failed to count keys", zap.Error(err))
		return nil, err
	}
	if count >= 5 {
		log.Warn("key limit reached", zap.Int("count", count))
		return nil, pixkey.ErrKeyLimitReached
	}

	existing, err := s.keyRepo.FindByValue(ctx, keyValue)
	if err == nil && existing != nil {
		log.Warn("key already registered", zap.String("key_value", keyValue))
		return nil, pixkey.ErrKeyAlreadyExists
	}

	key, err := pixkey.NewPixKey(accountID, keyType, keyValue)
	if err != nil {
		log.Warn("domain validation failed", zap.Error(err))
		return nil, err
	}

	if err := s.keyRepo.Create(ctx, key); err != nil {
		log.Error("failed to persist key", zap.Error(err))
		return nil, err
	}

	log.Info("pix key registered", zap.String("key_id", key.ID))
	return key, nil
}

func (s *PixKeyService) GetByValue(ctx context.Context, keyValue string) (*pixkey.PixKey, error) {
	log := logger.FromContext(ctx).With(
		zap.String("operation", "pixkey.get"),
		zap.String("key_value", keyValue),
	)

	key, err := s.keyRepo.FindByValue(ctx, keyValue)
	if err != nil {
		log.Info("key not found", zap.Error(err))
		return nil, err
	}

	return key, nil
}

func (s *PixKeyService) Delete(ctx context.Context, keyValue, accountID string) error {
	log := logger.FromContext(ctx).With(
		zap.String("operation", "pixkey.delete"),
		zap.String("key_value", keyValue),
		zap.String("account_id", accountID),
	)

	key, err := s.keyRepo.FindByValue(ctx, keyValue)
	if err != nil {
		log.Info("key not found", zap.Error(err))
		return err
	}

	if !key.BelongsTo(accountID) {
		log.Warn("unauthorized key deletion attempt")
		return pixkey.ErrUnauthorized
	}

	if err := s.keyRepo.Delete(ctx, keyValue); err != nil {
		log.Error("failed to delete key", zap.Error(err))
		return err
	}

	log.Info("pix key deleted")
	return nil
}

func (s *PixKeyService) ListByAccount(ctx context.Context, accountID string) ([]pixkey.PixKey, error) {
	log := logger.FromContext(ctx).With(
		zap.String("operation", "pixkey.list"),
		zap.String("account_id", accountID),
	)

	keys, err := s.keyRepo.FindByAccountID(ctx, accountID)
	if err != nil {
		log.Error("failed to list keys", zap.Error(err))
		return nil, err
	}

	log.Info("keys listed", zap.Int("count", len(keys)))
	return keys, nil
}
