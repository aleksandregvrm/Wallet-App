-- +goose Up
CREATE TYPE transaction_type AS ENUM ('transfer', 'deposit', 'withdrawal');

ALTER TABLE transactions ALTER COLUMN transaction_type TYPE transaction_type USING transaction_type::transaction_type;

-- +goose Down
ALTER TABLE transactions
    ALTER COLUMN transaction_type TYPE TEXT
    USING transaction_type::TEXT;


ALTER TABLE transactions transaction_type TEXT NOT NULL;