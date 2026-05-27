package usecase_test

import (
	"context"
	"testing"

	"go-api/internal/application/usecase"
	"go-api/internal/domain/account"
	"go-api/internal/domain/pixkey"
)

func TestPixKeyService_Register(t *testing.T) {
	t.Run("registers key successfully", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		accRepo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 0)
		keyRepo := newMockPixKeyRepo()
		svc := usecase.NewPixKeyService(keyRepo, accRepo)

		k, err := svc.Register(context.Background(), "acc-1", "EMAIL", "user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if k.ID == "" {
			t.Error("key ID should not be empty")
		}
		if k.AccountID != "acc-1" {
			t.Errorf("AccountID = %q", k.AccountID)
		}
	})

	t.Run("account not found", func(t *testing.T) {
		svc := usecase.NewPixKeyService(newMockPixKeyRepo(), newMockAccountRepo())
		_, err := svc.Register(context.Background(), "missing", "EMAIL", "a@b.com")
		if err != account.ErrAccountNotFound {
			t.Errorf("expected ErrAccountNotFound, got %v", err)
		}
	})

	t.Run("inactive account rejected", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		acc := makeAccount("acc-1", "11144477735", 0)
		acc.Deactivate()
		accRepo.accounts["acc-1"] = acc
		svc := usecase.NewPixKeyService(newMockPixKeyRepo(), accRepo)

		_, err := svc.Register(context.Background(), "acc-1", "EMAIL", "a@b.com")
		if err != account.ErrAccountInactive {
			t.Errorf("expected ErrAccountInactive, got %v", err)
		}
	})

	t.Run("key limit reached (5 keys)", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		accRepo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 0)
		keyRepo := newMockPixKeyRepo()
		keyRepo.count = 5
		svc := usecase.NewPixKeyService(keyRepo, accRepo)

		_, err := svc.Register(context.Background(), "acc-1", "EMAIL", "a@b.com")
		if err != pixkey.ErrKeyLimitReached {
			t.Errorf("expected ErrKeyLimitReached, got %v", err)
		}
	})

	t.Run("duplicate key rejected", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		accRepo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 0)
		keyRepo := newMockPixKeyRepo()
		k, _ := pixkey.NewPixKey("acc-1", "EMAIL", "dup@example.com")
		keyRepo.keys["dup@example.com"] = k
		svc := usecase.NewPixKeyService(keyRepo, accRepo)

		_, err := svc.Register(context.Background(), "acc-1", "EMAIL", "dup@example.com")
		if err != pixkey.ErrKeyAlreadyExists {
			t.Errorf("expected ErrKeyAlreadyExists, got %v", err)
		}
	})

	t.Run("invalid key type rejected", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		accRepo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 0)
		svc := usecase.NewPixKeyService(newMockPixKeyRepo(), accRepo)

		_, err := svc.Register(context.Background(), "acc-1", "UNKNOWN", "value")
		if err != pixkey.ErrInvalidKeyType {
			t.Errorf("expected ErrInvalidKeyType, got %v", err)
		}
	})

	t.Run("count error propagated", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		accRepo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 0)
		keyRepo := newMockPixKeyRepo()
		keyRepo.countErr = errDB
		svc := usecase.NewPixKeyService(keyRepo, accRepo)

		_, err := svc.Register(context.Background(), "acc-1", "EMAIL", "a@b.com")
		if err != errDB {
			t.Errorf("expected errDB, got %v", err)
		}
	})
}

func TestPixKeyService_GetByValue(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		keyRepo := newMockPixKeyRepo()
		k, _ := pixkey.NewPixKey("acc-1", "EMAIL", "a@b.com")
		keyRepo.keys["a@b.com"] = k
		svc := usecase.NewPixKeyService(keyRepo, newMockAccountRepo())

		got, err := svc.GetByValue(context.Background(), "a@b.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Value != "a@b.com" {
			t.Errorf("Value = %q", got.Value)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := usecase.NewPixKeyService(newMockPixKeyRepo(), newMockAccountRepo())
		_, err := svc.GetByValue(context.Background(), "missing")
		if err != pixkey.ErrKeyNotFound {
			t.Errorf("expected ErrKeyNotFound, got %v", err)
		}
	})
}

