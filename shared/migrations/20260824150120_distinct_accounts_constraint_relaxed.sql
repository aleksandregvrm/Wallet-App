-- +goose Up
-- Structure that enables Deposit and Withdrawal flows to work since we will have from and to for the same account id
-- Such constraint release is essential to follow the same structure
ALTER TABLE transactions
DROP CONSTRAINT chk_transactions_distinct_accounts;

ALTER TABLE transactions ADD CONSTRAINT chk_transactions_distinct_accounts CHECK (
    transaction_type <> 'transfer'
    OR from_account <> to_account
);

-- +goose Down
-- Just in case the structure is modified rolling back the migration
ALTER TABLE transactions
DROP CONSTRAINT chk_transactions_distinct_accounts;

ALTER TABLE transactions ADD CONSTRAINT chk_transactions_distinct_accounts CHECK (from_account <> to_account);