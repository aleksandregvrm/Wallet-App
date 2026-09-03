package services

import (
	"context"
	"encoding/json"
	"fmt"
	"go-task-wallet-service/shared/events"
	"go-task-wallet-service/shared/logging"
	models "go-task-wallet-service/shared/pkg/models"
	"go-task-wallet-service/wallet-service/internal/domain"
	"strconv"
)

type WalletService struct {
	walletRepository domain.WalletRepository
	eventPublisher   domain.EventPublisher
	logging.Logger
}

func NewWalletService(walletRepository domain.WalletRepository, eventPublisher domain.EventPublisher) *WalletService {
	return &WalletService{
		walletRepository: walletRepository,
		eventPublisher:   eventPublisher,
		Logger:           logging.NewInternalLogger(),
	}
}

// method to check whether a given user already has 3 accounts of the same currency. If so another account with the same currency is not permitted
func (s *WalletService) canOpenNewAccounts(alreadyOpenedAccounts []models.AccountModel, currency string) bool {
	count := 0
	for _, acc := range alreadyOpenedAccounts {
		if acc.Currency == currency {
			count++
		}
	}
	return count < 3
}

// Check whether provided account belongs to the given user
// Example use: withdrawals, to make sure user cannot withdraw from someone elses account
// And cannot check the balance of someone elses account
func (s *WalletService) checkUserAccountCompatibility(ctx context.Context, userId, accountId string) (*models.AccountModel, bool) {
	account, err := s.walletRepository.FindAccountById(ctx, accountId)
	if err != nil {
		return nil, false
	}

	if account.UserID != userId {
		return nil, false
	}

	return account, true
}

// Dispatched domain event handlers
func (s *WalletService) HandleDepositDispatched(ctx context.Context, aggregateId string, payload json.RawMessage) error {
	var depositPayload domain.DepositRequestedPayload
	if err := json.Unmarshal(payload, &depositPayload); err != nil {
		return fmt.Errorf("%w: decode deposit.requested payload for aggregate_id=%s: %v", events.ErrInvalidMessage, aggregateId, err)
	}

	_, err := s.deposit(ctx, depositPayload.ID, depositPayload.UserId, depositPayload.ToAccount, depositPayload.Currency, depositPayload.Amount)
	return err
}

func (s *WalletService) HandleWithdrawalDispatched(ctx context.Context, aggregateId string, payload json.RawMessage) error {
	var withdrawalPayload domain.WithdrawalRequestedPayload
	if err := json.Unmarshal(payload, &withdrawalPayload); err != nil {
		return fmt.Errorf("%w: decode withdrawal.requested payload for aggregate_id=%s: %v", events.ErrInvalidMessage, aggregateId, err)
	}

	_, err := s.withdraw(ctx, withdrawalPayload.ID, withdrawalPayload.UserId, withdrawalPayload.ToAccount, withdrawalPayload.Amount)
	return err
}

func (s *WalletService) HandleTransferDispatched(ctx context.Context, aggregateId string, payload json.RawMessage) error {
	var transferPayload domain.TransferRequestedPayload
	if err := json.Unmarshal(payload, &transferPayload); err != nil {
		return fmt.Errorf("%w: decode transfer.requested payload for aggregate_id=%s: %v", events.ErrInvalidMessage, aggregateId, err)
	}

	_, err := s.transfer(ctx, transferPayload.ID, transferPayload.UserId, transferPayload.FromAccount, transferPayload.ToAccount, transferPayload.Currency, transferPayload.Amount)
	return err
}

func (s *WalletService) OpenAccount(ctx context.Context, userId, currency string) (*domain.Account, error) {
	if userId == "" {
		return nil, fmt.Errorf("userId is required to open an account")
	}
	if currency == "" {
		return nil, fmt.Errorf("currency is required to open an account")
	}

	alreadyOpenedAccounts, err := s.walletRepository.FindAccountsByUserId(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing accounts for user %s: %w", userId, err)
	}
	if !s.canOpenNewAccounts(alreadyOpenedAccounts, currency) {
		return nil, fmt.Errorf("cannot create more than three accounts on currency: %s", currency)
	}

	account, err := s.walletRepository.InsertAccount(ctx, userId, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to open account for user %s: %w", userId, err)
	}

	return &domain.Account{
		ID:        account.ID,
		UserID:    account.UserID,
		Balance:   account.Balance,
		Currency:  account.Currency,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	}, nil
}
func (s *WalletService) UpdateAccount(fromAccountId, toAccountId string) (*domain.Account, error) {
	return nil, nil
}

