package unit_tests

import (
	"context"
	"errors"
	"testing"

	infra "go-task-wallet-service/api-gateway/internal/infra/grpc"
	"go-task-wallet-service/api-gateway/internal/services"
	"go-task-wallet-service/api-gateway/tests/fixtures"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"
)

func newWalletService(fake *fixtures.FakeWalletServiceClient) *services.WalletService {
	return services.NewWalletService(&infra.GrpcHandler{WalletClient: fake})
}

func TestWalletService_DepositFunds_Success(t *testing.T) {
	fake := &fixtures.FakeWalletServiceClient{
		DepositFunc: func(ctx context.Context, in *pbWallet.DepositRequest) (*pbWallet.DepositResponse, error) {
			if in.GetUserId() != "user-1" || in.GetAccountId() != "account-1" || in.GetCurrency() != "USD" || in.GetAmount() != 500 {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.DepositResponse{
				Transaction: &pbWallet.Transaction{
					Id: "txn-1", FromAccount: "", ToAccount: "account-1",
					Amount: 500, Currency: "USD", Status: "pending",
				},
			}, nil
		},
	}
	svc := newWalletService(fake)

	txn, err := svc.DepositFunds(context.Background(), "user-1", "account-1", "USD", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txn.ID != "txn-1" || txn.ToAccount != "account-1" || txn.Amount != 500 || txn.Status != "pending" {
		t.Fatalf("unexpected transaction: %+v", txn)
	}
}

func TestWalletService_DepositFunds_GrpcError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fixtures.FakeWalletServiceClient{
		DepositFunc: func(ctx context.Context, in *pbWallet.DepositRequest) (*pbWallet.DepositResponse, error) {
			return nil, wantErr
		},
	}
	svc := newWalletService(fake)

	txn, err := svc.DepositFunds(context.Background(), "user-1", "account-1", "USD", 500)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if txn != nil {
		t.Fatalf("expected nil transaction on error, got: %+v", txn)
	}
}

func TestWalletService_WithdrawFunds_Success(t *testing.T) {
	fake := &fixtures.FakeWalletServiceClient{
		WithdrawFunc: func(ctx context.Context, in *pbWallet.WithdrawRequest) (*pbWallet.WithdrawResponse, error) {
			if in.GetUserId() != "user-1" || in.GetAccountId() != "account-1" || in.GetAmount() != 200 {
				t.Fatalf("unexpected request: %+v", in)
			}
			if in.GetTransactionId() == "" {
				t.Fatalf("expected a generated idempotency key, got empty string")
			}
			return &pbWallet.WithdrawResponse{
				Transaction: &pbWallet.Transaction{
					Id: "txn-2", FromAccount: "account-1", Amount: 200, Currency: "USD", Status: "pending",
				},
			}, nil
		},
	}
	svc := newWalletService(fake)

	txn, err := svc.WithdrawFunds(context.Background(), "user-1", "account-1", 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txn.ID != "txn-2" || txn.FromAccount != "account-1" || txn.Amount != 200 {
		t.Fatalf("unexpected transaction: %+v", txn)
	}
}

func TestWalletService_WithdrawFunds_GrpcError(t *testing.T) {
	wantErr := errors.New("insufficient funds")
	fake := &fixtures.FakeWalletServiceClient{
		WithdrawFunc: func(ctx context.Context, in *pbWallet.WithdrawRequest) (*pbWallet.WithdrawResponse, error) {
			return nil, wantErr
		},
	}
	svc := newWalletService(fake)

	txn, err := svc.WithdrawFunds(context.Background(), "user-1", "account-1", 200)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if txn != nil {
		t.Fatalf("expected nil transaction on error, got: %+v", txn)
	}
}

func TestWalletService_TransferFunds_Success(t *testing.T) {
	fake := &fixtures.FakeWalletServiceClient{
		TransferFunc: func(ctx context.Context, in *pbWallet.TransferRequest) (*pbWallet.TransferResponse, error) {
			if in.GetUserId() != "user-1" || in.GetFromAccountId() != "account-1" || in.GetToAccountId() != "account-2" {
				t.Fatalf("unexpected request: %+v", in)
			}
			if in.GetIdempotencyKey() == "" {
				t.Fatalf("expected a generated idempotency key, got empty string")
			}
			return &pbWallet.TransferResponse{
				Transaction: &pbWallet.Transaction{
					Id: "txn-3", FromAccount: "account-1", ToAccount: "account-2",
					Amount: 300, Currency: "USD", Status: "pending",
				},
			}, nil
		},
	}
	svc := newWalletService(fake)

	txn, err := svc.TransferFunds(context.Background(), "user-1", "account-1", "account-2", "USD", 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txn.ID != "txn-3" || txn.FromAccount != "account-1" || txn.ToAccount != "account-2" || txn.Amount != 300 {
		t.Fatalf("unexpected transaction: %+v", txn)
	}
}

func TestWalletService_TransferFunds_GrpcError(t *testing.T) {
	wantErr := errors.New("account not found")
	fake := &fixtures.FakeWalletServiceClient{
		TransferFunc: func(ctx context.Context, in *pbWallet.TransferRequest) (*pbWallet.TransferResponse, error) {
			return nil, wantErr
		},
	}
	svc := newWalletService(fake)

	txn, err := svc.TransferFunds(context.Background(), "user-1", "account-1", "account-2", "USD", 300)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if txn != nil {
		t.Fatalf("expected nil transaction on error, got: %+v", txn)
	}
}

func TestWalletService_GetBalance_Success(t *testing.T) {
	fake := &fixtures.FakeWalletServiceClient{
		GetBalanceFunc: func(ctx context.Context, in *pbWallet.GetBalanceRequest) (*pbWallet.GetBalanceResponse, error) {
			if in.GetUserId() != "user-1" || in.GetAccountId() != "account-1" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.GetBalanceResponse{Balance: 4200, Currency: "USD"}, nil
		},
	}
	svc := newWalletService(fake)

	bal, err := svc.GetBalance(context.Background(), "user-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// AccountID is populated from the input param, not the proto response
	// (GetBalanceResponse carries no account_id field) - pin that behavior.
	if bal.AccountID != "account-1" || bal.Balance != 4200 || bal.Currency != "USD" {
		t.Fatalf("unexpected balance: %+v", bal)
	}
}

func TestWalletService_GetBalance_GrpcError(t *testing.T) {
	wantErr := errors.New("account not found")
	fake := &fixtures.FakeWalletServiceClient{
		GetBalanceFunc: func(ctx context.Context, in *pbWallet.GetBalanceRequest) (*pbWallet.GetBalanceResponse, error) {
			return nil, wantErr
		},
	}
	svc := newWalletService(fake)

	bal, err := svc.GetBalance(context.Background(), "user-1", "account-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if bal != nil {
		t.Fatalf("expected nil balance on error, got: %+v", bal)
	}
}

func TestWalletService_ListTransactions_Success(t *testing.T) {
	fake := &fixtures.FakeWalletServiceClient{
		ListTransactionsFunc: func(ctx context.Context, in *pbWallet.ListTransactionsRequest) (*pbWallet.ListTransactionsResponse, error) {
			if in.GetAccountId() != "account-1" || in.GetPage() != "2" || in.GetPageSize() != 10 {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.ListTransactionsResponse{
				Transactions: []*pbWallet.Transaction{
					{Id: "txn-1", FromAccount: "account-1", Amount: 100, Currency: "USD", Status: "completed"},
					{Id: "txn-2", ToAccount: "account-1", Amount: 200, Currency: "USD", Status: "completed"},
				},
				NextPageToken: "3",
			}, nil
		},
	}
	svc := newWalletService(fake)

	resp, err := svc.ListTransactions(context.Background(), "account-1", 2, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Transactions) != 2 || resp.NextPageToken != "3" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Transactions[0].ID != "txn-1" || resp.Transactions[1].ID != "txn-2" {
		t.Fatalf("unexpected transaction ordering/ids: %+v", resp.Transactions)
	}
}

func TestWalletService_ListTransactions_GrpcError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fixtures.FakeWalletServiceClient{
		ListTransactionsFunc: func(ctx context.Context, in *pbWallet.ListTransactionsRequest) (*pbWallet.ListTransactionsResponse, error) {
			return nil, wantErr
		},
	}
	svc := newWalletService(fake)

	resp, err := svc.ListTransactions(context.Background(), "account-1", 1, 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response on error, got: %+v", resp)
	}
}
