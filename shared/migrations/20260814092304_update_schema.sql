-- +goose Up
ALTER TABLE IF EXISTS transactions ADD COLUMN transaction_type TEXT NOT NULL;

-- +goose Down
ALTER TABLE IF EXISTS transactions DROP COLUMN transaction_type;
