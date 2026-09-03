package integration_tests

import (
	"context"
	"net/http"
	"testing"

	"go-task-wallet-service/api-gateway/tests/fixtures"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAccountFlow_Create_Success(t *testing.T) {
	walletSrv := &fixtures.FakeWalletServiceServer{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			if in.GetOwnerUser() != "user-1" || in.GetCurrency() != "USD" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.CreateAccountResponse{
				Account: &pbWallet.Account{
					Id: "account-1", Balance: 0, Currency: "USD", OwnerUser: "user-1",
					CreatedAt: timestamppb.Now(),
				},
			}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	token := fixtures.ValidAccessToken(t, "user-1", "janedoe", "jane@example.com")
	resp := postJSON(t, server.URL+"/account/create?userId=user-1", map[string]string{"currency": "USD"},
		map[string]string{"Authorization": "Bearer " + token})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
	var body struct {
		Status    string `json:"status"`
		AccountID string `json:"account_id"`
		Currency  string `json:"currency"`
	}
	decodeJSON(t, resp, &body)
	if body.Status != "account created" || body.AccountID != "account-1" || body.Currency != "USD" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestAccountFlow_Create_Unauthenticated(t *testing.T) {
	called := false
	walletSrv := &fixtures.FakeWalletServiceServer{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			called = true
			return &pbWallet.CreateAccountResponse{}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	resp := postJSON(t, server.URL+"/account/create?userId=user-1", map[string]string{"currency": "USD"}, nil)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", resp.StatusCode)
	}
	if called {
		t.Fatalf("expected the backend never to be reached without a token")
	}
}

func TestAccountFlow_Create_TokenUserIdMismatch(t *testing.T) {
	called := false
	walletSrv := &fixtures.FakeWalletServiceServer{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			called = true
			return &pbWallet.CreateAccountResponse{}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	token := fixtures.ValidAccessToken(t, "user-1", "janedoe", "jane@example.com")
	resp := postJSON(t, server.URL+"/account/create?userId=user-2", map[string]string{"currency": "USD"},
		map[string]string{"Authorization": "Bearer " + token})

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", resp.StatusCode)
	}
	if called {
		t.Fatalf("expected the backend never to be reached on a userId/token mismatch")
	}
}

func TestAccountFlow_Create_MissingCurrency(t *testing.T) {
	called := false
	walletSrv := &fixtures.FakeWalletServiceServer{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			called = true
			return &pbWallet.CreateAccountResponse{}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	token := fixtures.ValidAccessToken(t, "user-1", "janedoe", "jane@example.com")
	resp := postJSON(t, server.URL+"/account/create?userId=user-1", map[string]string{"currency": ""},
		map[string]string{"Authorization": "Bearer " + token})

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got: %d", resp.StatusCode)
	}
	if called {
		t.Fatalf("expected the backend never to be reached when validation fails")
	}
}

func TestAccountFlow_Create_BackendError(t *testing.T) {
	walletSrv := &fixtures.FakeWalletServiceServer{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			return nil, status.Error(codes.Internal, "database unavailable")
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	token := fixtures.ValidAccessToken(t, "user-1", "janedoe", "jane@example.com")
	resp := postJSON(t, server.URL+"/account/create?userId=user-1", map[string]string{"currency": "USD"},
		map[string]string{"Authorization": "Bearer " + token})

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got: %d", resp.StatusCode)
	}
}
