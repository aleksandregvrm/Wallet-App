package fixtures

import (
	"context"
	"fmt"
	"go-task-wallet-service/api-gateway/internal/domain"
)

// Domain fixtures declaring mocked services which implement the domain service structure exactly as in the domain of the api-gateway
// Since the implementation is based on the interface the thing that matters is the input output

type FakeUserService struct {
	RegisterUserFunc func(ctx context.Context, user *domain.User) (*domain.UserAuth, error)
	LoginUserFunc    func(ctx context.Context, user *domain.User) (*domain.UserAuth, error)
	RefreshTokenFunc func(ctx context.Context, refreshToken string) (string, error)
}

var _ domain.UserService = (*FakeUserService)(nil)

func (f *FakeUserService) RegisterUser(ctx context.Context, user *domain.User) (*domain.UserAuth, error) {
	if f.RegisterUserFunc == nil {
		return nil, fmt.Errorf("Fixtures: RegisterUserFunc not set")
	}
	return f.RegisterUserFunc(ctx, user)
}

func (f *FakeUserService) LoginUser(ctx context.Context, user *domain.User) (*domain.UserAuth, error) {
	if f.LoginUserFunc == nil {
		return nil, fmt.Errorf("Fixtures: LoginUserFunc not set")
	}
	return f.LoginUserFunc(ctx, user)
}

func (f *FakeUserService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	if f.RefreshTokenFunc == nil {
		return "", fmt.Errorf("fixtures: RefreshTokenFunc not set")
	}
	return f.RefreshTokenFunc(ctx, refreshToken)
}

type FakeAccountService struct {
	CreateAccountFunc func(ctx context.Context, account *domain.Account) error
	UpdateAccountFunc func(ctx context.Context, account *domain.Account) error
}

var _ domain.AccountService = (*FakeAccountService)(nil)

func (f *FakeAccountService) CreateAccount(ctx context.Context, account *domain.Account) error {
	if f.CreateAccountFunc == nil {
		return fmt.Errorf("fixtures: CreateAccountFunc not set")
	}
	return f.CreateAccountFunc(ctx, account)
}

func (f *FakeAccountService) UpdateAccount(ctx context.Context, account *domain.Account) error {
	if f.UpdateAccountFunc == nil {
		return fmt.Errorf("fixtures: UpdateAccountFunc not set")
	}
	return f.UpdateAccountFunc(ctx, account)
}

type FakeWalletService struct {
	TransferFundsFunc               func(ctx context.Context, userId, fromAccountID, toAccountID, currency string, amount int64) (*domain.Transaction, error)
	TransferFundsWithEmailNotifFunc func(ctx context.Context, fromEmail, toEmail, currency string, amount int64) error
	DepositFundsFunc                func(ctx context.Context, userId, accountID, currency string, amount int64) (*domain.Transaction, error)
	WithdrawFundsFunc               func(ctx context.Context, userId, accountID string, amount int64) (*domain.Transaction, error)
	ListTransactionsFunc            func(ctx context.Context, accountID string, page int32, pageSize int8) (*domain.ListTransactionsResponse, error)
	GetBalanceFunc                  func(ctx context.Context, userId, accountID string) (*domain.Balance, error)
}

var _ domain.WalletService = (*FakeWalletService)(nil)

func (f *FakeWalletService) TransferFunds(ctx context.Context, userId, fromAccountID, toAccountID, currency string, amount int64) (*domain.Transaction, error) {
	if f.TransferFundsFunc == nil {
		return nil, fmt.Errorf("fixtures: TransferFundsFunc not set")
	}
	return f.TransferFundsFunc(ctx, userId, fromAccountID, toAccountID, currency, amount)
}

func (f *FakeWalletService) TransferFundsWithEmailNotification(ctx context.Context, fromEmail, toEmail, currency string, amount int64) error {
	if f.TransferFundsWithEmailNotifFunc == nil {
		return fmt.Errorf("fixtures: TransferFundsWithEmailNotifFunc not set")
	}
	return f.TransferFundsWithEmailNotifFunc(ctx, fromEmail, toEmail, currency, amount)
}

func (f *FakeWalletService) DepositFunds(ctx context.Context, userId, accountID, currency string, amount int64) (*domain.Transaction, error) {
	if f.DepositFundsFunc == nil {
		return nil, fmt.Errorf("fixtures: DepositFundsFunc not set")
	}
	return f.DepositFundsFunc(ctx, userId, accountID, currency, amount)
}

func (f *FakeWalletService) WithdrawFunds(ctx context.Context, userId, accountID string, amount int64) (*domain.Transaction, error) {
	if f.WithdrawFundsFunc == nil {
		return nil, fmt.Errorf("fixtures: WithdrawFundsFunc not set")
	}
	return f.WithdrawFundsFunc(ctx, userId, accountID, amount)
}

func (f *FakeWalletService) ListTransactions(ctx context.Context, accountID string, page int32, pageSize int8) (*domain.ListTransactionsResponse, error) {
	if f.ListTransactionsFunc == nil {
		return nil, fmt.Errorf("fixtures: ListTransactionsFunc not set")
	}
	return f.ListTransactionsFunc(ctx, accountID, page, pageSize)
}

func (f *FakeWalletService) GetBalance(ctx context.Context, userId, accountID string) (*domain.Balance, error) {
	if f.GetBalanceFunc == nil {
		return nil, fmt.Errorf("fixtures: GetBalanceFunc not set")
	}
	return f.GetBalanceFunc(ctx, userId, accountID)
}
