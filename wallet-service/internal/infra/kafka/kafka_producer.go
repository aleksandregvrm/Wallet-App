package kafka

import (
	"context"
	"time"

	"go-task-wallet-service/shared/events"
	"go-task-wallet-service/shared/logging"
	"go-task-wallet-service/wallet-service/internal/domain"
)

const publishTimeout = 5 * time.Second

var _ domain.EventPublisher = (*Producer)(nil)

// Producer publishes messages to Kafka synchronously. Depends on
// events.ConnectMessaging (interface), not a concrete broker client, so the
// messaging backend is swappable.
//
// The sole responsibility of the publisher is to deliver the message to the kafka broker and get the
type Producer struct {
	messaging events.ConnectMessaging
	topic     string
	logging.InternalLogger
}

func NewProducer(messaging events.ConnectMessaging, topic string) *Producer {
	return &Producer{
		messaging: messaging,
		topic:     topic,
	}
}

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	return p.messaging.PublishMessage(ctx, p.topic, key, value)
}
