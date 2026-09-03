package domain

import (
	"context"
	"encoding/json"
	types "go-task-wallet-service/shared/pkg/models"
	"time"
)

// Domain Transaction
type Transaction struct {
	ID          string
	FromAccount string
	ToAccount   string
	Amount      int64
	Currency    string
	Status      types.TransactionStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Domain request/response pair for ListTransactions, mirroring the
// ListTransactionsRequest/ListTransactionsResponse proto messages.
type ListTransactionsRequest struct {
	AccountID string
	Page      int32
	PageSize  int8
}

type ListTransactionsResponse struct {
	Transactions  []Transaction
	NextPageToken string
}

// Deposit Operation
type DepositResponse struct {
	Transaction Transaction
}

// The actual structure received by the wallet service
type DepositRequestedPayload struct {
	ID          string "json:`id`"
	FromAccount string "json:`from_account`"
	ToAccount   string "json:`to_account`"
	Amount      int64  "json:`amount`"
	Currency    string "json:`currency`"
	UserId      string "json:`user_id`"
}

// Withdrawal Operation
type WithdrawResponse struct {
	Transaction Transaction
}

// The actual structure received by the wallet service
type WithdrawalRequestedPayload struct {
	ID          string "json:`id`"
	FromAccount string "json:`from_account`"
	ToAccount   string "json:`to_account`"
	Amount      int64  "json:`amount`"
	UserId      string "json:`user_id`"
}

// Transfer Operation
type TransferResponse struct {
	Transaction Transaction
}

// The actual structure received by the wallet service
type TransferRequestedPayload struct {
	ID          string "json:`id`"
	FromAccount string "json:`from_account`"
	ToAccount   string "json:`to_account`"
	Amount      int64  "json:`amount`"
	Currency    string "json:`currency`"
	UserId      string "json:`user_id`"
}

// Domain Account
type Account struct {
	ID        string
	UserID    string
	Balance   int64
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OutboxTransaction is the typed payload for OutboxRelayService.InsertOutboxRelay
// when the aggregate being relayed is a transaction (transfer/deposit/withdraw).
type OutboxTransaction struct {
	TransactionId string
	FromAccount   string
	ToAccount     string
	Amount        int64
	Currency      string
	Status        types.TransactionStatus
	EventType     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OutboxInsertConfig struct {
	AggregateType string // Iterable domain event
	AggregateId   string // Iterable domain event id
	EventType     string // Example - transaction.created
	Payload       map[string]interface{}

	// Owning user, combined with Partition_key (the account id) to form the
	// per-account idempotency lock key — see WalletRepository.OutboxInsert.
	UserId string

	// Message broker/Event streamer configs
	Topic         string
	Partition_key string // Kafka specific field
}

type WalletRepository interface {
	// account
	InsertAccount(ctx context.Context, userId, currency string) (*types.AccountModel, error)
	FindAccountById(ctx context.Context, accountId string) (*types.AccountModel, error)
	FindAccountsByUserId(ctx context.Context, userId string) ([]types.AccountModel, error)
	UpdateAccount(ctx context.Context, fromAccountId, toAccountId string) (*types.AccountModel, error)

	// purely cache related operation of updating the cache with the new balance value until balance update happens and the cache is invalidated.
	// In case the cache is invalidated we cache it again with one persistence query
	GetOrStoreAccountBalance(ctx context.Context, accountId string, amount int64) (int64, error)

	// Outbox
	// Insert operation only permitted once idempotency check is passed
	OutboxInsert(ctx context.Context, outboxInsertConfig *OutboxInsertConfig) error
	OutboxUpdate(ctx context.Context, aggregateId, status string) error

	// Records a failed publish attempt with exponential backoff — requeues to
	// pending with a growing backoff. This ensures possible republish of messages if during the publishers work
	// something goes wrong and recovery is needed.
	OutboxMarkPublishFailure(ctx context.Context, aggregateId, errMsg string) (string, error)
	OutboxDelete(ctx context.Context, aggregateId string) error
	OutboxGet(ctx context.Context, aggregateId string) (*types.OutboxModel, error) // Retrieving single outbox row

	// Retrieving a fixed batch of pending outbox jobs. Locks the selected rows
	// (FOR UPDATE SKIP LOCKED) for the duration of the read only — the
	// transaction is committed internally before this returns, so concurrent
	// callers never see the same row twice, but the caller never touches a tx.
	OutboxPendingGetBatch(ctx context.Context, limit int) ([]*types.OutboxModel, error)

	// Transactions
	TransferFunds(ctx context.Context, transactionId, fromAccountId, toAccountId, currency string, amount int64) (*types.TransactionModel, error)
	CreatePendingTransaction(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error
	DepositFunds(ctx context.Context, transactionId, accountId, currency string, amount int64) (*types.TransactionModel, error)
	WithdrawFunds(ctx context.Context, transactionId, accountId string, amount int64) (*types.TransactionModel, error)
	ListTransactions(ctx context.Context, accountId string, offset, limit int) ([]types.TransactionModel, error)
	// userId/accountId are used to release the per-account idempotency key
	// (see OutboxInsert) once the transaction finishes processing.
	UpdateTransactionStatus(ctx context.Context, transactionId, userId, accountId string, status types.TransactionStatus) error

	// Idempotency release in case there's a retry to be executed on the outbox. This prevents a stalemate state in case something internally fails or has a downtime
	IdempotencyRelease(ctx context.Context, transactionId, userId, accountId string) error
}

type WalletService interface {
	// Dispatched domain event handlers
	HandleDepositDispatched(ctx context.Context, aggregateId string, payload json.RawMessage) error
	HandleWithdrawalDispatched(ctx context.Context, aggregateId string, payload json.RawMessage) error
	HandleTransferDispatched(ctx context.Context, aggregateId string, payload json.RawMessage) error

	// Actual business logic executed
	OpenAccount(ctx context.Context, userId, currency string) (*Account, error)
	UpdateAccount(fromAccountId, toAccountId string) (*Account, error)
	GetBalance(ctx context.Context, userId, accountId string) (*Account, error)
	ListTransactions(ctx context.Context, listTransactionRequest ListTransactionsRequest) (*ListTransactionsResponse, error)
}

// Outbox relay service is the communication point between the outbox database and subsequent message enqueueing
// Should be run in it's own separate goroutine
type OutboxRelayService interface {
	InsertOutboxRelay(ctx context.Context, aggregateType, aggregateId, eventType, partitionKey, userId string, payload map[string]interface{}) error
	CheckAndRelay(ctx context.Context) // Polls on an interval, ensuring persistence to the outbox table is drained
}

// Service with a dedicated responsibility to match the incoming event with an appropriate handler, Which happens in WalletService
// It remains as a crucial service in event processing. Since any event that is not handled will be lost, and the system will be in an inconsistent state. This service ensures that all events are processed and handled correctly.
type EventDispatcherService interface {
	Handle(ctx context.Context, message []byte) error
}

// EventPublisher decouples the service layer from a concrete messaging
// it depends on this contract, not on concrete implementation of publisher
type EventPublisher interface {
	Publish(ctx context.Context, key string, value []byte) error
}

type EventConsumer interface {
	Consume(ctx context.Context, handler func(ctx context.Context, message []byte) error) error
}

// Ensured the versioning. Should be bumped once the structure changes
// This should differentiate it if we have old messages left behind with previous version
const EnvelopeVersion = 1

// Envelope structure of the domain event triggered after the outbox operation
type Envelope struct {
	EventId     string          `json:"event_id"`
	EventType   string          `json:"event_type"`
	AggregateId string          `json:"aggregate_id"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Version     int             `json:"version"`
	Payload     json.RawMessage `json:"payload"`
}
