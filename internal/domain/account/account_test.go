package account_test

import (
	"testing"

	"go-api/internal/domain/account"
)

// ── Money ──────────────────────────────────────────────────────────────────

func TestNewMoney(t *testing.T) {
	tests := []struct {
		name    string
		cents   int64
		wantErr bool
	}{
		{"zero is valid", 0, false},
		{"positive is valid", 100, false},
		{"negative returns error", -1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := account.NewMoney(tc.cents)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.Cents() != tc.cents {
				t.Errorf("Cents() = %d, want %d", m.Cents(), tc.cents)
			}
		})
	}
}

func TestMoneyIsPositive(t *testing.T) {
	if account.Money(0).IsPositive() {
		t.Error("zero should not be positive")
	}
	if !account.Money(1).IsPositive() {
		t.Error("1 should be positive")
	}
}

func TestMoneyAdd(t *testing.T) {
	a := account.Money(100)
	b := account.Money(50)
	if got := a.Add(b).Cents(); got != 150 {
		t.Errorf("Add = %d, want 150", got)
	}
}

func TestMoneySub(t *testing.T) {
	t.Run("sufficient funds", func(t *testing.T) {
		m := account.Money(100)
		result, err := m.Sub(account.Money(40))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Cents() != 60 {
			t.Errorf("Sub = %d, want 60", result.Cents())
		}
	})
	t.Run("exact amount", func(t *testing.T) {
		m := account.Money(100)
		result, err := m.Sub(account.Money(100))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Cents() != 0 {
			t.Errorf("Sub = %d, want 0", result.Cents())
		}
	})
	t.Run("insufficient funds", func(t *testing.T) {
		m := account.Money(10)
		_, err := m.Sub(account.Money(20))
		if err != account.ErrInsufficientFunds {
			t.Errorf("expected ErrInsufficientFunds, got %v", err)
		}
	})
}

func TestCPFString(t *testing.T) {
	cpf, _ := account.NewCPF("11144477735")
	if cpf.String() != "11144477735" {
		t.Errorf("String() = %q, want %q", cpf.String(), "11144477735")
	}
}

func TestMoneyString(t *testing.T) {
	if got := account.Money(15050).String(); got != "R$ 150.50" {
		t.Errorf("String() = %q, want %q", got, "R$ 150.50")
	}
	if got := account.Money(100).String(); got != "R$ 1.00" {
		t.Errorf("String() = %q, want %q", got, "R$ 1.00")
	}
}

// ── CPF ───────────────────────────────────────────────────────────────────

func TestNewCPF(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid CPF digits only", "11144477735", false},
		{"valid CPF with punctuation", "111.444.777-35", false},
		{"all same digits", "11111111111", true},
		{"wrong length", "1234567890", true},
		{"wrong check digit", "11144477734", true},
		{"wrong first check digit only", "11144477715", true},
		{"valid CPF where r1 computation reaches 10 (edge case)", "00000020206", false},
		{"valid CPF where r2 computation reaches 10 (edge case)", "00000800040", false},
		{"empty", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := account.NewCPF(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ── Account ───────────────────────────────────────────────────────────────

func newTestAccount(t *testing.T) *account.Account {
	t.Helper()
	a, err := account.NewAccount("João Silva", "11144477735")
	if err != nil {
		t.Fatalf("NewAccount: %v", err)
	}
	return a
}

func TestNewAccount(t *testing.T) {
	t.Run("valid account", func(t *testing.T) {
		a := newTestAccount(t)
		if a.ID == "" {
			t.Error("ID should not be empty")
		}
		if a.OwnerName != "João Silva" {
			t.Errorf("OwnerName = %q", a.OwnerName)
		}
		if a.Balance.Cents() != 0 {
			t.Error("initial balance should be 0")
		}
		if !a.IsActive() {
			t.Error("new account should be active")
		}
	})
	t.Run("empty name", func(t *testing.T) {
		_, err := account.NewAccount("", "11144477735")
		if err != account.ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})
	t.Run("whitespace name", func(t *testing.T) {
		_, err := account.NewAccount("   ", "11144477735")
		if err != account.ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})
	t.Run("invalid CPF", func(t *testing.T) {
		_, err := account.NewAccount("Test", "00000000000")
		if err == nil {
			t.Error("expected error for invalid CPF")
		}
	})
}

func TestAccountDebit(t *testing.T) {
	t.Run("successful debit", func(t *testing.T) {
		a := newTestAccount(t)
		a.Balance = account.Money(1000)
		if err := a.Debit(account.Money(300)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Balance.Cents() != 700 {
			t.Errorf("balance = %d, want 700", a.Balance.Cents())
		}
	})
	t.Run("insufficient funds", func(t *testing.T) {
		a := newTestAccount(t)
		a.Balance = account.Money(100)
		err := a.Debit(account.Money(200))
		if err != account.ErrInsufficientFunds {
			t.Errorf("expected ErrInsufficientFunds, got %v", err)
		}
	})
	t.Run("debit from inactive account", func(t *testing.T) {
		a := newTestAccount(t)
		a.Balance = account.Money(1000)
		a.Deactivate()
		err := a.Debit(account.Money(100))
		if err != account.ErrAccountInactive {
			t.Errorf("expected ErrAccountInactive, got %v", err)
		}
	})
}

func TestAccountCredit(t *testing.T) {
	t.Run("successful credit", func(t *testing.T) {
		a := newTestAccount(t)
		if err := a.Credit(account.Money(500)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Balance.Cents() != 500 {
			t.Errorf("balance = %d, want 500", a.Balance.Cents())
		}
	})
	t.Run("credit to inactive account", func(t *testing.T) {
		a := newTestAccount(t)
		a.Deactivate()
		err := a.Credit(account.Money(100))
		if err != account.ErrAccountInactive {
			t.Errorf("expected ErrAccountInactive, got %v", err)
		}
	})
}

func TestAccountActivateDeactivate(t *testing.T) {
	a := newTestAccount(t)
	a.Deactivate()
	if a.IsActive() {
		t.Error("should be inactive after Deactivate")
	}
	a.Activate()
	if !a.IsActive() {
		t.Error("should be active after Activate")
	}
}
