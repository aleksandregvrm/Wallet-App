package unit_tests

import (
	"testing"
	"time"

	"go-task-wallet-service/api-gateway/internal/domain"
	"go-task-wallet-service/api-gateway/internal/mapping"
	pbAuth "go-task-wallet-service/shared/proto/impl/auth"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Domain to proto as well as Proto to Domain mapping remains the core part of gRPC client which expects protobuf generates style structs
// To translate them to protobufs and send the via gRPC calls. But since we have our domain logic which strictly requires translations/mappings
// mapper functions remain basically an adapter/mapper can be ideally unit tested with happy pass and non-happy pass data

func TestToRegisterUserProto_Nil(t *testing.T) {
	if got := mapping.ToRegisterUserProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToRegisterUserProto(t *testing.T) {
	user := &domain.User{ID: "user-1", Name: "Jane Doe", Username: "janedoe", Email: "jane@example.com", Password: "hunter22"}
	got := mapping.ToRegisterUserProto(user)
	if got.GetId() != "user-1" || got.GetName() != "Jane Doe" || got.GetUsername() != "janedoe" || got.GetEmail() != "jane@example.com" || got.GetPassword() != "hunter22" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToRegisterUserAuthDomain_Nil(t *testing.T) {
	if got := mapping.ToRegisterUserAuthDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToRegisterUserAuthDomain(t *testing.T) {
	resp := &pbAuth.RegisterUserResponse{Id: "user-1", AccessToken: "token-abc"}
	got := mapping.ToRegisterUserAuthDomain(resp)
	if got.ID != "user-1" || got.AccessToken != "token-abc" {
		t.Fatalf("unexpected domain: %+v", got)
	}
}

func TestToLoginUserProto_Nil(t *testing.T) {
	if got := mapping.ToLoginUserProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToLoginUserProto(t *testing.T) {
	user := &domain.User{Username: "janedoe", Password: "hunter22", Email: "jane@example.com"}
	got := mapping.ToLoginUserProto(user)
	if got.GetUsername() != "janedoe" || got.GetPassword() != "hunter22" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToLoginUserDomain_Nil(t *testing.T) {
	if got := mapping.ToLoginUserDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToLoginUserDomain(t *testing.T) {
	resp := &pbAuth.LoginUserResponse{Id: "user-1", AccessToken: "token-abc"}
	got := mapping.ToLoginUserDomain(resp)
	if got.ID != "user-1" || got.AccessToken != "token-abc" {
		t.Fatalf("unexpected domain: %+v", got)
	}
}

func TestToCreateAccountProto_Nil(t *testing.T) {
	if got := mapping.ToCreateAccountProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToCreateAccountProto(t *testing.T) {
	account := &domain.Account{ID: "account-1", Balance: 999, Currency: "USD", OwnerUser: "user-1"}
	got := mapping.ToCreateAccountProto(account)
	if got.GetOwnerUser() != "user-1" || got.GetCurrency() != "USD" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToAccountDomain_Nil(t *testing.T) {
	if got := mapping.ToAccountDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToAccountDomain_WithCreatedAt(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := &pbWallet.Account{Id: "account-1", Balance: 1000, Currency: "USD", OwnerUser: "user-1", CreatedAt: timestamppb.New(createdAt)}
	got := mapping.ToAccountDomain(a)
	if got.ID != "account-1" || got.Balance != 1000 || got.Currency != "USD" || got.OwnerUser != "user-1" {
		t.Fatalf("unexpected domain: %+v", got)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected CreatedAt: %v", got.CreatedAt)
	}
}

func TestToAccountDomain_NilCreatedAt(t *testing.T) {
	a := &pbWallet.Account{Id: "account-1", Balance: 1000, Currency: "USD", OwnerUser: "user-1"}
	got := mapping.ToAccountDomain(a)
	if !got.CreatedAt.IsZero() {
		t.Fatalf("expected zero CreatedAt, got: %v", got.CreatedAt)
	}
}

func TestToTransactionDomain_Nil(t *testing.T) {
	if got := mapping.ToTransactionDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToTransactionDomain_WithTimestamps(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	tx := &pbWallet.Transaction{
		Id: "txn-1", FromAccount: "account-1", ToAccount: "account-2", Amount: 500, Currency: "USD", Status: "completed",
		CreatedAt: timestamppb.New(createdAt), UpdatedAt: timestamppb.New(updatedAt),
	}
	got := mapping.ToTransactionDomain(tx)
	if got.ID != "txn-1" || got.FromAccount != "account-1" || got.ToAccount != "account-2" || got.Amount != 500 || got.Currency != "USD" || got.Status != "completed" {
		t.Fatalf("unexpected domain: %+v", got)
	}
	if !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected timestamps: created=%v updated=%v", got.CreatedAt, got.UpdatedAt)
	}
}

func TestToTransactionDomain_NilTimestamps(t *testing.T) {
	tx := &pbWallet.Transaction{Id: "txn-1", Amount: 500, Currency: "USD", Status: "pending"}
	got := mapping.ToTransactionDomain(tx)
	if !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("expected zero timestamps, got: created=%v updated=%v", got.CreatedAt, got.UpdatedAt)
	}
}

func TestToDepositFundsProto(t *testing.T) {
	got := mapping.ToDepositFundsProto("user-1", "account-1", "USD", 500)
	if got.GetUserId() != "user-1" || got.GetAccountId() != "account-1" || got.GetCurrency() != "USD" || got.GetAmount() != 500 {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToWithdrawFundsProto(t *testing.T) {
	got := mapping.ToWithdrawFundsProto("user-1", "account-1", 200, "idem-key-1")
	if got.GetUserId() != "user-1" || got.GetAccountId() != "account-1" || got.GetAmount() != 200 || got.GetTransactionId() != "idem-key-1" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToTransferFundsProto(t *testing.T) {
	got := mapping.ToTransferFundsProto("user-1", "account-1", "account-2", "USD", 300, "idem-key-1")
	if got.GetUserId() != "user-1" || got.GetFromAccountId() != "account-1" || got.GetToAccountId() != "account-2" || got.GetCurrency() != "USD" || got.GetAmount() != 300 || got.GetIdempotencyKey() != "idem-key-1" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToGetBalanceProto(t *testing.T) {
	got := mapping.ToGetBalanceProto("user-1", "account-1")
	if got.GetUserId() != "user-1" || got.GetAccountId() != "account-1" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToListTransactionsProto(t *testing.T) {
	got := mapping.ToListTransactionsProto("account-1", 2, 10)
	if got.GetAccountId() != "account-1" || got.GetPage() != "2" || got.GetPageSize() != 10 {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToListTransactionsDomain_Nil(t *testing.T) {
	if got := mapping.ToListTransactionsDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToListTransactionsDomain(t *testing.T) {
	resp := &pbWallet.ListTransactionsResponse{
		Transactions: []*pbWallet.Transaction{
			{Id: "txn-1", Amount: 100, Currency: "USD", Status: "completed"},
			nil,
			{Id: "txn-2", Amount: 200, Currency: "USD", Status: "completed"},
		},
		NextPageToken: "3",
	}
	got := mapping.ToListTransactionsDomain(resp)
	if len(got.Transactions) != 2 {
		t.Fatalf("expected nil entries to be filtered out, got %d transactions: %+v", len(got.Transactions), got.Transactions)
	}
	if got.Transactions[0].ID != "txn-1" || got.Transactions[1].ID != "txn-2" {
		t.Fatalf("unexpected transaction ids: %+v", got.Transactions)
	}
	if got.NextPageToken != "3" {
		t.Fatalf("unexpected NextPageToken: %q", got.NextPageToken)
	}
}

func TestToListTransactionsDomain_EmptyTransactions(t *testing.T) {
	resp := &pbWallet.ListTransactionsResponse{NextPageToken: ""}
	got := mapping.ToListTransactionsDomain(resp)
	if len(got.Transactions) != 0 {
		t.Fatalf("expected empty slice, got: %+v", got.Transactions)
	}
}
