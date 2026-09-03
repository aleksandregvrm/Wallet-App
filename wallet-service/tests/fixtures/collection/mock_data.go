package collection

import (
	"time"

	types "go-task-wallet-service/shared/pkg/models"
	"go-task-wallet-service/wallet-service/internal/domain"
)

func MockAccount() *domain.Account {
	return &domain.Account{
		ID:        "account-1",
		UserID:    "user-1",
		Balance:   10000,
		Currency:  "USD",
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}
}

func MockAccountModel() *types.AccountModel {
	return &types.AccountModel{
		ID:        "account-1",
		UserID:    "user-1",
		Balance:   10000,
		Currency:  "USD",
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}
}

func MockTransactionModel() *types.TransactionModel {
	return &types.TransactionModel{
		ID:          "transaction-1",
		FromAccount: "account-1",
		ToAccount:   "account-1",
		Amount:      500,
		Currency:    "USD",
		Status:      types.TransactionStatusCompleted,
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}
}

func MockOutboxModel() *types.OutboxModel {
	return &types.OutboxModel{
		ID:            "outbox-1",
		AggregateType: "transaction",
		AggregateId:   "transaction-1",
		EventType:     "transaction.deposit.requested",
		Payload:       map[string]interface{}{"ID": "transaction-1"},
		Topic:         "wallet.events.v1",
		PartitionKey:  "account-1",
		IdempotencyKey: "user-1_account-1",
		Status:        types.OutboxStatusPending,
		CreatedAt:     time.Unix(0, 0).UTC(),
		UpdatedAt:     time.Unix(0, 0).UTC(),
	}
}