func (s *WalletService) GetBalance(ctx context.Context, userId, accountId string) (*domain.Account, error) {
	if accountId == "" {
		return nil, fmt.Errorf("AccountId is required to get balance")
	}

	account, isCompatible := s.checkUserAccountCompatibility(ctx, userId, accountId)
	if !isCompatible {
		return nil, fmt.Errorf("Account %s does not belong to user %s", accountId, userId)
	}

	// Checking whether the current balance is cached if not writing the current balance in cache to be retrieved next time
	balance, err := s.walletRepository.GetOrStoreAccountBalance(ctx, accountId, account.Balance)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance for account %s: %w", accountId, err)
	}

	return &domain.Account{
		ID:        account.ID,
		UserID:    account.UserID,
		Balance:   balance,
		Currency:  account.Currency,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	}, nil
}

func (s *WalletService) deposit(ctx context.Context, transactionId, userId, accountId, currency string, amount int64) (*domain.DepositResponse, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("Cannot deposit 0 or negative amount to the account %s", accountId)
	}

	if err := s.walletRepository.CreatePendingTransaction(ctx, transactionId, "deposit", accountId, accountId, currency, amount); err != nil {
		return nil, fmt.Errorf("Failed to create pending deposit transaction: %w", err)
	}

	transaction, err := s.walletRepository.DepositFunds(ctx, transactionId, accountId, currency, amount)
	if err != nil {
		s.walletRepository.UpdateTransactionStatus(ctx, transactionId, userId, accountId, models.TransactionStatusFailed)
		return nil, fmt.Errorf("Failed to deposit funds to account: %s, error:%w", accountId, err)
	}

	s.walletRepository.UpdateTransactionStatus(ctx, transactionId, userId, accountId, models.TransactionStatusCompleted)

	s.LogInfo(ctx, "Users - %s, Account - %s, has deposited amount:%d, currency:%s", userId, accountId, amount, currency)

	return &domain.DepositResponse{
		Transaction: domain.Transaction{
			ID:          transaction.ID,
			FromAccount: transaction.FromAccount,
			ToAccount:   transaction.ToAccount,
			Amount:      transaction.Amount,
			Currency:    transaction.Currency,
			Status:      transaction.Status,
			CreatedAt:   transaction.CreatedAt,
			UpdatedAt:   transaction.UpdatedAt,
		},
	}, nil
}

func (s *WalletService) withdraw(ctx context.Context, transactionId, userId, accountId string, amount int64) (*domain.WithdrawResponse, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("Cannot withdraw 0 or negative amount from the account %s", accountId)
	}

	account, isCompatible := s.checkUserAccountCompatibility(ctx, userId, accountId)

	if !isCompatible {
		return nil, fmt.Errorf("Account %s does not belong to user %s", accountId, userId)
	}

	// Balance check
	// We already have this implemented in the repository layer, this serves as an additional layer of security
	if account.Balance < amount {
		// Private functions should be called to handle transaction failure in here
		s.LogWarn(ctx, "account: %s insufficient balance. amount: %d current balance is: %d", accountId, amount, account.Balance)
		return nil, fmt.Errorf("Insufficient balance for withdrawal, requested amount:%d, available balance: %d", amount, account.Balance)
	}

	if err := s.walletRepository.CreatePendingTransaction(ctx, transactionId, "withdrawal", accountId, accountId, account.Currency, amount); err != nil {
		return nil, fmt.Errorf("Failed to create pending withdrawal transaction: %w", err)
	}

	// Preventing withdrawal when the amount is more than 6000.
	if account.Balance > int64(600000) {
		s.LogWarn(ctx, "account: %s cannot withdraw because the current balance is: %d", accountId, account.Balance)
		s.walletRepository.UpdateTransactionStatus(ctx, transactionId, userId, accountId, models.TransactionStatusFailed)
		return nil, fmt.Errorf("Cannot withdraw from account %s while balance exceeds 6000", accountId)
	}

	transaction, err := s.walletRepository.WithdrawFunds(ctx, transactionId, accountId, amount)
	if err != nil {
		s.walletRepository.UpdateTransactionStatus(ctx, transactionId, userId, accountId, models.TransactionStatusFailed)
		return nil, fmt.Errorf("Failed to withdraw funds from account: %s, error:%w", accountId, err)
	}

	s.walletRepository.UpdateTransactionStatus(ctx, transactionId, userId, accountId, models.TransactionStatusCompleted)

	s.LogInfo(ctx, "Users - %s, Account - %s, has withdrawn amount:%d", userId, accountId, amount)

	return &domain.WithdrawResponse{
		Transaction: domain.Transaction{
			ID:          transaction.ID,
			FromAccount: transaction.FromAccount,
			ToAccount:   transaction.ToAccount,
			Amount:      transaction.Amount,
			Currency:    transaction.Currency,
			Status:      transaction.Status,
			CreatedAt:   transaction.CreatedAt,
			UpdatedAt:   transaction.UpdatedAt,
		},
	}, nil
}

