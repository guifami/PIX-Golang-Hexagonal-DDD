package usecase_test

import (
	"context"
	"testing"

	"go-api/internal/application/usecase"
	"go-api/internal/domain/account"
)

func TestAccountService_Create(t *testing.T) {
	t.Run("creates account successfully", func(t *testing.T) {
		repo := newMockAccountRepo()
		svc := usecase.NewAccountService(repo)

		acc, err := svc.Create(context.Background(), "João Silva", "11144477735")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if acc.ID == "" {
			t.Error("ID should not be empty")
		}
		if acc.OwnerName != "João Silva" {
			t.Errorf("OwnerName = %q", acc.OwnerName)
		}
	})

	t.Run("rejects duplicate CPF", func(t *testing.T) {
		repo := newMockAccountRepo()
		svc := usecase.NewAccountService(repo)

		_, _ = svc.Create(context.Background(), "First", "11144477735")
		_, err := svc.Create(context.Background(), "Second", "11144477735")
		if err != account.ErrCPFAlreadyInUse {
			t.Errorf("expected ErrCPFAlreadyInUse, got %v", err)
		}
	})

	t.Run("rejects invalid CPF", func(t *testing.T) {
		repo := newMockAccountRepo()
		svc := usecase.NewAccountService(repo)

		_, err := svc.Create(context.Background(), "Test", "00000000000")
		if err == nil {
			t.Error("expected error for invalid CPF")
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		repo := newMockAccountRepo()
		svc := usecase.NewAccountService(repo)

		_, err := svc.Create(context.Background(), "", "11144477735")
		if err != account.ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := newMockAccountRepo()
		repo.createErr = errDB
		svc := usecase.NewAccountService(repo)

		_, err := svc.Create(context.Background(), "João", "11144477735")
		if err != errDB {
			t.Errorf("expected errDB, got %v", err)
		}
	})
}

func TestAccountService_GetByID(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo := newMockAccountRepo()
		acc := makeAccount("acc-1", "11144477735", 500)
		repo.accounts["acc-1"] = acc
		svc := usecase.NewAccountService(repo)

		got, err := svc.GetByID(context.Background(), "acc-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "acc-1" {
			t.Errorf("ID = %q", got.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockAccountRepo()
		svc := usecase.NewAccountService(repo)

		_, err := svc.GetByID(context.Background(), "missing")
		if err != account.ErrAccountNotFound {
			t.Errorf("expected ErrAccountNotFound, got %v", err)
		}
	})
}

func TestAccountService_GetBalance(t *testing.T) {
	t.Run("returns balance", func(t *testing.T) {
		repo := newMockAccountRepo()
		repo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 9999)
		svc := usecase.NewAccountService(repo)

		bal, err := svc.GetBalance(context.Background(), "acc-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bal.Cents() != 9999 {
			t.Errorf("balance = %d, want 9999", bal.Cents())
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockAccountRepo()
		svc := usecase.NewAccountService(repo)

		_, err := svc.GetBalance(context.Background(), "missing")
		if err != account.ErrAccountNotFound {
			t.Errorf("expected ErrAccountNotFound, got %v", err)
		}
	})
}

func TestAccountService_Deposit(t *testing.T) {
	t.Run("successful deposit", func(t *testing.T) {
		repo := newMockAccountRepo()
		repo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 1000)
		svc := usecase.NewAccountService(repo)

		acc, err := svc.Deposit(context.Background(), "acc-1", 500)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if acc.Balance.Cents() != 1500 {
			t.Errorf("balance after deposit = %d, want 1500", acc.Balance.Cents())
		}
	})

	t.Run("zero amount rejected", func(t *testing.T) {
		repo := newMockAccountRepo()
		repo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 0)
		svc := usecase.NewAccountService(repo)

		_, err := svc.Deposit(context.Background(), "acc-1", 0)
		if err != account.ErrInvalidAmount {
			t.Errorf("expected ErrInvalidAmount, got %v", err)
		}
	})

	t.Run("negative amount rejected", func(t *testing.T) {
		repo := newMockAccountRepo()
		svc := usecase.NewAccountService(repo)

		_, err := svc.Deposit(context.Background(), "acc-1", -100)
		if err != account.ErrInvalidAmount {
			t.Errorf("expected ErrInvalidAmount, got %v", err)
		}
	})

	t.Run("account not found", func(t *testing.T) {
		repo := newMockAccountRepo()
		svc := usecase.NewAccountService(repo)

		_, err := svc.Deposit(context.Background(), "missing", 100)
		if err != account.ErrAccountNotFound {
			t.Errorf("expected ErrAccountNotFound, got %v", err)
		}
	})

	t.Run("credit to inactive account rejected", func(t *testing.T) {
		repo := newMockAccountRepo()
		acc := makeAccount("acc-1", "11144477735", 0)
		acc.Deactivate()
		repo.accounts["acc-1"] = acc
		svc := usecase.NewAccountService(repo)

		_, err := svc.Deposit(context.Background(), "acc-1", 100)
		if err != account.ErrAccountInactive {
			t.Errorf("expected ErrAccountInactive, got %v", err)
		}
	})

	t.Run("propagates update error", func(t *testing.T) {
		repo := newMockAccountRepo()
		repo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 0)
		repo.updateErr = errDB
		svc := usecase.NewAccountService(repo)

		_, err := svc.Deposit(context.Background(), "acc-1", 100)
		if err != errDB {
			t.Errorf("expected errDB, got %v", err)
		}
	})
}
