package integration_tests

import (
	"context"
	"net/http"
	"testing"

	"go-task-wallet-service/api-gateway/tests/fixtures"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"
)

func authHeader(t *testing.T, userId, username, email string) map[string]string {
	token := fixtures.ValidAccessToken(t, userId, username, email)
	return map[string]string{"Authorization": "Bearer " + token}
}

func TestWalletFlow_Deposit_Success(t *testing.T) {
	walletSrv := &fixtures.FakeWalletServiceServer{
		DepositFunc: func(ctx context.Context, in *pbWallet.DepositRequest) (*pbWallet.DepositResponse, error) {
			if in.GetUserId() != "user-1" || in.GetAccountId() != "account-1" || in.GetAmount() != 500 {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.DepositResponse{
				Transaction: &pbWallet.Transaction{Id: "txn-1", ToAccount: "account-1", Amount: 500, Currency: "USD", Status: "pending"},
			}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	resp := postJSON(t, server.URL+"/wallet/deposit?userId=user-1",
		map[string]any{"account_id": "account-1", "currency": "USD", "amount": 500},
		authHeader(t, "user-1", "janedoe", "jane@example.com"))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
	var body struct {
		Transaction struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"transaction"`
	}
	decodeJSON(t, resp, &body)
	if body.Transaction.ID != "txn-1" || body.Transaction.Status != "pending" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestWalletFlow_Withdraw_Success(t *testing.T) {
	walletSrv := &fixtures.FakeWalletServiceServer{
		WithdrawFunc: func(ctx context.Context, in *pbWallet.WithdrawRequest) (*pbWallet.WithdrawResponse, error) {
			if in.GetUserId() != "user-1" || in.GetAccountId() != "account-1" || in.GetAmount() != 200 {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.WithdrawResponse{
				Transaction: &pbWallet.Transaction{Id: "txn-2", FromAccount: "account-1", Amount: 200, Currency: "USD", Status: "pending"},
			}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	resp := postJSON(t, server.URL+"/wallet/withdraw?userId=user-1",
		map[string]any{"account_id": "account-1", "amount": 200},
		authHeader(t, "user-1", "janedoe", "jane@example.com"))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
	var body struct {
		Transaction struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"transaction"`
	}
	decodeJSON(t, resp, &body)
	if body.Transaction.ID != "txn-2" || body.Transaction.Status != "pending" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestWalletFlow_Transfer_Success(t *testing.T) {
	walletSrv := &fixtures.FakeWalletServiceServer{
		TransferFunc: func(ctx context.Context, in *pbWallet.TransferRequest) (*pbWallet.TransferResponse, error) {
			if in.GetUserId() != "user-1" || in.GetFromAccountId() != "account-1" || in.GetToAccountId() != "account-2" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.TransferResponse{
				Transaction: &pbWallet.Transaction{Id: "txn-3", FromAccount: "account-1", ToAccount: "account-2", Amount: 300, Currency: "USD", Status: "pending"},
			}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	resp := postJSON(t, server.URL+"/wallet/transfer?userId=user-1",
		map[string]any{"from_account_id": "account-1", "to_account_id": "account-2", "currency": "USD", "amount": 300},
		authHeader(t, "user-1", "janedoe", "jane@example.com"))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
	var body struct {
		Transaction struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"transaction"`
	}
	decodeJSON(t, resp, &body)
	if body.Transaction.ID != "txn-3" || body.Transaction.Status != "pending" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestWalletFlow_GetBalance_Success(t *testing.T) {
	walletSrv := &fixtures.FakeWalletServiceServer{
		GetBalanceFunc: func(ctx context.Context, in *pbWallet.GetBalanceRequest) (*pbWallet.GetBalanceResponse, error) {
			if in.GetUserId() != "user-1" || in.GetAccountId() != "account-1" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbWallet.GetBalanceResponse{Balance: 4200, Currency: "USD"}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/wallet/balance?userId=user-1&accountId=account-1", nil)
	for k, v := range authHeader(t, "user-1", "janedoe", "jane@example.com") {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
	var body struct {
		AccountID string `json:"account_id"`
		Balance   int64  `json:"balance"`
		Currency  string `json:"currency"`
	}
	decodeJSON(t, resp, &body)
	if body.AccountID != "account-1" || body.Balance != 4200 || body.Currency != "USD" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestWalletFlow_GetBalance_MissingAccountId(t *testing.T) {
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, &fixtures.FakeWalletServiceServer{})

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/wallet/balance?userId=user-1", nil)
	for k, v := range authHeader(t, "user-1", "janedoe", "jane@example.com") {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got: %d", resp.StatusCode)
	}
}

func TestWalletFlow_Deposit_Unauthenticated(t *testing.T) {
	called := false
	walletSrv := &fixtures.FakeWalletServiceServer{
		DepositFunc: func(ctx context.Context, in *pbWallet.DepositRequest) (*pbWallet.DepositResponse, error) {
			called = true
			return &pbWallet.DepositResponse{}, nil
		},
	}
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, walletSrv)

	resp := postJSON(t, server.URL+"/wallet/deposit?userId=user-1",
		map[string]any{"account_id": "account-1", "currency": "USD", "amount": 500}, nil)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", resp.StatusCode)
	}
	if called {
		t.Fatalf("expected the backend never to be reached without a token")
	}
}
