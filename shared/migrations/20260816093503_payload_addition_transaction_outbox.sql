-- +goose Up
ALTER TABLE transaction_outbox ADD COLUMN payload JSON NOT NULL;

-- +goose Down
ALTER TABLE transaction_outbox DROP COLUMN payload;
