package utils

// File dedicated to Account related validations

import (
	"errors"
	"go-task-wallet-service/api-gateway/internal/domain"
)

// Errors based on failed Account validation
var (
	ErrCurrencyRequired  = errors.New("currency is required")
	ErrOwnerUserRequired = errors.New("owner user is required")
)

// ValidateAccount runs basic sanity checks on an Account before creation.
func ValidateAccount(account *domain.Account) error {
	if account.Currency == "" {
		return ErrCurrencyRequired
	}
	if account.OwnerUser == "" {
		return ErrOwnerUserRequired
	}

	return nil
}
