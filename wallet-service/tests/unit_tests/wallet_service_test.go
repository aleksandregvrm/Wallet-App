package unit_tests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go-task-wallet-service/shared/events"
	types "go-task-wallet-service/shared/pkg/models"
	"go-task-wallet-service/wallet-service/internal/domain"
	"go-task-wallet-service/wallet-service/internal/services"
	"go-task-wallet-service/wallet-service/tests/fixtures"
)

func newWalletService(repo *fixtures.FakeWalletRepository) *services.WalletService {
	return services.NewWalletService(repo, &fixtures.FakeEventPublisher{})
}

// --- OpenAccount ---

func TestWalletService_OpenAccount_Success(t *testing.T) {
	var insertCalled bool
	repo := &fixtures.FakeWalletRepository{
		FindAccountsByUserIdFunc: func(ctx context.Context, userId string) ([]types.AccountModel, error) {
			return nil, nil
		},
		InsertAccountFunc: func(ctx context.Context, userId, currency string) (*types.AccountModel, error) {
			insertCalled = true
			if userId != "user-1" || currency != "USD" {
				t.Fatalf("unexpected insert args: userId=%q currency=%q", userId, currency)
			}
			return &types.AccountModel{ID: "account-1", UserID: userId, Currency: currency}, nil
		},
	}
	svc := newWalletService(repo)

	account, err := svc.OpenAccount(context.Background(), "user-1", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !insertCalled {
		t.Fatalf("expected InsertAccount to be called")
	}
	if account.ID != "account-1" {
		t.Fatalf("unexpected account: %+v", account)
	}
}

func TestWalletService_OpenAccount_MissingUserId(t *testing.T) {
	svc := newWalletService(&fixtures.FakeWalletRepository{})

	_, err := svc.OpenAccount(context.Background(), "", "USD")
	if err == nil {
		t.Fatalf("expected an error for a missing userId")
	}
}

func TestWalletService_OpenAccount_MissingCurrency(t *testing.T) {
	svc := newWalletService(&fixtures.FakeWalletRepository{})

	_, err := svc.OpenAccount(context.Background(), "user-1", "")
	if err == nil {
		t.Fatalf("expected an error for a missing currency")
	}
}

func TestWalletService_OpenAccount_FindAccountsError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	repo := &fixtures.FakeWalletRepository{
		FindAccountsByUserIdFunc: func(ctx context.Context, userId string) ([]types.AccountModel, error) {
			return nil, wantErr
		},
	}
	svc := newWalletService(repo)

	_, err := svc.OpenAccount(context.Background(), "user-1", "USD")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}

func TestWalletService_OpenAccount_MaxAccountsPerCurrencyReached(t *testing.T) {
	insertCalled := false
	repo := &fixtures.FakeWalletRepository{
		FindAccountsByUserIdFunc: func(ctx context.Context, userId string) ([]types.AccountModel, error) {
			return []types.AccountModel{
				{ID: "account-1", Currency: "USD"},
				{ID: "account-2", Currency: "USD"},
				{ID: "account-3", Currency: "USD"},
			}, nil
		},
		InsertAccountFunc: func(ctx context.Context, userId, currency string) (*types.AccountModel, error) {
			insertCalled = true
			return nil, nil
		},
	}
	svc := newWalletService(repo)

	_, err := svc.OpenAccount(context.Background(), "user-1", "USD")
	if err == nil {
		t.Fatalf("expected an error when the user already has 3 accounts in this currency")
	}
	if insertCalled {
		t.Fatalf("expected InsertAccount never to be called once the per-currency limit is reached")
	}
}

func TestWalletService_OpenAccount_DifferentCurrencyStillAllowed(t *testing.T) {
	var insertCalled bool
	repo := &fixtures.FakeWalletRepository{
		FindAccountsByUserIdFunc: func(ctx context.Context, userId string) ([]types.AccountModel, error) {
			return []types.AccountModel{
				{ID: "account-1", Currency: "USD"},
				{ID: "account-2", Currency: "USD"},
				{ID: "account-3", Currency: "USD"},
			}, nil
		},
		InsertAccountFunc: func(ctx context.Context, userId, currency string) (*types.AccountModel, error) {
			insertCalled = true
			return &types.AccountModel{ID: "account-4", UserID: userId, Currency: currency}, nil
		},
	}
	svc := newWalletService(repo)

	_, err := svc.OpenAccount(context.Background(), "user-1", "EUR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !insertCalled {
		t.Fatalf("expected InsertAccount to be called for a currency under the per-currency limit")
	}
}

