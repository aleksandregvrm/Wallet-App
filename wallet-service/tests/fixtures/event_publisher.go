package fixtures

import (
	"context"
	"fmt"

	"go-task-wallet-service/wallet-service/internal/domain"
)

var _ domain.EventPublisher = (*FakeEventPublisher)(nil)

type FakeEventPublisher struct {
	PublishFunc func(ctx context.Context, key string, value []byte) error
}

func (f *FakeEventPublisher) Publish(ctx context.Context, key string, value []byte) error {
	if f.PublishFunc == nil {
		return fmt.Errorf("fixtures: PublishFunc not set")
	}
	return f.PublishFunc(ctx, key, value)
}
