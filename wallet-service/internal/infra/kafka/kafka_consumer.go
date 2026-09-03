package kafka

import (
	"context"

	"go-task-wallet-service/shared/events"
	"go-task-wallet-service/shared/logging"
)

// Consumer listens for messages on a topic and dispatches them to a
// handler. Depends on events.ConnectMessaging (interface), not a concrete
// broker client, so the messaging backend is swappable.
type Consumer struct {
	messaging events.ConnectMessaging
	topic     string
	logging.InternalLogger
}

func NewConsumer(messaging events.ConnectMessaging, topic string) *Consumer {
	return &Consumer{
		messaging:      messaging,
		topic:          topic,
		InternalLogger: *logging.NewInternalLogger(),
	}
}

// Consume begins consuming messages on the configured topic. The underlying
// messaging client runs its own goroutine internally (see
// KafkaMessaging.ConsumeMessage) and stops when ctx is cancelled.
func (c *Consumer) Consume(ctx context.Context, handler func(ctx context.Context, message []byte) error) error {
	return c.messaging.ConsumeMessage(ctx, c.topic, handler)
}
