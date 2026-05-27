package account

import "errors"

var (
	ErrAccountNotFound  = errors.New("account not found")
	ErrAccountInactive  = errors.New("account is inactive")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidCPF       = errors.New("invalid CPF")
	ErrInvalidName      = errors.New("invalid account owner name")
	ErrInvalidAmount    = errors.New("amount must be non-negative")
	ErrCPFAlreadyInUse  = errors.New("CPF already in use")
)
