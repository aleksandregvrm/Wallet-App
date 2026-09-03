package unit_tests

import (
	"context"
	"testing"
	"time"

	types "go-task-wallet-service/shared/pkg/models"
	"go-task-wallet-service/wallet-service/internal/domain"
	"go-task-wallet-service/wallet-service/internal/services"
	"go-task-wallet-service/wallet-service/tests/fixtures"
)

func TestOutboxRelay_InsertOutboxRelay_DelegatesToRepository(t *testing.T) {
	var gotConfig *domain.OutboxInsertConfig
	repo := &fixtures.FakeWalletRepository{
		OutboxInsertFunc: func(ctx context.Context, cfg *domain.OutboxInsertConfig) error {
			gotConfig = cfg
			return nil
		},
	}
	relay := services.NewOutboxRelayService(repo, &fixtures.FakeEventPublisher{}, "wallet.events.v1", 10, time.Hour)

	payload := map[string]interface{}{"ID": "tx-1"}
	err := relay.InsertOutboxRelay(context.Background(), "transaction", "tx-1", "transaction.deposit.requested", "account-1", "user-1", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotConfig.AggregateType != "transaction" || gotConfig.AggregateId != "tx-1" || gotConfig.EventType != "transaction.deposit.requested" {
		t.Fatalf("unexpected config: %+v", gotConfig)
	}
	if gotConfig.UserId != "user-1" || gotConfig.Partition_key != "account-1" || gotConfig.Topic != "wallet.events.v1" {
		t.Fatalf("unexpected config: %+v", gotConfig)
	}
}

type outboxUpdateCall struct {
	aggregateId string
	status      string
}

func TestOutboxRelay_CheckAndRelay_PublishesAndMarksPublished(t *testing.T) {
	fetched := false
	updateCh := make(chan outboxUpdateCall, 1)
	publishCh := make(chan string, 1)

	repo := &fixtures.FakeWalletRepository{
		OutboxPendingGetBatchFunc: func(ctx context.Context, limit int) ([]*types.OutboxModel, error) {
			if fetched {
				return nil, nil
			}
			fetched = true
			return []*types.OutboxModel{{
				ID:             "outbox-1",
				AggregateType:  "transaction",
				AggregateId:    "tx-1",
				EventType:      "transaction.deposit.requested",
				Payload:        map[string]interface{}{"ID": "tx-1"},
				Topic:          "wallet.events.v1",
				PartitionKey:   "account-1",
				IdempotencyKey: "user-1_account-1",
				Status:         types.OutboxStatusPending,
				CreatedAt:      time.Now(),
			}}, nil
		},
		OutboxUpdateFunc: func(ctx context.Context, aggregateId, status string) error {
			updateCh <- outboxUpdateCall{aggregateId: aggregateId, status: status}
			return nil
		},
	}
	publisher := &fixtures.FakeEventPublisher{
		PublishFunc: func(ctx context.Context, key string, value []byte) error {
			publishCh <- key
			return nil
		},
	}

	relay := services.NewOutboxRelayService(repo, publisher, "wallet.events.v1", 10, 5*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go relay.CheckAndRelay(ctx)

	select {
	case key := <-publishCh:
		if key != "account-1" {
			t.Fatalf("expected publish to be partitioned by account-1, got: %s", key)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for Publish to be called")
	}

	select {
	case call := <-updateCh:
		if call.aggregateId != "tx-1" || call.status != string(types.OutboxStatusPublished) {
			t.Fatalf("unexpected update call: %+v", call)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for OutboxUpdate(published) to be called")
	}
}

type publishFailureCall struct{ aggregateId, errMsg string }

func TestOutboxRelay_CheckAndRelay_PublishFailure_Retries_NoIdempotencyReleaseYet(t *testing.T) {
	fetched := false
	markCh := make(chan publishFailureCall, 1)
	releaseCalled := false

	repo := &fixtures.FakeWalletRepository{
		OutboxPendingGetBatchFunc: func(ctx context.Context, limit int) ([]*types.OutboxModel, error) {
			if fetched {
				return nil, nil
			}
			fetched = true
			return []*types.OutboxModel{{
				ID:             "outbox-1",
				AggregateId:    "tx-1",
				EventType:      "transaction.deposit.requested",
				Payload:        map[string]interface{}{"ID": "tx-1"},
				PartitionKey:   "account-1",
				IdempotencyKey: "user-1_account-1",
				CreatedAt:      time.Now(),
			}}, nil
		},
		OutboxMarkPublishFailureFunc: func(ctx context.Context, aggregateId, errMsg string) (string, error) {
			markCh <- publishFailureCall{aggregateId: aggregateId, errMsg: errMsg}
			// Simulates a retry still available (below outboxMaxPublishRetries).
			return string(types.OutboxStatusPending), nil
		},
		IdempotencyReleaseFunc: func(ctx context.Context, transactionId, userId, accountId string) error {
			releaseCalled = true
			return nil
		},
	}
	publisher := &fixtures.FakeEventPublisher{
		PublishFunc: func(ctx context.Context, key string, value []byte) error {
			return context.DeadlineExceeded
		},
	}

	relay := services.NewOutboxRelayService(repo, publisher, "wallet.events.v1", 10, 5*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go relay.CheckAndRelay(ctx)

	select {
	case call := <-markCh:
		if call.aggregateId != "tx-1" || call.errMsg == "" {
			t.Fatalf("unexpected publish-failure call: %+v", call)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for OutboxMarkPublishFailure to be called")
	}

	// Negative assertion: a still-retryable failure must not release the
	// per-account lock — the transaction is still logically in flight.
	time.Sleep(50 * time.Millisecond)
	cancel()

	if releaseCalled {
		t.Fatalf("expected IdempotencyRelease not to be called while retries remain")
	}
}

func TestOutboxRelay_CheckAndRelay_PublishFailure_ExhaustsRetries_ReleasesIdempotency(t *testing.T) {
	fetched := false
	type releaseCall struct{ transactionId, userId, accountId string }
	releaseCh := make(chan releaseCall, 1)

	repo := &fixtures.FakeWalletRepository{
		OutboxPendingGetBatchFunc: func(ctx context.Context, limit int) ([]*types.OutboxModel, error) {
			if fetched {
				return nil, nil
			}
			fetched = true
			return []*types.OutboxModel{{
				ID:             "outbox-1",
				AggregateId:    "tx-1",
				EventType:      "transaction.deposit.requested",
				Payload:        map[string]interface{}{"ID": "tx-1"},
				PartitionKey:   "account-1",
				IdempotencyKey: "user-1_account-1",
				CreatedAt:      time.Now(),
			}}, nil
		},
		OutboxMarkPublishFailureFunc: func(ctx context.Context, aggregateId, errMsg string) (string, error) {
			// Simulates outboxMaxPublishRetries having been reached.
			return string(types.OutboxStatusFailed), nil
		},
		IdempotencyReleaseFunc: func(ctx context.Context, transactionId, userId, accountId string) error {
			releaseCh <- releaseCall{transactionId: transactionId, userId: userId, accountId: accountId}
			return nil
		},
	}
	publisher := &fixtures.FakeEventPublisher{
		PublishFunc: func(ctx context.Context, key string, value []byte) error {
			return context.DeadlineExceeded
		},
	}

	relay := services.NewOutboxRelayService(repo, publisher, "wallet.events.v1", 10, 5*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go relay.CheckAndRelay(ctx)

	select {
	case call := <-releaseCh:
		if call.transactionId != "tx-1" || call.userId != "user-1" || call.accountId != "account-1" {
			t.Fatalf("unexpected idempotency release call: %+v", call)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for IdempotencyRelease to be called")
	}
}

func TestOutboxRelay_CheckAndRelay_EmptyBatch_NeverPublishes(t *testing.T) {
	publishCalled := false
	repo := &fixtures.FakeWalletRepository{
		OutboxPendingGetBatchFunc: func(ctx context.Context, limit int) ([]*types.OutboxModel, error) {
			return nil, nil
		},
	}
	publisher := &fixtures.FakeEventPublisher{
		PublishFunc: func(ctx context.Context, key string, value []byte) error {
			publishCalled = true
			return nil
		},
	}

	relay := services.NewOutboxRelayService(repo, publisher, "wallet.events.v1", 10, 5*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	go relay.CheckAndRelay(ctx)

	// Negative assertion: give a few poll cycles a chance to run, then confirm
	// nothing was published for an empty batch.
	time.Sleep(50 * time.Millisecond)
	cancel()

	if publishCalled {
		t.Fatalf("expected Publish never to be called for an empty pending batch")
	}
}
