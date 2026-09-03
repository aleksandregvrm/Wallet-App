package collection

import (
	"go-task-wallet-service/api-gateway/internal/domain"
	"time"
)

// Mock domain user
func MockUser() *domain.User {
	return &domain.User{
		ID:        "user-1",
		Name:      "Jane Doe",
		Username:  "janedoe",
		Email:     "jane@example.com",
		Password:  "hunter22",
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}
}

// Mock domain account
func MockAccount() *domain.Account {
	return &domain.Account{
		ID:        "account-1",
		Balance:   1000,
		Currency:  "USD",
		OwnerUser: "user-1",
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}
}

// Mock domain transaction
func MockTransaction() *domain.Transaction {
	return &domain.Transaction{
		ID:          "txn-1",
		FromAccount: "account-1",
		ToAccount:   "account-2",
		Amount:      500,
		Currency:    "USD",
		Status:      "completed",
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}
}
