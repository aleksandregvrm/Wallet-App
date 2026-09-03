package unit_tests

import (
	"testing"
	"time"

	types "go-task-wallet-service/shared/pkg/models"
	pb "go-task-wallet-service/shared/proto/impl/wallet"
	"go-task-wallet-service/wallet-service/internal/domain"
	"go-task-wallet-service/wallet-service/internal/mapping"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestToAccountDomain_Nil(t *testing.T) {
	if got := mapping.ToAccountDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToAccountDomain(t *testing.T) {
	createdAt := time.Unix(1000, 0).UTC()
	req := &pb.Account{Id: "account-1", Balance: 5000, Currency: "USD", OwnerUser: "user-1", CreatedAt: timestamppb.New(createdAt)}
	got := mapping.ToAccountDomain(req)
	if got.ID != "account-1" || got.UserID != "user-1" || got.Balance != 5000 || got.Currency != "USD" {
		t.Fatalf("unexpected domain: %+v", got)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected CreatedAt: %v", got.CreatedAt)
	}
}

func TestToAccountProto_Nil(t *testing.T) {
	if got := mapping.ToAccountProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToAccountProto(t *testing.T) {
	account := &domain.Account{ID: "account-1", UserID: "user-1", Balance: 5000, Currency: "USD"}
	got := mapping.ToAccountProto(account)
	if got.GetId() != "account-1" || got.GetOwnerUser() != "user-1" || got.GetBalance() != 5000 || got.GetCurrency() != "USD" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToTransactionDomain_Nil(t *testing.T) {
	if got := mapping.ToTransactionDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToTransactionDomain(t *testing.T) {
	createdAt := time.Unix(1000, 0).UTC()
	updatedAt := time.Unix(2000, 0).UTC()
	req := &pb.Transaction{
		Id: "transaction-1", FromAccount: "account-1", ToAccount: "account-2",
		Amount: 750, Currency: "USD", Status: "completed",
		CreatedAt: timestamppb.New(createdAt), UpdatedAt: timestamppb.New(updatedAt),
	}
	got := mapping.ToTransactionDomain(req)
	if got.ID != "transaction-1" || got.FromAccount != "account-1" || got.ToAccount != "account-2" || got.Amount != 750 || got.Currency != "USD" {
		t.Fatalf("unexpected domain: %+v", got)
	}
	if got.Status != types.TransactionStatusCompleted {
		t.Fatalf("unexpected status: %v", got.Status)
	}
	if !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected timestamps: created=%v updated=%v", got.CreatedAt, got.UpdatedAt)
	}
}

func TestToTransactionProto_Nil(t *testing.T) {
	if got := mapping.ToTransactionProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToTransactionProto(t *testing.T) {
	transaction := &domain.Transaction{
		ID: "transaction-1", FromAccount: "account-1", ToAccount: "account-2",
		Amount: 750, Currency: "USD", Status: types.TransactionStatusPending,
	}
	got := mapping.ToTransactionProto(transaction)
	if got.GetId() != "transaction-1" || got.GetFromAccount() != "account-1" || got.GetToAccount() != "account-2" {
		t.Fatalf("unexpected proto: %+v", got)
	}
	if got.GetAmount() != 750 || got.GetCurrency() != "USD" || got.GetStatus() != "pending" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToListTransactionsDomain_Nil(t *testing.T) {
	if got := mapping.ToListTransactionsDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToListTransactionsDomain(t *testing.T) {
	req := &pb.ListTransactionsRequest{AccountId: "account-1", Page: "2", PageSize: 16}
	got := mapping.ToListTransactionsDomain(req)
	if got.AccountID != "account-1" || got.Page != 2 || got.PageSize != 16 {
		t.Fatalf("unexpected domain: %+v", got)
	}
}

func TestToListTransactionsDomain_MalformedPageFallsBackToZero(t *testing.T) {
	req := &pb.ListTransactionsRequest{AccountId: "account-1", Page: "not-a-number", PageSize: 16}
	got := mapping.ToListTransactionsDomain(req)
	if got.Page != 0 {
		t.Fatalf("expected page to fall back to 0 for a malformed value, got: %d", got.Page)
	}
}

func TestToListTransactionsProto_Nil(t *testing.T) {
	if got := mapping.ToListTransactionsProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToListTransactionsProto(t *testing.T) {
	resp := &domain.ListTransactionsResponse{
		Transactions: []domain.Transaction{
			{ID: "transaction-1", FromAccount: "account-1", ToAccount: "account-2", Amount: 100, Status: types.TransactionStatusCompleted},
			{ID: "transaction-2", FromAccount: "account-1", ToAccount: "account-2", Amount: 200, Status: types.TransactionStatusPending},
		},
		NextPageToken: "2",
	}
	got := mapping.ToListTransactionsProto(resp)
	if len(got.GetTransactions()) != 2 {
		t.Fatalf("expected 2 transactions, got: %d", len(got.GetTransactions()))
	}
	if got.GetTransactions()[0].GetId() != "transaction-1" || got.GetTransactions()[1].GetId() != "transaction-2" {
		t.Fatalf("unexpected transaction ordering: %+v", got.GetTransactions())
	}
	if got.GetNextPageToken() != "2" {
		t.Fatalf("unexpected next page token: %q", got.GetNextPageToken())
	}
}

func TestToListTransactionsProto_EmptyList(t *testing.T) {
	resp := &domain.ListTransactionsResponse{Transactions: nil, NextPageToken: ""}
	got := mapping.ToListTransactionsProto(resp)
	if len(got.GetTransactions()) != 0 {
		t.Fatalf("expected an empty slice, got: %+v", got.GetTransactions())
	}
}
