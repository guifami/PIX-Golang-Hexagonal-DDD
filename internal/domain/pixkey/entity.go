package pixkey

import (
	"time"

	"github.com/google/uuid"
)

type PixKey struct {
	ID        string
	AccountID string
	Type      KeyType
	Value     string
	Status    KeyStatus
	CreatedAt time.Time
}

func NewPixKey(accountID, keyType, keyValue string) (*PixKey, error) {
	kt, err := ParseKeyType(keyType)
	if err != nil {
		return nil, err
	}

	if err := ValidateKeyValue(kt, keyValue); err != nil {
		return nil, err
	}

	return &PixKey{
		ID:        uuid.NewString(),
		AccountID: accountID,
		Type:      kt,
		Value:     keyValue,
		Status:    Active,
		CreatedAt: time.Now(),
	}, nil
}

func (k *PixKey) BelongsTo(accountID string) bool {
	return k.AccountID == accountID
}

func (k *PixKey) IsActive() bool {
	return k.Status == Active
}

func (k *PixKey) Deactivate() {
	k.Status = Inactive
}
