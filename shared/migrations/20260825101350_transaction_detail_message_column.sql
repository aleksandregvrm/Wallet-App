-- +goose Up
-- Dedicated column to write the detailed message of the transaction
ALTER TABLE transactions
ADD COLUMN transaction_detail_message TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE transactions DROP COLUMN transaction_detail_message;