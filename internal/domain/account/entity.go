package account

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID        string
	OwnerName string
	CPF       CPF
	Balance   Money
	Status    AccountStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewAccount(ownerName, cpf string) (*Account, error) {
	name := strings.TrimSpace(ownerName)
	if name == "" {
		return nil, ErrInvalidName
	}

	validCPF, err := NewCPF(cpf)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &Account{
		ID:        uuid.NewString(),
		OwnerName: name,
		CPF:       validCPF,
		Balance:   Money(0),
		Status:    Active,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (a *Account) Debit(amount Money) error {
	if a.Status != Active {
		return ErrAccountInactive
	}
	newBalance, err := a.Balance.Sub(amount)
	if err != nil {
		return err
	}
	a.Balance = newBalance
	a.UpdatedAt = time.Now()
	return nil
}

func (a *Account) Credit(amount Money) error {
	if a.Status != Active {
		return ErrAccountInactive
	}
	a.Balance = a.Balance.Add(amount)
	a.UpdatedAt = time.Now()
	return nil
}

func (a *Account) Deactivate() {
	a.Status = Inactive
	a.UpdatedAt = time.Now()
}

func (a *Account) Activate() {
	a.Status = Active
	a.UpdatedAt = time.Now()
}

func (a *Account) IsActive() bool {
	return a.Status == Active
}
