package fixtures

import (
	"context"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"
)

type FakeWalletServiceServer struct {
	pbWallet.UnimplementedWalletServiceServer
	CreateAccountFunc    func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error)
	GetBalanceFunc       func(ctx context.Context, in *pbWallet.GetBalanceRequest) (*pbWallet.GetBalanceResponse, error)
	DepositFunc          func(ctx context.Context, in *pbWallet.DepositRequest) (*pbWallet.DepositResponse, error)
	WithdrawFunc         func(ctx context.Context, in *pbWallet.WithdrawRequest) (*pbWallet.WithdrawResponse, error)
	TransferFunc         func(ctx context.Context, in *pbWallet.TransferRequest) (*pbWallet.TransferResponse, error)
	ListTransactionsFunc func(ctx context.Context, in *pbWallet.ListTransactionsRequest) (*pbWallet.ListTransactionsResponse, error)
}

var _ pbWallet.WalletServiceServer = (*FakeWalletServiceServer)(nil)

func (f *FakeWalletServiceServer) CreateAccount(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
	if f.CreateAccountFunc == nil {
		return f.UnimplementedWalletServiceServer.CreateAccount(ctx, in)
	}
	return f.CreateAccountFunc(ctx, in)
}

func (f *FakeWalletServiceServer) GetBalance(ctx context.Context, in *pbWallet.GetBalanceRequest) (*pbWallet.GetBalanceResponse, error) {
	if f.GetBalanceFunc == nil {
		return f.UnimplementedWalletServiceServer.GetBalance(ctx, in)
	}
	return f.GetBalanceFunc(ctx, in)
}

func (f *FakeWalletServiceServer) Deposit(ctx context.Context, in *pbWallet.DepositRequest) (*pbWallet.DepositResponse, error) {
	if f.DepositFunc == nil {
		return f.UnimplementedWalletServiceServer.Deposit(ctx, in)
	}
	return f.DepositFunc(ctx, in)
}

func (f *FakeWalletServiceServer) Withdraw(ctx context.Context, in *pbWallet.WithdrawRequest) (*pbWallet.WithdrawResponse, error) {
	if f.WithdrawFunc == nil {
		return f.UnimplementedWalletServiceServer.Withdraw(ctx, in)
	}
	return f.WithdrawFunc(ctx, in)
}

func (f *FakeWalletServiceServer) Transfer(ctx context.Context, in *pbWallet.TransferRequest) (*pbWallet.TransferResponse, error) {
	if f.TransferFunc == nil {
		return f.UnimplementedWalletServiceServer.Transfer(ctx, in)
	}
	return f.TransferFunc(ctx, in)
}

func (f *FakeWalletServiceServer) ListTransactions(ctx context.Context, in *pbWallet.ListTransactionsRequest) (*pbWallet.ListTransactionsResponse, error) {
	if f.ListTransactionsFunc == nil {
		return f.UnimplementedWalletServiceServer.ListTransactions(ctx, in)
	}
	return f.ListTransactionsFunc(ctx, in)
}