func TestPixKeyService_Delete(t *testing.T) {
	t.Run("deletes own key", func(t *testing.T) {
		keyRepo := newMockPixKeyRepo()
		k, _ := pixkey.NewPixKey("acc-1", "EMAIL", "a@b.com")
		keyRepo.keys["a@b.com"] = k
		svc := usecase.NewPixKeyService(keyRepo, newMockAccountRepo())

		if err := svc.Delete(context.Background(), "a@b.com", "acc-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, exists := keyRepo.keys["a@b.com"]; exists {
			t.Error("key should have been deleted")
		}
	})

	t.Run("unauthorized deletion", func(t *testing.T) {
		keyRepo := newMockPixKeyRepo()
		k, _ := pixkey.NewPixKey("acc-1", "EMAIL", "a@b.com")
		keyRepo.keys["a@b.com"] = k
		svc := usecase.NewPixKeyService(keyRepo, newMockAccountRepo())

		err := svc.Delete(context.Background(), "a@b.com", "acc-other")
		if err != pixkey.ErrUnauthorized {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("key not found", func(t *testing.T) {
		svc := usecase.NewPixKeyService(newMockPixKeyRepo(), newMockAccountRepo())
		err := svc.Delete(context.Background(), "missing", "acc-1")
		if err != pixkey.ErrKeyNotFound {
			t.Errorf("expected ErrKeyNotFound, got %v", err)
		}
	})
}

func TestPixKeyService_Delete_RepoError(t *testing.T) {
	keyRepo := newMockPixKeyRepo()
	k, _ := pixkey.NewPixKey("acc-1", "EMAIL", "a@b.com")
	keyRepo.keys["a@b.com"] = k
	keyRepo.deleteErr = errDB
	svc := usecase.NewPixKeyService(keyRepo, newMockAccountRepo())

	err := svc.Delete(context.Background(), "a@b.com", "acc-1")
	if err != errDB {
		t.Errorf("expected errDB, got %v", err)
	}
}

func TestPixKeyService_Register_CreateError(t *testing.T) {
	accRepo := newMockAccountRepo()
	accRepo.accounts["acc-1"] = makeAccount("acc-1", "11144477735", 0)
	keyRepo := newMockPixKeyRepo()
	keyRepo.createErr = errDB
	svc := usecase.NewPixKeyService(keyRepo, accRepo)

	_, err := svc.Register(context.Background(), "acc-1", "EMAIL", "x@y.com")
	if err != errDB {
		t.Errorf("expected errDB, got %v", err)
	}
}

func TestPixKeyService_ListByAccount_Error(t *testing.T) {
	keyRepo := newMockPixKeyRepo()
	keyRepo.listErr = errDB
	svc := usecase.NewPixKeyService(keyRepo, newMockAccountRepo())

	_, err := svc.ListByAccount(context.Background(), "acc-1")
	if err != errDB {
		t.Errorf("expected errDB, got %v", err)
	}
}

func TestPixKeyService_ListByAccount(t *testing.T) {
	t.Run("returns keys for account", func(t *testing.T) {
		keyRepo := newMockPixKeyRepo()
		k1, _ := pixkey.NewPixKey("acc-1", "EMAIL", "a@b.com")
		k2, _ := pixkey.NewPixKey("acc-1", "EMAIL", "c@d.com")
		k3, _ := pixkey.NewPixKey("acc-2", "EMAIL", "e@f.com")
		keyRepo.keys["a@b.com"] = k1
		keyRepo.keys["c@d.com"] = k2
		keyRepo.keys["e@f.com"] = k3
		svc := usecase.NewPixKeyService(keyRepo, newMockAccountRepo())

		keys, err := svc.ListByAccount(context.Background(), "acc-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("got %d keys, want 2", len(keys))
		}
	})

	t.Run("empty list when no keys", func(t *testing.T) {
		svc := usecase.NewPixKeyService(newMockPixKeyRepo(), newMockAccountRepo())
		keys, err := svc.ListByAccount(context.Background(), "acc-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("expected 0 keys, got %d", len(keys))
		}
	})
}