func (s *WalletService) transfer(ctx context.Context, transactionId, userId, fromAccountId, toAccountId, currency string, amount int64) (*domain.TransferResponse, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("Cannot transfer 0 or negative amount from account %s", fromAccountId)
	}

	if err := s.walletRepository.CreatePendingTransaction(ctx, transactionId, "transfer", fromAccountId, toAccountId, currency, amount); err != nil {
		return nil, fmt.Errorf("Failed to create pending transfer transaction: %w", err)
	}

	transaction, err := s.walletRepository.TransferFunds(ctx, transactionId, fromAccountId, toAccountId, currency, amount)
	if err != nil {
		s.walletRepository.UpdateTransactionStatus(ctx, transactionId, userId, fromAccountId, models.TransactionStatusFailed)
		return nil, fmt.Errorf("Failed to transfer funds from account: %s, error:%w", fromAccountId, err)
	}

	s.walletRepository.UpdateTransactionStatus(ctx, transactionId, userId, fromAccountId, models.TransactionStatusCompleted)

	s.LogInfo(ctx, "Users - %s, Account - %s, has transferred amount:%d to account:%s", userId, fromAccountId, amount, toAccountId)

	return &domain.TransferResponse{
		Transaction: domain.Transaction{
			ID:          transaction.ID,
			FromAccount: transaction.FromAccount,
			ToAccount:   transaction.ToAccount,
			Amount:      transaction.Amount,
			Currency:    transaction.Currency,
			Status:      transaction.Status,
			CreatedAt:   transaction.CreatedAt,
			UpdatedAt:   transaction.UpdatedAt,
		},
	}, nil
}

func (s *WalletService) ListTransactions(ctx context.Context, listTransactionsRequest domain.ListTransactionsRequest) (*domain.ListTransactionsResponse, error) {
	if listTransactionsRequest.AccountID == "" {
		return nil, fmt.Errorf("accountId is required to list transactions")
	}

	pageSize := listTransactionsRequest.PageSize
	if pageSize < 8 {
		pageSize = 8 // Fallback to min 8 of page size
	}

	page := listTransactionsRequest.Page
	if page < 0 {
		page = 0 // Defaulting to page 0
	}

	// Calculating offset based on the current page and the size of the page
	offset := int(page) * int(pageSize)

	transactionsList, err := s.walletRepository.ListTransactions(ctx, listTransactionsRequest.AccountID, offset, int(pageSize))
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions for account %s: %w", listTransactionsRequest.AccountID, err)
	}

	// Making a slice for efficient population for the list of transactions
	transactions := make([]domain.Transaction, 0, len(transactionsList))
	for _, t := range transactionsList {
		transactions = append(transactions, domain.Transaction{
			ID:          t.ID,
			FromAccount: t.FromAccount,
			ToAccount:   t.ToAccount,
			Amount:      t.Amount,
			Currency:    t.Currency,
			Status:      t.Status,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		})
	}

	// Calculating next page based on the transactions list and the current page
	var nextPageToken string
	if len(transactionsList) == int(pageSize) {
		nextPageToken = strconv.Itoa(int(page) + 1)
	}

	return &domain.ListTransactionsResponse{
		Transactions:  transactions,
		NextPageToken: nextPageToken,
	}, nil
}
