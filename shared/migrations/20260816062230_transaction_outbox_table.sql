-- +goose Up
CREATE TYPE outbox_status AS ENUM ('pending', 'published', 'failed');

CREATE TABLE
    IF NOT EXISTS transaction_outbox (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        aggregate_type TEXT NOT NULL,
        aggregate_id TEXT NOT NULL,
        event_type TEXT NOT NULL,
        -- Kafka specific fields
        topic TEXT NOT NULL,
        partition_key TEXT, -- Routing of the message
        status outbox_status NOT NULL DEFAULT 'pending',
        error_message TEXT DEFAULT NULL,
        -- Timing of the updates and creation
        created_at TIMESTAMPTZ NOT NULL DEFAULT now (),
        published_at TIMESTAMPTZ, -- null until relay confirms Kafka ack
        updated_at TIMESTAMPTZ NOT NULL DEFAULT now ()
    );

-- Ensuring that listing prioritizes unpublished messages first
CREATE INDEX idx_transaction_outbox_unpublished ON transaction_outbox (created_at)
WHERE
    status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_transaction_outbox_unpublished;
DROP TABLE IF EXISTS transaction_outbox;