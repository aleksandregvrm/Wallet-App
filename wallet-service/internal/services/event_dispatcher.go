package services

import (
	"context"
	"encoding/json"
	"fmt"
	"go-task-wallet-service/shared/events"
	"go-task-wallet-service/shared/logging"
	"go-task-wallet-service/wallet-service/internal/domain"
)

type EventDispatcher struct {
	handlers map[string]EventHandler
	logging.Logger
}

// Callback function structure that will execute a logic on a consumed message
// Common structure for every handler which satisfies the basic consuming structure
type EventHandler func(ctx context.Context, aggregateId string, payload json.RawMessage) error

func NewEventDispatcherService() *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string]EventHandler),
		Logger:   logging.NewInternalLogger(),
	}
}

// Registering an evnt handled based on the eventType input, Once the match is made the handler will handle the event accordingly
func (d *EventDispatcher) Register(eventType string, handler EventHandler) {
	d.handlers[eventType] = handler
}

func (d *EventDispatcher) Handle(ctx context.Context, message []byte) error {
	var envelope domain.Envelope
	if err := json.Unmarshal(message, &envelope); err != nil {
		return fmt.Errorf("%w: Event dispatcher: failed to decode envelope: %v", events.ErrInvalidMessage, err)
	}

	handler, ok := d.handlers[envelope.EventType]
	if !ok {
		return fmt.Errorf("%w: Event dispatcher: no handler registered for event_type=%s", events.ErrInvalidMessage, envelope.EventType)
	}

	// Injecting the request Id in the context which will be server upstream
	ctx = logging.WithRequestID(ctx, envelope.AggregateId)

	d.LogInfo(ctx, "%s logging envelope id and type to dispatch, type: %s", envelope.AggregateId, envelope.EventType)

	if err := handler(ctx, envelope.AggregateId, envelope.Payload); err != nil {
		return fmt.Errorf("Event dispatchr: handler for event_type=%s failed: %w", envelope.EventType, err)
	}

	return nil
}
