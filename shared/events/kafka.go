package events

import (
	"context"
	"errors"
	"fmt"
	"go-task-wallet-service/shared/logging"
	"go-task-wallet-service/shared/retry"
	"io"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// Each topic would have it's own dedicated dlq topic - {topic}.dlq with this naming convention
const DlqTopicSuffix = ".dlq"

var _ ConnectMessaging = (*KafkaMessaging)(nil)

type KafkaMessaging struct {
	brokers []string
	groupID string

	mu      sync.Mutex
	writer  *kafka.Writer
	readers map[string]*kafka.Reader
	wg      sync.WaitGroup

	logging.Logger
}

func NewKafkaMessagingInstance(groupID string, brokers []string) (*KafkaMessaging, error) {
	// In case of no brokers provided, return an error
	if len(brokers) == 0 {
		return nil, errors.New("messaging: at least one broker address is required")
	}

	return &KafkaMessaging{
		brokers: brokers,
		groupID: groupID,
		readers: make(map[string]*kafka.Reader),
		Logger:  logging.NewInternalLogger(),
	}, nil
}

func (k *KafkaMessaging) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Retrying the connection with a backoff strategy to handle transient errors
	err := retry.WithBackoff(ctx, retry.DefaultConfig(), func() error {
		conn, dialErr := kafka.DialContext(ctx, "tcp", k.brokers[0])

		// Attempt classification of the error
		if dialErr != nil {
			return fmt.Errorf("dial %s: %w", k.brokers[0], dialErr)
		}
		defer conn.Close()

		// Connection Cluster issues
		_, brokerErr := conn.Brokers()

		if brokerErr != nil {
			return fmt.Errorf("cluster not ready: %w", brokerErr)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Connection to kafka has failed: %w", err)
	}

	// Locking the mutex here for thread safety
	k.mu.Lock()

	defer k.mu.Unlock() // Unlocking the mutex releasing the thread

	k.writer = &kafka.Writer{
		Addr:                   kafka.TCP(k.brokers...),
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: false,
		BatchTimeout:           5 * time.Millisecond,
	}

	return nil
}

// Implementation for appending message to DLQ
func (k *KafkaMessaging) AppendToDlq(ctx context.Context, topic, key string, message []byte, reason string) error {
	// Locking the thread
	k.mu.Lock()
	writer := k.writer
	k.mu.Unlock()

	if writer == nil {
		return errors.New("Kafka writer is not initialized")
	}

	dlqTopic := topic + DlqTopicSuffix

	msg := kafka.Message{
		Topic: dlqTopic,
		Key:   []byte(key),
		Value: message,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{Key: "x-failure-reason", Value: []byte(reason)}, // The failure reason defined
			{Key: "x-source-topic", Value: []byte(topic)},    // Topic where failure occured
		},
	}

	writeErr := writer.WriteMessages(ctx, msg)

	if writeErr != nil {
		return fmt.Errorf("Failed to write message to DLQ topic %s: %w", dlqTopic, writeErr)
	}

	return nil
}

func (k *KafkaMessaging) PublishMessage(ctx context.Context, topic, key string, message []byte) error {
	// Locking the thread
	k.mu.Lock()
	writer := k.writer
	k.mu.Unlock()

	if writer == nil {
		return errors.New("Kafka writer is not initialized")
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: message,
		Time:  time.Now(),
	}

	err := writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("Failed to write message to topic %s: %w", topic, err)
	}

	return nil
}

func (k *KafkaMessaging) ConsumeMessage(ctx context.Context, topic string, handler func(ctx context.Context, message []byte) error) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     k.brokers,
		Topic:       topic,
		GroupID:     k.groupID,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})

	k.mu.Lock()
	k.readers[topic] = reader
	k.mu.Unlock()

	// Allocation of goroutine thread to consume the incoming message
	// Iterating over the message with dispatched function (function as an argument).
	// Constantly checks for the arriving message
	k.wg.Add(1)
	go func() {
		defer k.wg.Done()

		// Retry policy in case there's a downtime to make sure the connection to the broker is retried
		const initialBackoff = 1 * time.Second
		const maxBackoff = 30 * time.Second
		backoff := initialBackoff

		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, context.Canceled) {
					return // reader closed during shutdown
				}

				k.LogError(ctx, "Fetching messages has failed for topic:%s, error:%v, retrying in %v", topic, err, backoff)

				select {
				// The only cause of failure should be the context exit, Other errors are subject to retries
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}

				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}

			// Backoff is reset after a successful consume operation
			backoff = initialBackoff

			if dispatchErr := k.dispatch(ctx, topic, msg, handler); dispatchErr != nil {
				if errors.Is(dispatchErr, ErrInvalidMessage) {
					// Messages that ar deemed as failed are routed to the DLQ on their respective topics
					k.LogError(ctx, "Messaging: invalid message, routing to DLQ, topic:%s, partition:%d, offset:%d, error:%v",
						topic, msg.Partition, msg.Offset, dispatchErr)

					// The actual handler thrown error routed to the dlq
					if dlqErr := k.AppendToDlq(ctx, topic, string(msg.Key), msg.Value, dispatchErr.Error()); dlqErr != nil {
						k.LogError(ctx, "Messaging: failed to route invalid message to DLQ, offset not committed, topic:%s, partition:%d, offset:%d, error:%v",
							topic, msg.Partition, msg.Offset, dlqErr)
						continue
					}

					if err := reader.CommitMessages(ctx, msg); err != nil {
						k.LogError(ctx, "Messaging: commit failed after DLQ routing, topic:%s, partition:%d, offset:%d, error:%v",
							topic, msg.Partition, msg.Offset, err)
					}
					continue
				}

				k.LogError(ctx, "Messaging: handler failed, offset not committed, topic:%s, partition:%d, offset:%d, error:%v",
					topic, msg.Partition, msg.Offset, dispatchErr)
				continue
			}

			if err := reader.CommitMessages(ctx, msg); err != nil {
				k.LogError(ctx, "Messaging: commit failed, topic:%s, partition:%d, offset:%d, error:%v",
					topic, msg.Partition, msg.Offset, err)
			}
		}
	}()

	return nil
}

// Dispatching a callback function on an event
func (k *KafkaMessaging) dispatch(ctx context.Context, topic string, msg kafka.Message, handler func(ctx context.Context, message []byte) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			k.LogError(ctx, "Messaging: handler panicked, topic:%s, partition:%d, offset:%d, panic:%v",
				topic, msg.Partition, msg.Offset, r)
			err = fmt.Errorf("handler panicked: %v", r)
		}
	}()

	return handler(ctx, msg.Value)
}

func (k *KafkaMessaging) CloseConnection() error {
	k.mu.Lock()

	var errs []error

	if k.writer != nil {
		if err := k.writer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("Messaging: close writer: %w", err))
		}
	}

	for topic, r := range k.readers {
		if err := r.Close(); err != nil {
			errs = append(errs, fmt.Errorf("Messaging: close readr for %s: %w", topic, err))
		}
	}

	k.mu.Unlock()

	// Unlocks the thread
	k.wg.Wait()

	return errors.Join(errs...)
}
