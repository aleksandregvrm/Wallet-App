-- +goose Up
ALTER TABLE accounts ADD COLUMN reconciled_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE accounts DROP COLUMN reconciled_at;
