-- +goose Up
-- Making sure at a database level we won't be having two outbox writes with the same aggregate_id
ALTER TABLE transaction_outbox
    ADD CONSTRAINT transaction_outbox_aggregate_id_key UNIQUE (aggregate_id);

-- +goose Down
ALTER TABLE transaction_outbox
    DROP CONSTRAINT IF EXISTS transaction_outbox_aggregate_id_key;