-- +goose Up
-- Converting amount and balances to minor units as per task
ALTER TABLE accounts
    ALTER COLUMN balance TYPE BIGINT USING ROUND(balance * 100)::BIGINT,
    ALTER COLUMN balance SET DEFAULT 0;

ALTER TABLE transactions
    ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT;

-- +goose Down
ALTER TABLE accounts
    ALTER COLUMN balance TYPE NUMERIC(20, 8) USING (balance::NUMERIC / 100),
    ALTER COLUMN balance SET DEFAULT 0;

ALTER TABLE transactions
    ALTER COLUMN amount TYPE NUMERIC(20, 8) USING (amount::NUMERIC / 100);
