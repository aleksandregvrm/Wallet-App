package integration_tests

import (
	"context"
	"net/http"
	"testing"

	"go-task-wallet-service/api-gateway/tests/fixtures"
	sharedDomain "go-task-wallet-service/shared/pkg/domain"
	"go-task-wallet-service/shared/pkg/session"
	pbAuth "go-task-wallet-service/shared/proto/impl/auth"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"
)

// Integration test which tests the whole user flow as part of a single integration/e2e test which include multiple services and moving parts

func TestFullFlow_Register_CreateAccount_Deposit(t *testing.T) {
	var balance int64

	authSrv := &fixtures.FakeAuthServiceServer{
		RegisterUserFunc: func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
			token, err := session.GenerateAccessToken(&sharedDomain.User{ID: "user-1", Username: in.GetUsername(), Email: in.GetEmail()})
			if err != nil {
				t.Fatalf("failed to sign token in fake RegisterUser: %v", err)
			}
			return &pbAuth.RegisterUserResponse{Id: "user-1", AccessToken: token}, nil
		},
	}
	walletSrv := &fixtures.FakeWalletServiceServer{
		CreateAccountFunc: func(ctx context.Context, in *pbWallet.CreateAccountRequest) (*pbWallet.CreateAccountResponse, error) {
			return &pbWallet.CreateAccountResponse{
				Account: &pbWallet.Account{Id: "account-1", Balance: 0, Currency: in.GetCurrency(), OwnerUser: in.GetOwnerUser()},
			}, nil
		},
		DepositFunc: func(ctx context.Context, in *pbWallet.DepositRequest) (*pbWallet.DepositResponse, error) {
			balance += in.GetAmount()
			return &pbWallet.DepositResponse{
				Transaction: &pbWallet.Transaction{Id: "txn-1", ToAccount: in.GetAccountId(), Amount: in.GetAmount(), Currency: in.GetCurrency(), Status: "completed"},
			}, nil
		},
		GetBalanceFunc: func(ctx context.Context, in *pbWallet.GetBalanceRequest) (*pbWallet.GetBalanceResponse, error) {
			return &pbWallet.GetBalanceResponse{Balance: balance, Currency: "USD"}, nil
		},
	}
	server := newTestServer(t, authSrv, walletSrv)

	registerResp := postJSON(t, server.URL+"/user/register", map[string]string{
		"name": "Jane Doe", "username": "janedoe", "email": "jane@example.com", "password": "hunter22",
	}, nil)
	if registerResp.StatusCode != http.StatusOK {
		t.Fatalf("register: expected 200, got: %d", registerResp.StatusCode)
	}
	var registerBody struct {
		AccessToken string `json:"acccess_token"`
	}
	decodeJSON(t, registerResp, &registerBody)
	authHeaders := map[string]string{"Authorization": "Bearer " + registerBody.AccessToken}

	createResp := postJSON(t, server.URL+"/account/create?userId=user-1", map[string]string{"currency": "USD"}, authHeaders)
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create account: expected 200, got: %d", createResp.StatusCode)
	}
	var createBody struct {
		AccountID string `json:"account_id"`
	}
	decodeJSON(t, createResp, &createBody)
	if createBody.AccountID != "account-1" {
		t.Fatalf("unexpected account id: %q", createBody.AccountID)
	}

	depositResp := postJSON(t, server.URL+"/wallet/deposit?userId=user-1",
		map[string]any{"account_id": createBody.AccountID, "currency": "USD", "amount": 1500}, authHeaders)
	if depositResp.StatusCode != http.StatusOK {
		t.Fatalf("deposit: expected 200, got: %d", depositResp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/wallet/balance?userId=user-1&accountId="+createBody.AccountID, nil)
	req.Header.Set("Authorization", authHeaders["Authorization"])
	balanceResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("balance request failed: %v", err)
	}
	if balanceResp.StatusCode != http.StatusOK {
		t.Fatalf("get balance: expected 200, got: %d", balanceResp.StatusCode)
	}
	var balanceBody struct {
		Balance int64 `json:"balance"`
	}
	decodeJSON(t, balanceResp, &balanceBody)
	if balanceBody.Balance != 1500 {
		t.Fatalf("expected balance 1500 after deposit, got: %d", balanceBody.Balance)
	}
}
