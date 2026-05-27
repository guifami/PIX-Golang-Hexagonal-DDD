package transaction_test

import (
	"testing"

	"go-api/internal/domain/account"
	"go-api/internal/domain/transaction"
)

func newMoney(t *testing.T, cents int64) account.Money {
	t.Helper()
	m, err := account.NewMoney(cents)
	if err != nil {
		t.Fatalf("NewMoney(%d): %v", cents, err)
	}
	return m
}

// ── NewTransaction ────────────────────────────────────────────────────────

func TestNewTransaction(t *testing.T) {
	t.Run("valid transaction", func(t *testing.T) {
		tx, err := transaction.NewTransaction("payer-id", "key@email.com", "desc", newMoney(t, 1000))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.ID == "" {
			t.Error("ID should not be empty")
		}
		if tx.Status != transaction.Pending {
			t.Errorf("initial status = %q, want PENDING", tx.Status)
		}
		if tx.Description != "desc" {
			t.Errorf("Description = %q", tx.Description)
		}
		if tx.Amount.Cents() != 1000 {
			t.Errorf("Amount = %d, want 1000", tx.Amount.Cents())
		}
		if tx.CompletedAt != nil {
			t.Error("CompletedAt should be nil initially")
		}
		if tx.FailureReason != nil {
			t.Error("FailureReason should be nil initially")
		}
	})
	t.Run("description is trimmed", func(t *testing.T) {
		tx, _ := transaction.NewTransaction("p", "k", "  hello  ", newMoney(t, 100))
		if tx.Description != "hello" {
			t.Errorf("Description = %q, want 'hello'", tx.Description)
		}
	})
	t.Run("zero amount rejected", func(t *testing.T) {
		_, err := transaction.NewTransaction("p", "k", "", newMoney(t, 0))
		if err != transaction.ErrInvalidAmount {
			t.Errorf("expected ErrInvalidAmount, got %v", err)
		}
	})
}

// ── State machine ─────────────────────────────────────────────────────────

func TestStartProcessing(t *testing.T) {
	t.Run("from PENDING", func(t *testing.T) {
		tx, _ := transaction.NewTransaction("p", "k", "", newMoney(t, 100))
		if err := tx.StartProcessing(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.Status != transaction.Processing {
			t.Errorf("status = %q, want PROCESSING", tx.Status)
		}
	})
	t.Run("from non-PENDING is error", func(t *testing.T) {
		tx, _ := transaction.NewTransaction("p", "k", "", newMoney(t, 100))
		_ = tx.StartProcessing()
		err := tx.StartProcessing()
		if err != transaction.ErrTransactionAlreadyProcessed {
			t.Errorf("expected ErrTransactionAlreadyProcessed, got %v", err)
		}
	})
}

func TestComplete(t *testing.T) {
	t.Run("from PROCESSING", func(t *testing.T) {
		tx, _ := transaction.NewTransaction("p", "k", "", newMoney(t, 100))
		_ = tx.StartProcessing()
		if err := tx.Complete("receiver-id"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.Status != transaction.Completed {
			t.Errorf("status = %q, want COMPLETED", tx.Status)
		}
		if tx.ReceiverAccountID != "receiver-id" {
			t.Errorf("ReceiverAccountID = %q", tx.ReceiverAccountID)
		}
		if tx.CompletedAt == nil {
			t.Error("CompletedAt should be set")
		}
	})
	t.Run("from non-PROCESSING is error", func(t *testing.T) {
		tx, _ := transaction.NewTransaction("p", "k", "", newMoney(t, 100))
		err := tx.Complete("r")
		if err != transaction.ErrInvalidTransactionStatus {
			t.Errorf("expected ErrInvalidTransactionStatus, got %v", err)
		}
	})
}

func TestFail(t *testing.T) {
	t.Run("from PENDING", func(t *testing.T) {
		tx, _ := transaction.NewTransaction("p", "k", "", newMoney(t, 100))
		if err := tx.Fail("some reason"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.Status != transaction.Failed {
			t.Errorf("status = %q, want FAILED", tx.Status)
		}
		if tx.FailureReason == nil || *tx.FailureReason != "some reason" {
			t.Errorf("FailureReason = %v", tx.FailureReason)
		}
		if tx.CompletedAt == nil {
			t.Error("CompletedAt should be set on fail")
		}
	})
	t.Run("from PROCESSING", func(t *testing.T) {
		tx, _ := transaction.NewTransaction("p", "k", "", newMoney(t, 100))
		_ = tx.StartProcessing()
		if err := tx.Fail("reason"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.Status != transaction.Failed {
			t.Errorf("status = %q, want FAILED", tx.Status)
		}
	})
	t.Run("from COMPLETED is error", func(t *testing.T) {
		tx, _ := transaction.NewTransaction("p", "k", "", newMoney(t, 100))
		_ = tx.StartProcessing()
		_ = tx.Complete("r")
		err := tx.Fail("too late")
		if err != transaction.ErrInvalidTransactionStatus {
			t.Errorf("expected ErrInvalidTransactionStatus, got %v", err)
		}
	})
}

func TestIsFinal(t *testing.T) {
	tx, _ := transaction.NewTransaction("p", "k", "", newMoney(t, 100))
	if tx.IsFinal() {
		t.Error("PENDING should not be final")
	}
	_ = tx.StartProcessing()
	if tx.IsFinal() {
		t.Error("PROCESSING should not be final")
	}
	_ = tx.Complete("r")
	if !tx.IsFinal() {
		t.Error("COMPLETED should be final")
	}
}
