package fixtures

import (
	"context"
	"fmt"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"

	"google.golang.org/grpc"
)

// Registering all wallet related methods of account creation, fun transfer, deposit, withdrawal and etc.
// Mocking the api-gateways functionality of gRPC client

type FakeWalletServiceClient struct {
	CreateAccountFunc      func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error)
	GetBalanceFunc         func(ctx context.Context, in *pbWallet.GetBalanceRequest) (*pbWallet.GetBalanceResponse, error)
	DepositFunc            func(ctx context.Context, in *pbWallet.DepositRequest) (*pbWallet.DepositResponse, error)
	WithdrawFunc           func(ctx context.Context, in *pbWallet.WithdrawRequest) (*pbWallet.WithdrawResponse, error)
	TransferFunc           func(ctx context.Context, in *pbWallet.TransferRequest) (*pbWallet.TransferResponse, error)
	ListTransactionsFunc   func(ctx context.Context, in *pbWallet.ListTransactionsRequest) (*pbWallet.ListTransactionsResponse, error)
	StreamTransactionsFunc func(ctx context.Context, in *pbWallet.StreamTransactionsRequest) (grpc.ServerStreamingClient[pbWallet.Transaction], error)
}

var _ pbWallet.WalletServiceClient = (*FakeWalletServiceClient)(nil)

func (f *FakeWalletServiceClient) CreateAccount(ctx context.Context, in *pbWallet.CreateAccountRequest, _ ...grpc.CallOption) (*pbWallet.CreateAccountResponse, error) {
	if f.CreateAccountFunc == nil {
		return nil, fmt.Errorf("fixtures: CreateAccountFunc not set")
	}
	return f.CreateAccountFunc(ctx, in)
}

func (f *FakeWalletServiceClient) GetBalance(ctx context.Context, in *pbWallet.GetBalanceRequest, _ ...grpc.CallOption) (*pbWallet.GetBalanceResponse, error) {
	if f.GetBalanceFunc == nil {
		return nil, fmt.Errorf("fixtures: GetBalanceFunc not set")
	}
	return f.GetBalanceFunc(ctx, in)
}

func (f *FakeWalletServiceClient) Deposit(ctx context.Context, in *pbWallet.DepositRequest, _ ...grpc.CallOption) (*pbWallet.DepositResponse, error) {
	if f.DepositFunc == nil {
		return nil, fmt.Errorf("fixtures: DepositFunc not set")
	}
	return f.DepositFunc(ctx, in)
}

func (f *FakeWalletServiceClient) Withdraw(ctx context.Context, in *pbWallet.WithdrawRequest, _ ...grpc.CallOption) (*pbWallet.WithdrawResponse, error) {
	if f.WithdrawFunc == nil {
		return nil, fmt.Errorf("fixtures: WithdrawFunc not set")
	}
	return f.WithdrawFunc(ctx, in)
}

func (f *FakeWalletServiceClient) Transfer(ctx context.Context, in *pbWallet.TransferRequest, _ ...grpc.CallOption) (*pbWallet.TransferResponse, error) {
	if f.TransferFunc == nil {
		return nil, fmt.Errorf("fixtures: TransferFunc not set")
	}
	return f.TransferFunc(ctx, in)
}

func (f *FakeWalletServiceClient) ListTransactions(ctx context.Context, in *pbWallet.ListTransactionsRequest, _ ...grpc.CallOption) (*pbWallet.ListTransactionsResponse, error) {
	if f.ListTransactionsFunc == nil {
		return nil, fmt.Errorf("fixtures: ListTransactionsFunc not set")
	}
	return f.ListTransactionsFunc(ctx, in)
}

func (f *FakeWalletServiceClient) StreamTransactions(ctx context.Context, in *pbWallet.StreamTransactionsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pbWallet.Transaction], error) {
	if f.StreamTransactionsFunc == nil {
		return nil, fmt.Errorf("fixtures: StreamTransactionsFunc not set")
	}
	return f.StreamTransactionsFunc(ctx, in)
}
