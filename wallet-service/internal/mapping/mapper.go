package mapping

import (
	models "go-task-wallet-service/shared/pkg/models"
	pb "go-task-wallet-service/shared/proto/impl/wallet"
	"go-task-wallet-service/wallet-service/internal/domain"
	"strconv"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Mapper functions to map from Proto to Domain and from Domain to Proto
// Essential when using gRPC

func ToAccountDomain(a *pb.Account) *domain.Account {
	if a == nil {
		return nil
	}

	account := &domain.Account{
		ID:       a.GetId(),
		UserID:   a.GetOwnerUser(),
		Balance:  a.GetBalance(),
		Currency: a.GetCurrency(),
	}
	if a.GetCreatedAt() != nil {
		account.CreatedAt = a.GetCreatedAt().AsTime()
	}

	return account
}

func ToAccountProto(account *domain.Account) *pb.Account {
	if account == nil {
		return nil
	}

	return &pb.Account{
		Id:        account.ID,
		Balance:   account.Balance,
		Currency:  account.Currency,
		OwnerUser: account.UserID,
		CreatedAt: timestamppb.New(account.CreatedAt),
	}
}

func ToTransactionDomain(t *pb.Transaction) *domain.Transaction {
	if t == nil {
		return nil
	}

	transaction := &domain.Transaction{
		ID:          t.GetId(),
		FromAccount: t.GetFromAccount(),
		ToAccount:   t.GetToAccount(),
		Amount:      t.GetAmount(),
		Currency:    t.GetCurrency(),
		Status:      models.TransactionStatus(t.GetStatus()),
	}
	// Formatting to Go time library convetion
	if t.GetCreatedAt() != nil {
		transaction.CreatedAt = t.GetCreatedAt().AsTime()
	}
	if t.GetUpdatedAt() != nil {
		transaction.UpdatedAt = t.GetUpdatedAt().AsTime()
	}

	return transaction
}

func ToTransactionProto(t *domain.Transaction) *pb.Transaction {
	if t == nil {
		return nil
	}

	return &pb.Transaction{
		Id:          t.ID,
		FromAccount: t.FromAccount,
		ToAccount:   t.ToAccount,
		Amount:      t.Amount,
		Currency:    t.Currency,
		Status:      string(t.Status),
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}
}

func ToListTransactionsDomain(req *pb.ListTransactionsRequest) *domain.ListTransactionsRequest {
	if req == nil {
		return nil
	}

	// Page arrives as a string on the wire (see wallet.proto); malformed
	// values fall back to 0 rather than failing the mapping.
	page, _ := strconv.Atoi(req.GetPage())

	return &domain.ListTransactionsRequest{
		AccountID: req.GetAccountId(),
		Page:      int32(page),
		PageSize:  int8(req.GetPageSize()), // Since page size is never going to be more than 128
	}
}

func ToListTransactionsProto(resp *domain.ListTransactionsResponse) *pb.ListTransactionsResponse {
	if resp == nil {
		return nil
	}

	transactions := make([]*pb.Transaction, 0, len(resp.Transactions))
	for i := range resp.Transactions {
		transactions = append(transactions, ToTransactionProto(&resp.Transactions[i]))
	}

	return &pb.ListTransactionsResponse{
		Transactions:  transactions,
		NextPageToken: resp.NextPageToken,
	}
}
