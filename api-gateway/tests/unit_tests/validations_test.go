package unit_tests

import (
	"errors"
	"testing"

	"go-task-wallet-service/api-gateway/internal/domain"
	"go-task-wallet-service/api-gateway/internal/utils"
)

func TestValidateAccount_Valid(t *testing.T) {
	account := &domain.Account{Currency: "USD", OwnerUser: "user-1"}
	if err := utils.ValidateAccount(account); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAccount_MissingCurrency(t *testing.T) {
	account := &domain.Account{Currency: "", OwnerUser: "user-1"}
	err := utils.ValidateAccount(account)
	if !errors.Is(err, utils.ErrCurrencyRequired) {
		t.Fatalf("expected ErrCurrencyRequired, got: %v", err)
	}
}

func TestValidateAccount_MissingOwnerUser(t *testing.T) {
	account := &domain.Account{Currency: "USD", OwnerUser: ""}
	err := utils.ValidateAccount(account)
	if !errors.Is(err, utils.ErrOwnerUserRequired) {
		t.Fatalf("expected ErrOwnerUserRequired, got: %v", err)
	}
}

func TestValidateAccount_MissingBoth_ReturnsCurrencyErrorFirst(t *testing.T) {
	account := &domain.Account{Currency: "", OwnerUser: ""}
	err := utils.ValidateAccount(account)
	if !errors.Is(err, utils.ErrCurrencyRequired) {
		t.Fatalf("expected ErrCurrencyRequired to take precedence, got: %v", err)
	}
}
