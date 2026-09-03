package unit_tests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go-task-wallet-service/shared/events"
	"go-task-wallet-service/wallet-service/internal/domain"
	"go-task-wallet-service/wallet-service/internal/services"
)

func TestEventDispatcher_Handle_Success(t *testing.T) {
	dispatcher := services.NewEventDispatcherService()

	var gotAggregateId string
	var gotPayload json.RawMessage
	dispatcher.Register("test.event", func(ctx context.Context, aggregateId string, payload json.RawMessage) error {
		gotAggregateId = aggregateId
		gotPayload = payload
		return nil
	})

	payload := json.RawMessage(`{"foo":"bar"}`)
	message, err := json.Marshal(domain.Envelope{EventType: "test.event", AggregateId: "agg-1", Version: domain.EnvelopeVersion, Payload: payload})
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	if err := dispatcher.Handle(context.Background(), message); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAggregateId != "agg-1" {
		t.Fatalf("unexpected aggregateId passed to handler: %q", gotAggregateId)
	}
	if string(gotPayload) != string(payload) {
		t.Fatalf("unexpected payload passed to handler: %s", gotPayload)
	}
}

func TestEventDispatcher_Handle_NoHandlerRegistered(t *testing.T) {
	dispatcher := services.NewEventDispatcherService()

	message, err := json.Marshal(domain.Envelope{EventType: "unregistered.event", AggregateId: "agg-1"})
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	err = dispatcher.Handle(context.Background(), message)
	if !errors.Is(err, events.ErrInvalidMessage) {
		t.Fatalf("expected a wrapped ErrInvalidMessage, got: %v", err)
	}
}

func TestEventDispatcher_Handle_MalformedEnvelope(t *testing.T) {
	dispatcher := services.NewEventDispatcherService()

	err := dispatcher.Handle(context.Background(), []byte("not-json"))
	if !errors.Is(err, events.ErrInvalidMessage) {
		t.Fatalf("expected a wrapped ErrInvalidMessage, got: %v", err)
	}
}

func TestEventDispatcher_Handle_HandlerError(t *testing.T) {
	dispatcher := services.NewEventDispatcherService()
	wantErr := errors.New("handler exploded")
	dispatcher.Register("test.event", func(ctx context.Context, aggregateId string, payload json.RawMessage) error {
		return wantErr
	})

	message, err := json.Marshal(domain.Envelope{EventType: "test.event", AggregateId: "agg-1"})
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	err = dispatcher.Handle(context.Background(), message)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}
