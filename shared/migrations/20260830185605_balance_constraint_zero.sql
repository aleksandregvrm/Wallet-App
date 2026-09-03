-- +goose Up
ALTER TABLE accounts ADD CONSTRAINT accounts_balance_nonnegative CHECK (balance >= 0);

-- +goose Down
ALTER TABLE accounts DROP CONSTRAINT accounts_balance_nonnegative;