func TestWalletService_OpenAccount_InsertAccountError(t *testing.T) {
	wantErr := errors.New("unique violation")
	repo := &fixtures.FakeWalletRepository{
		FindAccountsByUserIdFunc: func(ctx context.Context, userId string) ([]types.AccountModel, error) {
			return nil, nil
		},
		InsertAccountFunc: func(ctx context.Context, userId, currency string) (*types.AccountModel, error) {
			return nil, wantErr
		},
	}
	svc := newWalletService(repo)

	_, err := svc.OpenAccount(context.Background(), "user-1", "USD")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}

// --- GetBalance ---

func TestWalletService_GetBalance_Success(t *testing.T) {
	repo := &fixtures.FakeWalletRepository{
		FindAccountByIdFunc: func(ctx context.Context, accountId string) (*types.AccountModel, error) {
			return &types.AccountModel{ID: accountId, UserID: "user-1", Balance: 5000, Currency: "USD"}, nil
		},
		GetOrStoreAccountBalanceFunc: func(ctx context.Context, accountId string, amount int64) (int64, error) {
			return amount, nil
		},
	}
	svc := newWalletService(repo)

	account, err := svc.GetBalance(context.Background(), "user-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.Balance != 5000 || account.Currency != "USD" {
		t.Fatalf("unexpected account: %+v", account)
	}
}

func TestWalletService_GetBalance_MissingAccountId(t *testing.T) {
	svc := newWalletService(&fixtures.FakeWalletRepository{})

	_, err := svc.GetBalance(context.Background(), "user-1", "")
	if err == nil {
		t.Fatalf("expected an error for a missing accountId")
	}
}

func TestWalletService_GetBalance_AccountNotFound(t *testing.T) {
	repo := &fixtures.FakeWalletRepository{
		FindAccountByIdFunc: func(ctx context.Context, accountId string) (*types.AccountModel, error) {
			return nil, errors.New("no rows")
		},
	}
	svc := newWalletService(repo)

	_, err := svc.GetBalance(context.Background(), "user-1", "account-1")
	if err == nil {
		t.Fatalf("expected an error when the account cannot be found")
	}
}

func TestWalletService_GetBalance_AccountBelongsToDifferentUser(t *testing.T) {
	repo := &fixtures.FakeWalletRepository{
		FindAccountByIdFunc: func(ctx context.Context, accountId string) (*types.AccountModel, error) {
			return &types.AccountModel{ID: accountId, UserID: "someone-else", Balance: 5000}, nil
		},
	}
	svc := newWalletService(repo)

	_, err := svc.GetBalance(context.Background(), "user-1", "account-1")
	if err == nil {
		t.Fatalf("expected an error when the account belongs to a different user")
	}
}

// --- HandleDepositDispatched ---

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal test payload: %v", err)
	}
	return b
}

