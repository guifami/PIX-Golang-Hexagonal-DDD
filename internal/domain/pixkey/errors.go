package pixkey

import "errors"

var (
	ErrKeyNotFound       = errors.New("pix key not found")
	ErrKeyAlreadyExists  = errors.New("pix key already registered")
	ErrKeyLimitReached   = errors.New("maximum of 5 pix keys per account reached")
	ErrInvalidKeyType    = errors.New("invalid pix key type (allowed: CPF, CNPJ, PHONE, EMAIL, EVP)")
	ErrInvalidKeyValue   = errors.New("invalid pix key value for the given type")
	ErrUnauthorized      = errors.New("this pix key does not belong to the given account")
)
