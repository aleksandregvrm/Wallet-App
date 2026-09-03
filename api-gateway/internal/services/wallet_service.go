package services

import (
	"context"
	"go-task-wallet-service/api-gateway/internal/domain"
	infra "go-task-wallet-service/api-gateway/internal/infra/grpc"
	"go-task-wallet-service/api-gateway/internal/mapping"
	"go-task-wallet-service/shared/logging"

	"github.com/google/uuid"
)

type WalletService struct {
	grpc *infra.GrpcHandler
	logging.Logger
}

func NewWalletService(grpc *infra.GrpcHandler) *WalletService {
	return &WalletService{
		grpc:   grpc,
		Logger: logging.NewInternalLogger(),
	}
}

func (s *WalletService) TransferFunds(ctx context.Context, userId, fromAccountID, toAccountID, currency string, amount int64) (*domain.Transaction, error) {
	idempotencyKey := uuid.NewString()

	pbTransferFunds := mapping.ToTransferFundsProto(userId, fromAccountID, toAccountID, currency, amount, idempotencyKey)

	transferFundsResponsePb, err := s.grpc.WalletClient.Transfer(ctx, pbTransferFunds)
	if err != nil {
		s.LogInfo(ctx, "transfer failed from accountId:%s to accountId:%s, reason:%v", fromAccountID, toAccountID, err)
		return nil, err
	}

	transaction := mapping.ToTransactionDomain(transferFundsResponsePb.Transaction)

	s.LogInfo(ctx, "Transfer initialized from - %s to - %s", transaction.FromAccount, transaction.ToAccount)
	return transaction, nil
}

func (s *WalletService) TransferFundsWithEmailNotification(ctx context.Context, fromEmail, toEmail, currency string, amount int64) error {
	return nil
}

func (s *WalletService) DepositFunds(ctx context.Context, userId, accountID, currency string, amount int64) (*domain.Transaction, error) {
	pbDepositFunds := mapping.ToDepositFundsProto(userId, accountID, currency, amount)

	depositFundsResponsePb, err := s.grpc.WalletClient.Deposit(ctx, pbDepositFunds)

	if err != nil {
		s.LogInfo(ctx, "deposit failed to accountId:%s, reason:%v", accountID, err)
		return nil, err
	}

	transaction := mapping.ToTransactionDomain(depositFundsResponsePb.Transaction)

	s.LogInfo(ctx, "Deposit initialized to - %s", transaction.ToAccount)
	return transaction, nil
}

func (s *WalletService) WithdrawFunds(ctx context.Context, userId, accountID string, amount int64) (*domain.Transaction, error) {
	transactionId := uuid.NewString()

	pbWithdrawFunds := mapping.ToWithdrawFundsProto(userId, accountID, amount, transactionId)

	withdrawFundsResponsePb, err := s.grpc.WalletClient.Withdraw(ctx, pbWithdrawFunds)
	if err != nil {
		s.LogInfo(ctx, "withdrawal failed to accountId:%s, reason:%v", accountID, err)
		return nil, err
	}

	transaction := mapping.ToTransactionDomain(withdrawFundsResponsePb.Transaction)

	s.LogInfo(ctx, "Withdrawal initialized from - %s", transaction.FromAccount)
	return transaction, nil
}

func (s *WalletService) GetBalance(ctx context.Context, userId, accountID string) (*domain.Balance, error) {
	pbGetBalance := mapping.ToGetBalanceProto(userId, accountID)

	getBalanceResponsePb, err := s.grpc.WalletClient.GetBalance(ctx, pbGetBalance)
	if err != nil {
		s.LogInfo(ctx, "get balance failed for accountId:%s, reason:%v", accountID, err)
		return nil, err
	}

	return &domain.Balance{
		AccountID: accountID,
		Balance:   getBalanceResponsePb.GetBalance(),
		Currency:  getBalanceResponsePb.GetCurrency(),
	}, nil
}

func (s *WalletService) ListTransactions(ctx context.Context, accountID string, page int32, pageSize int8) (*domain.ListTransactionsResponse, error) {
	pbListTransactions := mapping.ToListTransactionsProto(accountID, page, pageSize)

	listTransactionsResponsePb, err := s.grpc.WalletClient.ListTransactions(ctx, pbListTransactions)

	if err != nil {
		s.LogInfo(ctx, "Retrieving transactions list has failed for accountId:%s, reason:%v", accountID, err)
		return nil, err
	}

	listTransactionsResponse := mapping.ToListTransactionsDomain(listTransactionsResponsePb)

	return listTransactionsResponse, nil

}
