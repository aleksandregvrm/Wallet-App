package unit_tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-task-wallet-service/api-gateway/internal/domain"
	infra "go-task-wallet-service/api-gateway/internal/infra/grpc"
	"go-task-wallet-service/api-gateway/internal/services"
	"go-task-wallet-service/api-gateway/tests/fixtures"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func newAccountService(fake *fixtures.FakeWalletServiceClient) *services.AccountService {
	return services.NewAccountService(&infra.GrpcHandler{WalletClient: fake})
}

// Account related functionality with their corresponding results
func TestAccountService_CreateAccount_Success(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fake := &fixtures.FakeWalletServiceClient{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			if in.GetOwnerUser() != "user-1" || in.GetCurrency() != "USD" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.CreateAccountResponse{
				Account: &pbWallet.Account{
					Id: "account-1", Balance: 0, Currency: "USD", OwnerUser: "user-1",
					CreatedAt: timestamppb.New(createdAt),
				},
			}, nil
		},
	}
	svc := newAccountService(fake)

	account := &domain.Account{OwnerUser: "user-1", Currency: "USD"}
	err := svc.CreateAccount(context.Background(), account)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.ID != "account-1" || account.Balance != 0 || account.Currency != "USD" || account.OwnerUser != "user-1" {
		t.Fatalf("unexpected account after create: %+v", account)
	}
	if !account.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected CreatedAt: %v", account.CreatedAt)
	}
}

func TestAccountService_CreateAccount_GrpcError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fixtures.FakeWalletServiceClient{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			return nil, wantErr
		},
	}
	accountService := newAccountService(fake)

	account := &domain.Account{OwnerUser: "user-1", Currency: "USD"}
	err := accountService.CreateAccount(context.Background(), account)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if account.ID != "" || account.Balance != 0 {
		t.Fatalf("expected account to be untouched on error, got: %+v", account)
	}
}

func TestAccountService_CreateAccount_NilAccountInResponse(t *testing.T) {
	fake := &fixtures.FakeWalletServiceClient{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			return &pbWallet.CreateAccountResponse{Account: nil}, nil
		},
	}
	accountService := newAccountService(fake)

	account := &domain.Account{OwnerUser: "user-1", Currency: "USD"}
	err := accountService.CreateAccount(context.Background(), account)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.OwnerUser != "user-1" || account.Currency != "USD" {
		t.Fatalf("expected account to be untouched when response has no account, got: %+v", account)
	}
}

// As the updateAccount functionality is not yet implemented
func TestAccountService_UpdateAccount_NotImplemented(t *testing.T) {
	accountService := newAccountService(&fixtures.FakeWalletServiceClient{})

	account := &domain.Account{ID: "account-1", OwnerUser: "user-1"}
	err := accountService.UpdateAccount(context.Background(), account)
	if err != nil {
		t.Fatalf("expected nil error from the stub, got: %v", err)
	}
	if account.ID != "account-1" || account.OwnerUser != "user-1" {
		t.Fatalf("expected account to be untouched by the stub, got: %+v", account)
	}
}
