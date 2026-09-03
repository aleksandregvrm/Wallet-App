package types

import "time"

// Shared types for Account and Transaction entities
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusRejected  TransactionStatus = "rejected"
)

type AccountModel struct {
	ID           string             `json:"id"`
	UserID       string             `json:"user_id"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	Transactions []TransactionModel `json:"transactions"`
	Balance      int64              `json:"balance"`
	Currency     string             `json:"currency"`
}

type TransactionModel struct {
	ID          string            `json:"id"`
	FromAccount string            `json:"from_account"`
	ToAccount   string            `json:"to_account"`
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	Status      TransactionStatus `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type UserModel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserAuthModel struct {
	ID                  string    `json:"id"`
	RefreshToken        string    `json:"refresh_token"`
	UserId              string    `json:"user_id"`
	IsCurrentlyLoggedIn bool      `json:"is_currently_logged_in"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Outbox statuses that define the proceeding action once outbox row has been picked up by outbox service
type OutboxStatus string

const (
	OutboxStatusPending OutboxStatus = "pending"
	OutboxStatusClaimed OutboxStatus = "claimed"
	// Final statuses
	OutboxStatusPublished OutboxStatus = "published"
	OutboxStatusFailed    OutboxStatus = "failed"
)

type OutboxModel struct {
	ID            string                 `json:"id"`
	AggregateType string                 `json:"aggregate_type"`
	AggregateId   string                 `json:"aggregate_id"`
	EventType     string                 `json:"event_type"`
	Payload       map[string]interface{} `json:"payload"`

	Topic        string `json:"topic"`
	PartitionKey string `json:"partition_key"`

	// Idempotency key combination of userId_accountId. Essential for idempotency lookups. making sure user cannot process another transaction
	// before first one is complete
	IdempotencyKey string `json:"idempotency_key"`

	Status OutboxStatus `json:"status"`
	// Error message is by default null
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}
