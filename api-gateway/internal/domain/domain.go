package domain

import (
	"context"
	"time"
)

type UserService interface {
	RegisterUser(ctx context.Context, user *User) (*UserAuth, error)
	LoginUser(ctx context.Context, user *User) (*UserAuth, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, error) // Route where client can refresh the customers expired refresh token
}

type AccountService interface {
	CreateAccount(ctx context.Context, account *Account) error
	UpdateAccount(ctx context.Context, account *Account) error
}

type WalletService interface {
	// Asynchronous call, response arrives immediately
	TransferFunds(ctx context.Context, userId, fromAccountID, toAccountID, currency string, amount int64) (*Transaction, error)
	TransferFundsWithEmailNotification(ctx context.Context, fromEmail, toEmail, currency string, amount int64) error
	DepositFunds(ctx context.Context, userId, accountID, currency string, amount int64) (*Transaction, error)
	WithdrawFunds(ctx context.Context, userId, accountID string, amount int64) (*Transaction, error)
	// Synchronous call
	ListTransactions(ctx context.Context, accountID string, page int32, pageSize int8) (*ListTransactionsResponse, error)
	GetBalance(ctx context.Context, userId, accountID string) (*Balance, error)
}

type Account struct {
	ID           string        `json:"id"`
	Balance      int64         `json:"balance"`
	Currency     string        `json:"currency"`
	OwnerUser    string        `json:"owner_user"`
	Transactions []Transaction `json:"transactions"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// Domain User
type User struct {
	ID        string
	Name      string
	Username  string
	Email     string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Domain Transaction
type Transaction struct {
	ID          string    `json:"id"`
	FromAccount string    `json:"from_account"`
	ToAccount   string    `json:"to_account"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListTransactionsResponse struct {
	Transactions  []Transaction
	NextPageToken string
}

type Balance struct {
	AccountID string
	Balance   int64
	Currency  string
}

// Distinct domain UserAuth which consists of the access token handed to the client.
// The refresh token stays server-side (persisted in auth-service's DB/cache); this
// module never sees or forwards it.
type UserAuth struct {
	ID                  string
	AccessToken         string
	UserId              string
	IsCurrentlyLoggedIn bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
