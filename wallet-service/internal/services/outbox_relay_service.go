package services

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"go-task-wallet-service/shared/logging"
	models "go-task-wallet-service/shared/pkg/models"
	"go-task-wallet-service/wallet-service/internal/domain"
)

// Implementation check for type safety
var _ domain.OutboxRelayService = (*OutboxRelay)(nil)

// OutboxRelay has two distinct public functionalities one is CheckAndRelay and InsertOutboxRelay
// CheckAndRelay is a poller which polls for configured amount of time and batch size retrieves all pending outbox rows
// And publishes message to EventPublisher
// While InsertOutboxRelay sits in the gRPC handler(controller) once that gRPC method is invoked, through wallet repository
// it is being inserted in the transaction_outbox table
type OutboxRelay struct {
	walletRepository domain.WalletRepository
	eventPublisher   domain.EventPublisher
	topic            string
	batchSize        int
	pollInterval     time.Duration
	logging.Logger
}

// Initializing outbox struct which encapsulated two methods, One is basically a poller second one is what poller will listen two, Outbox event inserts
func NewOutboxRelayService(walletRepository domain.WalletRepository, eventPublisher domain.EventPublisher, topic string, batchSize int, pollInterval time.Duration) *OutboxRelay {
	return &OutboxRelay{
		walletRepository: walletRepository,
		eventPublisher:   eventPublisher,
		topic:            topic,
		batchSize:        batchSize,
		pollInterval:     pollInterval,
		Logger:           logging.NewInternalLogger(),
	}
}

func (o *OutboxRelay) InsertOutboxRelay(ctx context.Context, aggregateType, aggregateId, eventType, partitionKey, userId string, payload map[string]interface{}) error {
	return o.walletRepository.OutboxInsert(ctx, &domain.OutboxInsertConfig{
		AggregateType: aggregateType,
		AggregateId:   aggregateId,
		EventType:     eventType,
		Payload:       payload,
		UserId:        userId,
		Topic:         o.topic,
		Partition_key: partitionKey,
	})
}

// CheckAndRelay running until context cancel is called
func (o *OutboxRelay) CheckAndRelay(ctx context.Context) {
	iteration := time.NewTicker(o.pollInterval)
	defer iteration.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-iteration.C: // On every read iteration checking invoking the Db check
			// Running draining operation of rows
			for {
				n := o.readAndRelay(ctx)
				if n < o.batchSize {
					break
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}
	}
}

// Reading from the outbox table and ensuring safe and durable publish of the batch to Messaging
func (o *OutboxRelay) readAndRelay(ctx context.Context) int {
	// Defines the max amount of concurrent goroutines running to relay from outbox to event streamer
	const maxRelayConcurrency int = 32

	rows, err := o.walletRepository.OutboxPendingGetBatch(ctx, o.batchSize)
	if err != nil {
		o.LogError(ctx, "outbox relay: failed to fetch pending batch: %v", err)
		return 0
	}

	// To not overload the thread with multiple redundant reads we simply return
	if len(rows) == 0 {
		return 0
	}
	o.LogInfo(ctx, "outbox relay: found %d pending rows", len(rows))

	var wg sync.WaitGroup
	limit := make(chan struct{}, maxRelayConcurrency)
	for _, row := range rows {
		wg.Add(1)
		// Allocating goroutine on each row operation making sure they are now being processed one at a time
		go func(row *models.OutboxModel) {
			limit <- struct{}{}
			defer wg.Done()
			defer func() { <-limit }()

			rowCtx := logging.WithRequestID(ctx, row.AggregateId)

			payload, err := json.Marshal(row.Payload)
			if err != nil {
				o.LogError(rowCtx, "Outbox relay: failed to marshal payload for aggregate_id=%s: %v", row.AggregateId, err)
				o.markFailed(rowCtx, row)
				return
			}

			message, err := json.Marshal(domain.Envelope{
				EventId:     row.ID,
				EventType:   row.EventType,
				AggregateId: row.AggregateId,
				OccurredAt:  row.CreatedAt,
				Version:     domain.EnvelopeVersion,
				Payload:     payload,
			})
			if err != nil {
				o.LogError(rowCtx, "outbox relay: failed to marshal envelope for aggregate_id=%s: %v", row.AggregateId, err)
				o.markFailed(rowCtx, row)
				return
			}

			// Publishing to the relevant partition
			o.LogInfo(ctx, "row structure: %v,  publishing aggregate_id=%s to partition_key=%s", row, row.AggregateId, row.PartitionKey)
			if err := o.eventPublisher.Publish(rowCtx, row.PartitionKey, message); err != nil {
				o.LogError(rowCtx, "outbox relay: failed to publish aggregate_id=%s: %v", row.AggregateId, err)

				// Recording failure and subsequent retries, if the failure is permanent then we will mark it as failed and release the idempotency lock. This ensures not every publishing failure is being treated as permanent. and the system has ability to heal itself without losing messages and falling into an inconsistent state
				status, markErr := o.walletRepository.OutboxMarkPublishFailure(rowCtx, row.AggregateId, err.Error())
				if markErr != nil {
					o.LogError(rowCtx, "outbox relay: failed to record publish failure for aggregate_id=%s: %v", row.AggregateId, markErr)
				} else if status == string(models.OutboxStatusFailed) {
					o.releaseIdempotency(rowCtx, row)
				}
				return
			}

			if err := o.walletRepository.OutboxUpdate(rowCtx, row.AggregateId, string(models.OutboxStatusPublished)); err != nil {
				o.LogError(rowCtx, "outbox relay: failed to mark aggregate_id=%s published: %v", row.AggregateId, err)
			}
		}(row)
	}
	wg.Wait()

	return len(rows)
}

// Marking messages failed in case we receive an invalid data
func (o *OutboxRelay) markFailed(ctx context.Context, row *models.OutboxModel) {
	if err := o.walletRepository.OutboxUpdate(ctx, row.AggregateId, string(models.OutboxStatusFailed)); err != nil {
		o.LogError(ctx, "outbox relay: failed to mark aggregate_id=%s failed: %v", row.AggregateId, err)
	}
	o.releaseIdempotency(ctx, row)
}

// Releases the per-account idempotency lock — only safe to call once a row
// has reached a terminal status (failed), since a row still queued for
// backoff retry is still logically "in flight" for that account.
func (o *OutboxRelay) releaseIdempotency(ctx context.Context, row *models.OutboxModel) {
	userId := strings.TrimSuffix(row.IdempotencyKey, "_"+row.PartitionKey)
	if err := o.walletRepository.IdempotencyRelease(ctx, row.AggregateId, userId, row.PartitionKey); err != nil {
		o.LogError(ctx, "outbox relay: failed to release idempotency for aggregate_id=%s: %v", row.AggregateId, err)
	}
}
