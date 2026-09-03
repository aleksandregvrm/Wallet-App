package grpc

import (
	"context"
	"fmt"
	"go-task-wallet-service/shared/events"
	"go-task-wallet-service/shared/logging"
	types "go-task-wallet-service/shared/pkg/models"
	pb "go-task-wallet-service/shared/proto/impl/wallet"
	"go-task-wallet-service/wallet-service/internal/domain"
	"go-task-wallet-service/wallet-service/internal/mapping"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type gRPCHandler struct {
	pb.UnimplementedWalletServiceServer
	Service            domain.WalletService
	outboxRelayService domain.OutboxRelayService
	logging.Logger
}

// gRPC module declaration
// Wiring connection to other service gRPC clients. In this case auth-service only communicates front to back to the api-gateway service. So we only need to establish a connection to the api-gateway service.
func NewGrpcHandler(s *grpc.Server, service domain.WalletService, outboxRelayService domain.OutboxRelayService) {
	handler := &gRPCHandler{
		Service:            service,
		outboxRelayService: outboxRelayService,
		Logger:             logging.NewInternalLogger(),
	}
	pb.RegisterWalletServiceServer(s, handler)
}

func (h *gRPCHandler) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	ctx = logging.WithRequestID(ctx, req.GetOwnerUser())

	account, err := h.Service.OpenAccount(ctx, req.GetOwnerUser(), req.GetCurrency())
	if err != nil {
		return nil, err
	}

	return &pb.CreateAccountResponse{
		Account: mapping.ToAccountProto(account),
	}, nil
}

func (h *gRPCHandler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	ctx = logging.WithRequestID(ctx, req.GetAccountId())

	account, err := h.Service.GetBalance(ctx, req.GetUserId(), req.GetAccountId())
	if err != nil {
		return nil, err
	}

	return &pb.GetBalanceResponse{
		Balance:  account.Balance,
		Currency: account.Currency,
	}, nil
}

func (h *gRPCHandler) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	// Defined idempotency key - transactionId
	transactionId := uuid.NewString()

	ctx = logging.WithRequestID(ctx, transactionId)

	// Returning Create transaction struct with initialized status since the actual processing is asynchronous
	pendingTransaction := &domain.Transaction{
		ID:          transactionId,
		ToAccount:   req.GetAccountId(),
		FromAccount: req.GetAccountId(),
		Amount:      req.GetAmount(),
		Currency:    req.GetCurrency(),
		Status:      types.TransactionStatusPending,
	}

	// Constructing the payload map
	payload := map[string]interface{}{
		"ID":          pendingTransaction.ID,
		"FromAccount": pendingTransaction.FromAccount,
		"ToAccount":   pendingTransaction.ToAccount,
		"Amount":      pendingTransaction.Amount,
		"Currency":    pendingTransaction.Currency,
		"UserId":      req.GetUserId(),
	}

	if err := h.outboxRelayService.InsertOutboxRelay(ctx, "transaction", transactionId, events.TransactionDepositRequested, req.GetAccountId(), req.GetUserId(), payload); err != nil {
		return nil, err
	}

	return &pb.DepositResponse{
		Transaction: mapping.ToTransactionProto(pendingTransaction),
	}, nil
}

func (h *gRPCHandler) Withdraw(ctx context.Context, req *pb.WithdrawRequest) (*pb.WithdrawResponse, error) {
	transactionId := req.GetTransactionId()
	if transactionId == "" {
		transactionId = uuid.NewString()
	}

	ctx = logging.WithRequestID(ctx, transactionId)

	// Returning Create transaction struct with initialized status since the actual processing is asynchronous
	pendingTransaction := &domain.Transaction{
		ID:          transactionId,
		ToAccount:   req.GetAccountId(),
		FromAccount: req.GetAccountId(),
		Amount:      req.GetAmount(),
		Status:      types.TransactionStatusPending,
	}

	// Constructing the payload map
	payload := map[string]interface{}{
		"ID":          pendingTransaction.ID,
		"FromAccount": pendingTransaction.FromAccount,
		"ToAccount":   pendingTransaction.ToAccount,
		"Amount":      pendingTransaction.Amount,
		"UserId":      req.GetUserId(),
	}

	if err := h.outboxRelayService.InsertOutboxRelay(ctx, "transaction", transactionId, events.TransactionWithdrawalRequested, req.GetAccountId(), req.GetUserId(), payload); err != nil {
		return nil, err
	}

	return &pb.WithdrawResponse{
		Transaction: mapping.ToTransactionProto(pendingTransaction),
	}, nil
}

func (h *gRPCHandler) Transfer(ctx context.Context, req *pb.TransferRequest) (*pb.TransferResponse, error) {
	transactionId := req.GetIdempotencyKey()
	if transactionId == "" {
		transactionId = uuid.NewString()
	}

	ctx = logging.WithRequestID(ctx, transactionId)

	// Returning Create transaction struct with initialized status since the actual processing is asynchronous
	pendingTransaction := &domain.Transaction{
		ID:          transactionId,
		FromAccount: req.GetFromAccountId(),
		ToAccount:   req.GetToAccountId(),
		Amount:      req.GetAmount(),
		Currency:    req.GetCurrency(),
		Status:      types.TransactionStatusPending,
	}

	// Constructing the payload map
	payload := map[string]interface{}{
		"ID":          pendingTransaction.ID,
		"FromAccount": pendingTransaction.FromAccount,
		"ToAccount":   pendingTransaction.ToAccount,
		"Amount":      pendingTransaction.Amount,
		"Currency":    pendingTransaction.Currency,
		"UserId":      req.GetUserId(),
	}

	if err := h.outboxRelayService.InsertOutboxRelay(ctx, "transaction", transactionId, events.TransactionTransferRequested, req.GetFromAccountId(), req.GetUserId(), payload); err != nil {
		return nil, err
	}

	return &pb.TransferResponse{
		Transaction: mapping.ToTransactionProto(pendingTransaction),
	}, nil
}

func (h *gRPCHandler) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	ctx = logging.WithRequestID(ctx, req.GetAccountId())

	domainListTransactionsRequest := mapping.ToListTransactionsDomain(req)

	transactionsList, err := h.Service.ListTransactions(ctx, *domainListTransactionsRequest)
	if err != nil {
		return nil, err
	}

	pbListTransactionsResponse := mapping.ToListTransactionsProto(transactionsList)

	return pbListTransactionsResponse, nil
}

func (h *gRPCHandler) StreamTransactions(req *pb.StreamTransactionsRequest, rpc grpc.ServerStreamingServer[pb.Transaction]) error {
	return fmt.Errorf("Method not yet implemented")
}
