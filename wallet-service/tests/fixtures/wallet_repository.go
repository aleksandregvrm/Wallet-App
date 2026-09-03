package fixtures

import (
	"context"
	"fmt"

	types "go-task-wallet-service/shared/pkg/models"
	"go-task-wallet-service/wallet-service/internal/domain"
)

var _ domain.WalletRepository = (*FakeWalletRepository)(nil)

type FakeWalletRepository struct {
	InsertAccountFunc         func(ctx context.Context, userId, currency string) (*types.AccountModel, error)
	FindAccountByIdFunc       func(ctx context.Context, accountId string) (*types.AccountModel, error)
	FindAccountsByUserIdFunc  func(ctx context.Context, userId string) ([]types.AccountModel, error)
	UpdateAccountFunc         func(ctx context.Context, fromAccountId, toAccountId string) (*types.AccountModel, error)
	OutboxInsertFunc          func(ctx context.Context, cfg *domain.OutboxInsertConfig) error
	OutboxUpdateFunc          func(ctx context.Context, aggregateId, status string) error
	OutboxMarkPublishFailureFunc func(ctx context.Context, aggregateId, errMsg string) (string, error)
	OutboxDeleteFunc          func(ctx context.Context, aggregateId string) error
	OutboxGetFunc             func(ctx context.Context, aggregateId string) (*types.OutboxModel, error)
	OutboxPendingGetBatchFunc func(ctx context.Context, limit int) ([]*types.OutboxModel, error)
	TransferFundsFunc         func(ctx context.Context, transactionId, fromAccountId, toAccountId, currency string, amount int64) (*types.TransactionModel, error)
	CreatePendingTransactionFunc func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error
	DepositFundsFunc          func(ctx context.Context, transactionId, accountId, currency string, amount int64) (*types.TransactionModel, error)
	WithdrawFundsFunc         func(ctx context.Context, transactionId, accountId string, amount int64) (*types.TransactionModel, error)
	ListTransactionsFunc      func(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error)
	UpdateTransactionStatusFunc func(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error
	IdempotencyReleaseFunc    func(ctx context.Context, transactionId, userId, accountId string) error
	GetOrStoreAccountBalanceFunc func(ctx context.Context, accountId string, amount int64) (int64, error)
}

func (f *FakeWalletRepository) InsertAccount(ctx context.Context, userId, currency string) (*types.AccountModel, error) {
	if f.InsertAccountFunc == nil {
		return nil, fmt.Errorf("fixtures: InsertAccountFunc not set")
	}
	return f.InsertAccountFunc(ctx, userId, currency)
}

func (f *FakeWalletRepository) FindAccountById(ctx context.Context, accountId string) (*types.AccountModel, error) {
	if f.FindAccountByIdFunc == nil {
		return nil, fmt.Errorf("fixtures: FindAccountByIdFunc not set")
	}
	return f.FindAccountByIdFunc(ctx, accountId)
}

func (f *FakeWalletRepository) FindAccountsByUserId(ctx context.Context, userId string) ([]types.AccountModel, error) {
	if f.FindAccountsByUserIdFunc == nil {
		return nil, fmt.Errorf("fixtures: FindAccountsByUserIdFunc not set")
	}
	return f.FindAccountsByUserIdFunc(ctx, userId)
}

func (f *FakeWalletRepository) UpdateAccount(ctx context.Context, fromAccountId, toAccountId string) (*types.AccountModel, error) {
	if f.UpdateAccountFunc == nil {
		return nil, fmt.Errorf("fixtures: UpdateAccountFunc not set")
	}
	return f.UpdateAccountFunc(ctx, fromAccountId, toAccountId)
}

func (f *FakeWalletRepository) OutboxInsert(ctx context.Context, cfg *domain.OutboxInsertConfig) error {
	if f.OutboxInsertFunc == nil {
		return fmt.Errorf("fixtures: OutboxInsertFunc not set")
	}
	return f.OutboxInsertFunc(ctx, cfg)
}

func (f *FakeWalletRepository) OutboxUpdate(ctx context.Context, aggregateId, status string) error {
	if f.OutboxUpdateFunc == nil {
		return fmt.Errorf("fixtures: OutboxUpdateFunc not set")
	}
	return f.OutboxUpdateFunc(ctx, aggregateId, status)
}

func (f *FakeWalletRepository) OutboxMarkPublishFailure(ctx context.Context, aggregateId, errMsg string) (string, error) {
	if f.OutboxMarkPublishFailureFunc == nil {
		return "", fmt.Errorf("fixtures: OutboxMarkPublishFailureFunc not set")
	}
	return f.OutboxMarkPublishFailureFunc(ctx, aggregateId, errMsg)
}

func (f *FakeWalletRepository) OutboxDelete(ctx context.Context, aggregateId string) error {
	if f.OutboxDeleteFunc == nil {
		return fmt.Errorf("fixtures: OutboxDeleteFunc not set")
	}
	return f.OutboxDeleteFunc(ctx, aggregateId)
}

func (f *FakeWalletRepository) OutboxGet(ctx context.Context, aggregateId string) (*types.OutboxModel, error) {
	if f.OutboxGetFunc == nil {
		return nil, fmt.Errorf("fixtures: OutboxGetFunc not set")
	}
	return f.OutboxGetFunc(ctx, aggregateId)
}

func (f *FakeWalletRepository) OutboxPendingGetBatch(ctx context.Context, limit int) ([]*types.OutboxModel, error) {
	if f.OutboxPendingGetBatchFunc == nil {
		return nil, fmt.Errorf("fixtures: OutboxPendingGetBatchFunc not set")
	}
	return f.OutboxPendingGetBatchFunc(ctx, limit)
}

func (f *FakeWalletRepository) TransferFunds(ctx context.Context, transactionId, fromAccountId, toAccountId, currency string, amount int64) (*types.TransactionModel, error) {
	if f.TransferFundsFunc == nil {
		return nil, fmt.Errorf("fixtures: TransferFundsFunc not set")
	}
	return f.TransferFundsFunc(ctx, transactionId, fromAccountId, toAccountId, currency, amount)
}

func (f *FakeWalletRepository) CreatePendingTransaction(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
	if f.CreatePendingTransactionFunc == nil {
		return fmt.Errorf("fixtures: CreatePendingTransactionFunc not set")
	}
	return f.CreatePendingTransactionFunc(ctx, transactionId, transactionType, fromAccount, toAccount, currency, amount)
}

func (f *FakeWalletRepository) DepositFunds(ctx context.Context, transactionId, accountId, currency string, amount int64) (*types.TransactionModel, error) {
	if f.DepositFundsFunc == nil {
		return nil, fmt.Errorf("fixtures: DepositFundsFunc not set")
	}
	return f.DepositFundsFunc(ctx, transactionId, accountId, currency, amount)
}

func (f *FakeWalletRepository) WithdrawFunds(ctx context.Context, transactionId, accountId string, amount int64) (*types.TransactionModel, error) {
	if f.WithdrawFundsFunc == nil {
		return nil, fmt.Errorf("fixtures: WithdrawFundsFunc not set")
	}
	return f.WithdrawFundsFunc(ctx, transactionId, accountId, amount)
}

func (f *FakeWalletRepository) ListTransactions(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error) {
	if f.ListTransactionsFunc == nil {
		return nil, fmt.Errorf("fixtures: ListTransactionsFunc not set")
	}
	return f.ListTransactionsFunc(ctx, accountId, offset, limit)
}

func (f *FakeWalletRepository) UpdateTransactionStatus(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error {
	if f.UpdateTransactionStatusFunc == nil {
		return fmt.Errorf("fixtures: UpdateTransactionStatusFunc not set")
	}
	return f.UpdateTransactionStatusFunc(ctx, transactionId, userId, accountId, status)
}

func (f *FakeWalletRepository) IdempotencyRelease(ctx context.Context, transactionId, userId, accountId string) error {
	if f.IdempotencyReleaseFunc == nil {
		return fmt.Errorf("fixtures: IdempotencyReleaseFunc not set")
	}
	return f.IdempotencyReleaseFunc(ctx, transactionId, userId, accountId)
}

func (f *FakeWalletRepository) GetOrStoreAccountBalance(ctx context.Context, accountId string, amount int64) (int64, error) {
	if f.GetOrStoreAccountBalanceFunc == nil {
		return 0, fmt.Errorf("fixtures: GetOrStoreAccountBalanceFunc not set")
	}
	return f.GetOrStoreAccountBalanceFunc(ctx, accountId, amount)
}
