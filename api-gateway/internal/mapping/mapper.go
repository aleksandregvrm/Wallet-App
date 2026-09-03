package mapping

import (
	"go-task-wallet-service/api-gateway/internal/domain"
	pbAuth "go-task-wallet-service/shared/proto/impl/auth"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"
	"strconv"
)

// Mapper functions to map from Proto to Domain and from Domain to Proto
// Essential when using gRPC

func ToRegisterUserProto(user *domain.User) *pbAuth.RegisterUserRequest {
	if user == nil {
		return nil
	}

	return &pbAuth.RegisterUserRequest{
		Id:       user.ID,
		Name:     user.Name,
		Username: user.Username,
		Email:    user.Email,
		Password: user.Password,
	}
}

// Mapping to register User response carrying access token notifying user session has been activated
func ToRegisterUserAuthDomain(resp *pbAuth.RegisterUserResponse) *domain.UserAuth {
	if resp == nil {
		return nil
	}

	return &domain.UserAuth{
		ID:          resp.GetId(),
		AccessToken: resp.AccessToken,
	}
}

func ToLoginUserProto(user *domain.User) *pbAuth.LoginUserRequest {
	if user == nil {
		return nil
	}

	return &pbAuth.LoginUserRequest{
		Username: user.Username,
		Password: user.Password,
	}
}

func ToLoginUserDomain(resp *pbAuth.LoginUserResponse) *domain.UserAuth {
	if resp == nil {
		return nil
	}

	return &domain.UserAuth{
		ID:          resp.GetId(),
		AccessToken: resp.AccessToken,
	}
}

func ToCreateAccountProto(account *domain.Account) *pbWallet.CreateAccountRequest {
	if account == nil {
		return nil
	}

	return &pbWallet.CreateAccountRequest{
		OwnerUser: account.OwnerUser,
		Currency:  account.Currency,
	}
}

func ToAccountDomain(a *pbWallet.Account) *domain.Account {
	if a == nil {
		return nil
	}

	account := &domain.Account{
		ID:        a.GetId(),
		Balance:   a.GetBalance(),
		Currency:  a.GetCurrency(),
		OwnerUser: a.GetOwnerUser(),
	}
	if a.GetCreatedAt() != nil {
		account.CreatedAt = a.GetCreatedAt().AsTime()
	}

	return account
}

func ToTransactionDomain(t *pbWallet.Transaction) *domain.Transaction {
	if t == nil {
		return nil
	}

	transaction := &domain.Transaction{
		ID:          t.GetId(),
		FromAccount: t.GetFromAccount(),
		ToAccount:   t.GetToAccount(),
		Amount:      t.GetAmount(),
		Currency:    t.GetCurrency(),
		Status:      t.GetStatus(),
	}
	if t.GetCreatedAt() != nil {
		transaction.CreatedAt = t.GetCreatedAt().AsTime()
	}
	if t.GetUpdatedAt() != nil {
		transaction.UpdatedAt = t.GetUpdatedAt().AsTime()
	}

	return transaction
}

func ToDepositFundsProto(userId, accountID, currency string, amount int64) *pbWallet.DepositRequest {
	return &pbWallet.DepositRequest{
		UserId:    userId,
		AccountId: accountID,
		Currency:  currency,
		Amount:    amount,
	}
}

func ToWithdrawFundsProto(userId, accountID string, amount int64, idempotencyKey string) *pbWallet.WithdrawRequest {
	return &pbWallet.WithdrawRequest{
		AccountId:     accountID,
		Amount:        amount,
		TransactionId: idempotencyKey,
		UserId:        userId,
	}
}

func ToTransferFundsProto(userId, fromAccountID, toAccountID, currency string, amount int64, idempotencyKey string) *pbWallet.TransferRequest {
	return &pbWallet.TransferRequest{
		FromAccountId:  fromAccountID,
		ToAccountId:    toAccountID,
		Currency:       currency,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
		UserId:         userId,
	}
}

func ToGetBalanceProto(userId, accountID string) *pbWallet.GetBalanceRequest {
	return &pbWallet.GetBalanceRequest{
		AccountId: accountID,
		UserId:    userId,
	}
}

func ToListTransactionsProto(accountID string, page int32, pageSize int8) *pbWallet.ListTransactionsRequest {
	return &pbWallet.ListTransactionsRequest{
		AccountId: accountID,
		// Page is a string on the wire (see wallet.proto).
		Page:     strconv.Itoa(int(page)),
		PageSize: int32(pageSize),
	}
}

func ToListTransactionsDomain(resp *pbWallet.ListTransactionsResponse) *domain.ListTransactionsResponse {
	if resp == nil {
		return nil
	}

	transactions := make([]domain.Transaction, 0, len(resp.GetTransactions()))
	for _, t := range resp.GetTransactions() {
		if transaction := ToTransactionDomain(t); transaction != nil {
			transactions = append(transactions, *transaction)
		}
	}

	return &domain.ListTransactionsResponse{
		Transactions:  transactions,
		NextPageToken: resp.GetNextPageToken(),
	}
}
