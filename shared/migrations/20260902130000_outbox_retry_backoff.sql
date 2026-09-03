-- +goose Up
ALTER TABLE transaction_outbox ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE transaction_outbox ADD COLUMN next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE transaction_outbox DROP COLUMN retry_count;
ALTER TABLE transaction_outbox DROP COLUMN next_attempt_at;