func TestWalletService_HandleDepositDispatched_Success(t *testing.T) {
	var pendingCreated, depositCalled bool
	var finalStatus types.TransactionStatus
	repo := &fixtures.FakeWalletRepository{
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			pendingCreated = true
			if transactionType != "deposit" || fromAccount != "account-1" || toAccount != "account-1" || amount != 500 {
				t.Fatalf("unexpected pending transaction args: type=%s from=%s to=%s amount=%d", transactionType, fromAccount, toAccount, amount)
			}
			return nil
		},
		DepositFundsFunc: func(ctx context.Context, transactionId, accountId, currency string, amount int64) (*types.TransactionModel, error) {
			depositCalled = true
			return &types.TransactionModel{ID: transactionId, FromAccount: accountId, ToAccount: accountId, Amount: amount, Currency: currency, Status: types.TransactionStatusCompleted}, nil
		},
		UpdateTransactionStatusFunc: func(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error {
			finalStatus = status
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.DepositRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 500, Currency: "USD", UserId: "user-1"})
	err := svc.HandleDepositDispatched(context.Background(), "tx-1", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pendingCreated || !depositCalled {
		t.Fatalf("expected both CreatePendingTransaction and DepositFunds to be called")
	}
	if finalStatus != types.TransactionStatusCompleted {
		t.Fatalf("expected the transaction to be marked completed, got: %s", finalStatus)
	}
}

func TestWalletService_HandleDepositDispatched_InvalidAmount(t *testing.T) {
	pendingCreated := false
	repo := &fixtures.FakeWalletRepository{
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			pendingCreated = true
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.DepositRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 0, Currency: "USD", UserId: "user-1"})
	err := svc.HandleDepositDispatched(context.Background(), "tx-1", payload)
	if err == nil {
		t.Fatalf("expected an error for a zero deposit amount")
	}
	if pendingCreated {
		t.Fatalf("expected CreatePendingTransaction never to be called for an invalid amount")
	}
}

func TestWalletService_HandleDepositDispatched_MalformedPayload(t *testing.T) {
	svc := newWalletService(&fixtures.FakeWalletRepository{})

	err := svc.HandleDepositDispatched(context.Background(), "tx-1", []byte("not-json"))
	if !errors.Is(err, events.ErrInvalidMessage) {
		t.Fatalf("expected a wrapped ErrInvalidMessage, got: %v", err)
	}
}

func TestWalletService_HandleDepositDispatched_CreatePendingTransactionError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	depositCalled := false
	repo := &fixtures.FakeWalletRepository{
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			return wantErr
		},
		DepositFundsFunc: func(ctx context.Context, transactionId, accountId, currency string, amount int64) (*types.TransactionModel, error) {
			depositCalled = true
			return nil, nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.DepositRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 500, Currency: "USD", UserId: "user-1"})
	err := svc.HandleDepositDispatched(context.Background(), "tx-1", payload)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if depositCalled {
		t.Fatalf("expected DepositFunds never to be called when creating the pending transaction fails")
	}
}

func TestWalletService_HandleDepositDispatched_DepositFundsError_MarksFailed(t *testing.T) {
	wantErr := errors.New("insufficient reconciliation")
	var finalStatus types.TransactionStatus
	repo := &fixtures.FakeWalletRepository{
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			return nil
		},
		DepositFundsFunc: func(ctx context.Context, transactionId, accountId, currency string, amount int64) (*types.TransactionModel, error) {
			return nil, wantErr
		},
		UpdateTransactionStatusFunc: func(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error {
			finalStatus = status
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.DepositRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 500, Currency: "USD", UserId: "user-1"})
	err := svc.HandleDepositDispatched(context.Background(), "tx-1", payload)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if finalStatus != types.TransactionStatusFailed {
		t.Fatalf("expected the transaction to be marked failed, got: %s", finalStatus)
	}
}

// --- HandleWithdrawalDispatched ---

func TestWalletService_HandleWithdrawalDispatched_Success(t *testing.T) {
	var finalStatus types.TransactionStatus
	repo := &fixtures.FakeWalletRepository{
		FindAccountByIdFunc: func(ctx context.Context, accountId string) (*types.AccountModel, error) {
			return &types.AccountModel{ID: accountId, UserID: "user-1", Balance: 1000, Currency: "USD"}, nil
		},
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			return nil
		},
		WithdrawFundsFunc: func(ctx context.Context, transactionId, accountId string, amount int64) (*types.TransactionModel, error) {
			return &types.TransactionModel{ID: transactionId, FromAccount: accountId, ToAccount: accountId, Amount: amount, Status: types.TransactionStatusCompleted}, nil
		},
		UpdateTransactionStatusFunc: func(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error {
			finalStatus = status
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.WithdrawalRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 200, UserId: "user-1"})
	err := svc.HandleWithdrawalDispatched(context.Background(), "tx-1", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finalStatus != types.TransactionStatusCompleted {
		t.Fatalf("expected the transaction to be marked completed, got: %s", finalStatus)
	}
}

func TestWalletService_HandleWithdrawalDispatched_InvalidAmount(t *testing.T) {
	svc := newWalletService(&fixtures.FakeWalletRepository{})

	payload := mustMarshal(t, domain.WithdrawalRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: -1, UserId: "user-1"})
	err := svc.HandleWithdrawalDispatched(context.Background(), "tx-1", payload)
	if err == nil {
		t.Fatalf("expected an error for a negative withdrawal amount")
	}
}

