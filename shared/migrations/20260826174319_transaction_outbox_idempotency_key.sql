-- +goose Up
ALTER TABLE transaction_outbox ADD COLUMN idempotency_key TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE transaction_outbox DROP COLUMN idempotency_key;

