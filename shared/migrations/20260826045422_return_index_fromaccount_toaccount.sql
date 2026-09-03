-- +goose Up
CREATE INDEX IF NOT EXISTS idx_transactions_from_account ON transactions (from_account);

CREATE INDEX IF NOT EXISTS idx_transactions_to_account ON transactions (to_account);

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_from_account;

DROP INDEX IF EXISTS idx_transactions_to_account;