func TestWalletService_HandleWithdrawalDispatched_AccountNotOwnedByUser(t *testing.T) {
	pendingCreated := false
	repo := &fixtures.FakeWalletRepository{
		FindAccountByIdFunc: func(ctx context.Context, accountId string) (*types.AccountModel, error) {
			return &types.AccountModel{ID: accountId, UserID: "someone-else", Balance: 1000}, nil
		},
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			pendingCreated = true
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.WithdrawalRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 200, UserId: "user-1"})
	err := svc.HandleWithdrawalDispatched(context.Background(), "tx-1", payload)
	if err == nil {
		t.Fatalf("expected an error when the account does not belong to the user")
	}
	if pendingCreated {
		t.Fatalf("expected CreatePendingTransaction never to be called for a foreign account")
	}
}

func TestWalletService_HandleWithdrawalDispatched_InsufficientBalance_PendingTransactionNeverCreated(t *testing.T) {
	pendingCreated := false
	repo := &fixtures.FakeWalletRepository{
		FindAccountByIdFunc: func(ctx context.Context, accountId string) (*types.AccountModel, error) {
			return &types.AccountModel{ID: accountId, UserID: "user-1", Balance: 100}, nil
		},
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			pendingCreated = true
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.WithdrawalRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 500, UserId: "user-1"})
	err := svc.HandleWithdrawalDispatched(context.Background(), "tx-1", payload)
	if err == nil {
		t.Fatalf("expected an error for insufficient balance")
	}
	if pendingCreated {
		t.Fatalf("expected CreatePendingTransaction never to be called when the balance check fails first")
	}
}

func TestWalletService_HandleWithdrawalDispatched_WithdrawFundsError_MarksFailed(t *testing.T) {
	wantErr := errors.New("insufficient reconciliation")
	var finalStatus types.TransactionStatus
	repo := &fixtures.FakeWalletRepository{
		FindAccountByIdFunc: func(ctx context.Context, accountId string) (*types.AccountModel, error) {
			return &types.AccountModel{ID: accountId, UserID: "user-1", Balance: 1000, Currency: "USD"}, nil
		},
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			return nil
		},
		WithdrawFundsFunc: func(ctx context.Context, transactionId, accountId string, amount int64) (*types.TransactionModel, error) {
			return nil, wantErr
		},
		UpdateTransactionStatusFunc: func(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error {
			finalStatus = status
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.WithdrawalRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 200, UserId: "user-1"})
	err := svc.HandleWithdrawalDispatched(context.Background(), "tx-1", payload)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if finalStatus != types.TransactionStatusFailed {
		t.Fatalf("expected the transaction to be marked failed, got: %s", finalStatus)
	}
}

