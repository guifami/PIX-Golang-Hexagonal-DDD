package transaction

import "errors"

var (
	ErrTransactionNotFound      = errors.New("transaction not found")
	ErrInvalidAmount            = errors.New("amount must be greater than zero")
	ErrSameAccount              = errors.New("payer and receiver cannot be the same account")
	ErrTransactionAlreadyProcessed = errors.New("transaction has already been processed")
	ErrInvalidTransactionStatus = errors.New("invalid transaction status transition")
)
