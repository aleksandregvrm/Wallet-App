package http

import "go-task-wallet-service/api-gateway/internal/domain"

// Module to describe Http layer transport data
// Including custom validations per route, request and response data structures, and any other HTTP transport specific logic.

// User
type CreateUserRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginUserRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Registering/ Logging In Response
type AuthenticateUserResponse struct {
	Status      string `json:"status"`
	AccessToken string `json:"acccess_token"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	Token string `json:"token"`
}

// Account
type UpdateAccountRequest struct {
	AccountID      string `json:"account_id"`
	OwnerAccountId string `json:"owner_account_id"`
}

type CreateAccountResponse struct {
	Status    string `json:"status"`
	AccountID string `json:"account_id"`
	Balance   int64  `json:"balance"`
	Currency  string `json:"currency"`
}

type UpdateAccountResponse struct {
	Status string `json:"status"`
}

// Transaction
type TransferFundsRequest struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Currency      string `json:"currency"`
	Amount        int64  `json:"amount"`
}

type TransferFundsWithEmailRequest struct {
	FromEmail string `json:"from_email"`
	ToEmail   string `json:"to_email"`
	Currency  string `json:"currency"`
	Amount    int64  `json:"amount"`
}

type TransferFundsResponse struct {
	Transaction domain.Transaction `json:"transaction"`
}

type DepositFundsRequest struct {
	AccountID string `json:"account_id"`
	Currency  string `json:"currency"`
	Amount    int64  `json:"amount"`
}

type DepositFundsResponse struct {
	Transaction domain.Transaction `json:"transaction"`
}

type WithdrawFundsRequest struct {
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount"`
}

type WithdrawFundsResponse struct {
	Transaction domain.Transaction `json:"transaction"`
}

type ListTransactionsRequest struct {
	AccountID string `json:"account_id"`
}

type GetBalanceResponse struct {
	AccountID string `json:"account_id"`
	Balance   int64  `json:"balance"`
	Currency  string `json:"currency"`
}

type ListTransactionsResponse struct {
	Transactions []domain.Transaction `json:"transactions"`
	NextPage     string               `json:"next_page"`
}