// NOTE: this pins down current behavior, which looks like a real bug — the guard
// rejects withdrawals whenever the ACCOUNT'S BALANCE exceeds 6000, regardless of
// how small the requested withdrawal amount is (wallet-serice.go:196). The comment
// above it ("Preventing withdrawal when the amount is more than 6000") suggests the
// intent was to cap the withdrawal amount, not gate on the pre-existing balance.
// If that's confirmed as a bug, this test's expectations should flip.
func TestWalletService_HandleWithdrawalDispatched_HighBalanceGuard_BlocksEvenSmallWithdrawal(t *testing.T) {
	var finalStatus types.TransactionStatus
	withdrawFundsCalled := false
	repo := &fixtures.FakeWalletRepository{
		FindAccountByIdFunc: func(ctx context.Context, accountId string) (*types.AccountModel, error) {
			return &types.AccountModel{ID: accountId, UserID: "user-1", Balance: 700000, Currency: "USD"}, nil
		},
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			return nil
		},
		WithdrawFundsFunc: func(ctx context.Context, transactionId, accountId string, amount int64) (*types.TransactionModel, error) {
			withdrawFundsCalled = true
			return &types.TransactionModel{ID: transactionId}, nil
		},
		UpdateTransactionStatusFunc: func(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error {
			finalStatus = status
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.WithdrawalRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-1", Amount: 100, UserId: "user-1"})
	err := svc.HandleWithdrawalDispatched(context.Background(), "tx-1", payload)
	if err == nil {
		t.Fatalf("expected the high-balance guard to reject this withdrawal (current behavior)")
	}
	if withdrawFundsCalled {
		t.Fatalf("expected WithdrawFunds never to be called once the high-balance guard trips")
	}
	if finalStatus != types.TransactionStatusFailed {
		t.Fatalf("expected the transaction to be marked failed, got: %s", finalStatus)
	}
}

// --- HandleTransferDispatched ---

func TestWalletService_HandleTransferDispatched_Success(t *testing.T) {
	var finalStatus types.TransactionStatus
	repo := &fixtures.FakeWalletRepository{
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			if transactionType != "transfer" || fromAccount != "account-1" || toAccount != "account-2" {
				t.Fatalf("unexpected pending transaction args: type=%s from=%s to=%s", transactionType, fromAccount, toAccount)
			}
			return nil
		},
		TransferFundsFunc: func(ctx context.Context, transactionId, fromAccountId, toAccountId, currency string, amount int64) (*types.TransactionModel, error) {
			return &types.TransactionModel{ID: transactionId, FromAccount: fromAccountId, ToAccount: toAccountId, Amount: amount, Currency: currency, Status: types.TransactionStatusCompleted}, nil
		},
		UpdateTransactionStatusFunc: func(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error {
			finalStatus = status
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.TransferRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-2", Amount: 300, Currency: "USD", UserId: "user-1"})
	err := svc.HandleTransferDispatched(context.Background(), "tx-1", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finalStatus != types.TransactionStatusCompleted {
		t.Fatalf("expected the transaction to be marked completed, got: %s", finalStatus)
	}
}

func TestWalletService_HandleTransferDispatched_InvalidAmount(t *testing.T) {
	svc := newWalletService(&fixtures.FakeWalletRepository{})

	payload := mustMarshal(t, domain.TransferRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-2", Amount: 0, Currency: "USD", UserId: "user-1"})
	err := svc.HandleTransferDispatched(context.Background(), "tx-1", payload)
	if err == nil {
		t.Fatalf("expected an error for a zero transfer amount")
	}
}

func TestWalletService_HandleTransferDispatched_CreatePendingTransactionError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	transferCalled := false
	repo := &fixtures.FakeWalletRepository{
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			return wantErr
		},
		TransferFundsFunc: func(ctx context.Context, transactionId, fromAccountId, toAccountId, currency string, amount int64) (*types.TransactionModel, error) {
			transferCalled = true
			return nil, nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.TransferRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-2", Amount: 300, Currency: "USD", UserId: "user-1"})
	err := svc.HandleTransferDispatched(context.Background(), "tx-1", payload)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if transferCalled {
		t.Fatalf("expected TransferFunds never to be called when creating the pending transaction fails")
	}
}

func TestWalletService_HandleTransferDispatched_TransferFundsError_MarksFailed(t *testing.T) {
	wantErr := errors.New("insufficient balance")
	var finalStatus types.TransactionStatus
	var failedAccountId string
	repo := &fixtures.FakeWalletRepository{
		CreatePendingTransactionFunc: func(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
			return nil
		},
		TransferFundsFunc: func(ctx context.Context, transactionId, fromAccountId, toAccountId, currency string, amount int64) (*types.TransactionModel, error) {
			return nil, wantErr
		},
		UpdateTransactionStatusFunc: func(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error {
			finalStatus = status
			failedAccountId = accountId
			return nil
		},
	}
	svc := newWalletService(repo)

	payload := mustMarshal(t, domain.TransferRequestedPayload{ID: "tx-1", FromAccount: "account-1", ToAccount: "account-2", Amount: 300, Currency: "USD", UserId: "user-1"})
	err := svc.HandleTransferDispatched(context.Background(), "tx-1", payload)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if finalStatus != types.TransactionStatusFailed {
		t.Fatalf("expected the transaction to be marked failed, got: %s", finalStatus)
	}
	if failedAccountId != "account-1" {
		t.Fatalf("expected the failure to be recorded against the sending account, got: %s", failedAccountId)
	}
}

// --- ListTransactions ---

func TestWalletService_ListTransactions_Success(t *testing.T) {
	repo := &fixtures.FakeWalletRepository{
		ListTransactionsFunc: func(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error) {
			return []types.TransactionModel{{ID: "tx-1"}, {ID: "tx-2"}}, nil
		},
	}
	svc := newWalletService(repo)

	resp, err := svc.ListTransactions(context.Background(), domain.ListTransactionsRequest{AccountID: "account-1", Page: 0, PageSize: 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Transactions) != 2 {
		t.Fatalf("unexpected transactions: %+v", resp.Transactions)
	}
}

func TestWalletService_ListTransactions_MissingAccountId(t *testing.T) {
	svc := newWalletService(&fixtures.FakeWalletRepository{})

	_, err := svc.ListTransactions(context.Background(), domain.ListTransactionsRequest{AccountID: ""})
	if err == nil {
		t.Fatalf("expected an error for a missing accountId")
	}
}

func TestWalletService_ListTransactions_PageSizeBelowMinimumDefaultsTo8(t *testing.T) {
	var gotLimit int
	repo := &fixtures.FakeWalletRepository{
		ListTransactionsFunc: func(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error) {
			gotLimit = limit
			return nil, nil
		},
	}
	svc := newWalletService(repo)

	_, err := svc.ListTransactions(context.Background(), domain.ListTransactionsRequest{AccountID: "account-1", PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLimit != 8 {
		t.Fatalf("expected the page size to default to 8, got: %d", gotLimit)
	}
}

func TestWalletService_ListTransactions_NegativePageDefaultsToZero(t *testing.T) {
	var gotOffset int
	repo := &fixtures.FakeWalletRepository{
		ListTransactionsFunc: func(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error) {
			gotOffset = offset
			return nil, nil
		},
	}
	svc := newWalletService(repo)

	_, err := svc.ListTransactions(context.Background(), domain.ListTransactionsRequest{AccountID: "account-1", Page: -5, PageSize: 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotOffset != 0 {
		t.Fatalf("expected a negative page to default to offset 0, got: %d", gotOffset)
	}
}

func TestWalletService_ListTransactions_NextPageTokenSetWhenFull(t *testing.T) {
	repo := &fixtures.FakeWalletRepository{
		ListTransactionsFunc: func(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error) {
			result := make([]types.TransactionModel, limit)
			return result, nil
		},
	}
	svc := newWalletService(repo)

	resp, err := svc.ListTransactions(context.Background(), domain.ListTransactionsRequest{AccountID: "account-1", Page: 1, PageSize: 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextPageToken != "2" {
		t.Fatalf("expected next page token %q, got: %q", "2", resp.NextPageToken)
	}
}

func TestWalletService_ListTransactions_NextPageTokenEmptyWhenPartial(t *testing.T) {
	repo := &fixtures.FakeWalletRepository{
		ListTransactionsFunc: func(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error) {
			return []types.TransactionModel{{ID: "tx-1"}}, nil
		},
	}
	svc := newWalletService(repo)

	resp, err := svc.ListTransactions(context.Background(), domain.ListTransactionsRequest{AccountID: "account-1", Page: 0, PageSize: 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextPageToken != "" {
		t.Fatalf("expected no next page token for a partial page, got: %q", resp.NextPageToken)
	}
}

func TestWalletService_ListTransactions_RepositoryError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	repo := &fixtures.FakeWalletRepository{
		ListTransactionsFunc: func(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error) {
			return nil, wantErr
		},
	}
	svc := newWalletService(repo)

	_, err := svc.ListTransactions(context.Background(), domain.ListTransactionsRequest{AccountID: "account-1", PageSize: 8})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}
