package postgres

import (
	"context"
	"database/sql"
	"errors"

	portout "go-api/internal/application/ports/out"
	"go-api/internal/domain/pixkey"
	"go-api/internal/infrastructure/logger"

	"go.uber.org/zap"
)

type PixKeyRepository struct {
	db *sql.DB
}

func NewPixKeyRepository(db *sql.DB) portout.PixKeyRepository {
	return &PixKeyRepository{db: db}
}

func (r *PixKeyRepository) Create(ctx context.Context, key *pixkey.PixKey) error {
	log := logger.FromContext(ctx).With(zap.String("repo", "pixkey.create"))

	const q = `
		INSERT INTO pix_keys (id, account_id, key_type, key_value, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, q,
		key.ID, key.AccountID, string(key.Type), key.Value, string(key.Status), key.CreatedAt,
	)
	if err != nil {
		log.Error("insert failed", zap.Error(err))
		return err
	}
	return nil
}

func (r *PixKeyRepository) FindByValue(ctx context.Context, keyValue string) (*pixkey.PixKey, error) {
	const q = `
		SELECT id, account_id, key_type, key_value, status, created_at
		FROM pix_keys WHERE key_value = $1`

	return r.scanKey(r.db.QueryRowContext(ctx, q, keyValue))
}

func (r *PixKeyRepository) FindByAccountID(ctx context.Context, accountID string) ([]pixkey.PixKey, error) {
	const q = `
		SELECT id, account_id, key_type, key_value, status, created_at
		FROM pix_keys WHERE account_id = $1 ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []pixkey.PixKey
	for rows.Next() {
		var k pixkey.PixKey
		var kt, ks string
		if err := rows.Scan(&k.ID, &k.AccountID, &kt, &k.Value, &ks, &k.CreatedAt); err != nil {
			return nil, err
		}
		k.Type = pixkey.KeyType(kt)
		k.Status = pixkey.KeyStatus(ks)
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *PixKeyRepository) CountByAccountID(ctx context.Context, accountID string) (int, error) {
	const q = `SELECT COUNT(*) FROM pix_keys WHERE account_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, q, accountID).Scan(&count)
	return count, err
}

func (r *PixKeyRepository) Delete(ctx context.Context, keyValue string) error {
	log := logger.FromContext(ctx).With(zap.String("repo", "pixkey.delete"))

	const q = `DELETE FROM pix_keys WHERE key_value = $1`
	result, err := r.db.ExecContext(ctx, q, keyValue)
	if err != nil {
		log.Error("delete failed", zap.Error(err))
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return pixkey.ErrKeyNotFound
	}
	return nil
}

func (r *PixKeyRepository) scanKey(row *sql.Row) (*pixkey.PixKey, error) {
	var k pixkey.PixKey
	var kt, ks string
	err := row.Scan(&k.ID, &k.AccountID, &kt, &k.Value, &ks, &k.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pixkey.ErrKeyNotFound
		}
		return nil, err
	}
	k.Type = pixkey.KeyType(kt)
	k.Status = pixkey.KeyStatus(ks)
	return &k, nil
}
