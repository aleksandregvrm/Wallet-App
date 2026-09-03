package events

import (
	"context"
	"errors"
)

// Messages with this error messages should never be nacked or retried. They should be sent to the DLQ right away
var ErrInvalidMessage = errors.New("invalid message")

// General Contract For Message Broker/Event Streamer
// Extendable to tools like Apache Kafka, RabbitMQ, AWS SQS, etc.
type ConnectMessaging interface {
	Connect() error
	AppendToDlq(ctx context.Context, topic, key string, message []byte, reason string) error
	PublishMessage(ctx context.Context, topic, key string, message []byte) error
	ConsumeMessage(ctx context.Context, topic string, handler func(ctx context.Context, message []byte) error) error
	CloseConnection() error
}

// Event topic
const (
	TransactionDepositRequested    = "transaction.deposit.requested"
	TransactionWithdrawalRequested = "transaction.withdrawal.requested"
	TransactionTransferRequested   = "transaction.transfer.requested"
	WalletEventsTopic              = "wallet.events.v1"
)
