package pixkey_test

import (
	"testing"

	"go-api/internal/domain/pixkey"
)

// ── ParseKeyType ──────────────────────────────────────────────────────────

func TestParseKeyType(t *testing.T) {
	tests := []struct {
		input   string
		want    pixkey.KeyType
		wantErr bool
	}{
		{"CPF", pixkey.CPF, false},
		{"cpf", pixkey.CPF, false},
		{"CNPJ", pixkey.CNPJ, false},
		{"PHONE", pixkey.Phone, false},
		{"EMAIL", pixkey.Email, false},
		{"EVP", pixkey.EVP, false},
		{"INVALID", "", true},
		{"", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := pixkey.ParseKeyType(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ParseKeyType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ── ValidateKeyValue ──────────────────────────────────────────────────────

func TestValidateKeyValue(t *testing.T) {
	tests := []struct {
		name    string
		kt      pixkey.KeyType
		value   string
		wantErr bool
	}{
		// CPF
		{"CPF valid", pixkey.CPF, "11144477735", false},
		{"CPF too short", pixkey.CPF, "1234567890", true},
		{"CPF with letters", pixkey.CPF, "1114447773a", true},
		// CNPJ
		{"CNPJ valid", pixkey.CNPJ, "12345678000199", false},
		{"CNPJ too short", pixkey.CNPJ, "1234567800019", true},
		// Phone
		{"Phone valid", pixkey.Phone, "+5511999998888", false},
		{"Phone no plus", pixkey.Phone, "5511999998888", true},
		{"Phone too short", pixkey.Phone, "+551199999", true},
		// Email
		{"Email valid", pixkey.Email, "user@example.com", false},
		{"Email no at", pixkey.Email, "userexample.com", true},
		{"Email no domain", pixkey.Email, "user@", true},
		// EVP
		{"EVP valid", pixkey.EVP, "550e8400-e29b-41d4-a716-446655440000", false},
		{"EVP invalid format", pixkey.EVP, "not-a-uuid", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := pixkey.ValidateKeyValue(tc.kt, tc.value)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ── NewPixKey ─────────────────────────────────────────────────────────────

func TestNewPixKey(t *testing.T) {
	t.Run("valid email key", func(t *testing.T) {
		k, err := pixkey.NewPixKey("account-id", "EMAIL", "test@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if k.ID == "" {
			t.Error("ID should not be empty")
		}
		if k.AccountID != "account-id" {
			t.Errorf("AccountID = %q", k.AccountID)
		}
		if k.Type != pixkey.Email {
			t.Errorf("Type = %q, want EMAIL", k.Type)
		}
		if !k.IsActive() {
			t.Error("new key should be active")
		}
	})
	t.Run("invalid key type", func(t *testing.T) {
		_, err := pixkey.NewPixKey("acc", "UNKNOWN", "value")
		if err != pixkey.ErrInvalidKeyType {
			t.Errorf("expected ErrInvalidKeyType, got %v", err)
		}
	})
	t.Run("invalid key value", func(t *testing.T) {
		_, err := pixkey.NewPixKey("acc", "CPF", "not-a-cpf")
		if err != pixkey.ErrInvalidKeyValue {
			t.Errorf("expected ErrInvalidKeyValue, got %v", err)
		}
	})
}

// ── PixKey methods ────────────────────────────────────────────────────────

func TestPixKeyBelongsTo(t *testing.T) {
	k, _ := pixkey.NewPixKey("owner-id", "EMAIL", "a@b.com")
	if !k.BelongsTo("owner-id") {
		t.Error("should belong to owner-id")
	}
	if k.BelongsTo("other-id") {
		t.Error("should not belong to other-id")
	}
}

func TestPixKeyDeactivate(t *testing.T) {
	k, _ := pixkey.NewPixKey("acc", "EMAIL", "a@b.com")
	if !k.IsActive() {
		t.Error("should be active initially")
	}
	k.Deactivate()
	if k.IsActive() {
		t.Error("should be inactive after Deactivate")
	}
}
