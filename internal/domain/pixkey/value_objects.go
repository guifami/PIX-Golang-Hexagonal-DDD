package pixkey

import (
	"regexp"
	"strings"
)

type KeyType string

const (
	CPF   KeyType = "CPF"
	CNPJ  KeyType = "CNPJ"
	Phone KeyType = "PHONE"
	Email KeyType = "EMAIL"
	EVP   KeyType = "EVP"
)

type KeyStatus string

const (
	Active   KeyStatus = "ACTIVE"
	Inactive KeyStatus = "INACTIVE"
)

var (
	reCPF   = regexp.MustCompile(`^\d{11}$`)
	reCNPJ  = regexp.MustCompile(`^\d{14}$`)
	rePhone = regexp.MustCompile(`^\+\d{10,13}$`)
	reEmail = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	reEVP   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func ParseKeyType(s string) (KeyType, error) {
	switch KeyType(strings.ToUpper(s)) {
	case CPF, CNPJ, Phone, Email, EVP:
		return KeyType(strings.ToUpper(s)), nil
	default:
		return "", ErrInvalidKeyType
	}
}

func ValidateKeyValue(kt KeyType, value string) error {
	switch kt {
	case CPF:
		if !reCPF.MatchString(value) {
			return ErrInvalidKeyValue
		}
	case CNPJ:
		if !reCNPJ.MatchString(value) {
			return ErrInvalidKeyValue
		}
	case Phone:
		if !rePhone.MatchString(value) {
			return ErrInvalidKeyValue
		}
	case Email:
		if !reEmail.MatchString(strings.ToLower(value)) {
			return ErrInvalidKeyValue
		}
	case EVP:
		if !reEVP.MatchString(strings.ToLower(value)) {
			return ErrInvalidKeyValue
		}
	}
	return nil
}
