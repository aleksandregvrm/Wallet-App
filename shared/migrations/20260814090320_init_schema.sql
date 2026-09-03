-- +goose Up
/* Users */
CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL,
    username    TEXT NOT NULL,
    password    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on id (explicit, in addition to PK)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_id ON users (id);

-- Index on username column, Since it will be part of transaction querying
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);

-- Index on email
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);


/* Accounts */

CREATE TABLE IF NOT EXISTS accounts (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    balance     NUMERIC(20, 8) NOT NULL DEFAULT 0,
    currency    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Note: Account.Transactions in the Go struct is a relation (loaded via join/query),
-- not a stored column — it maps to rows in the transactions table below.

-- Index on id (explicit, in addition to PK)
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_id ON accounts (id);

-- Index on user_id — fast lookup of "all accounts for this user"
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts (user_id);

/* Transactions */
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE transaction_status AS ENUM ('pending', 'completed', 'failed', 'rejected');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS transactions (
    id            TEXT PRIMARY KEY,
    from_account  TEXT NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    to_account    TEXT NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    amount        NUMERIC(20, 8) NOT NULL CHECK (amount > 0),
    currency      TEXT NOT NULL,
    status        transaction_status NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_transactions_distinct_accounts CHECK (from_account <> to_account)
);

-- Index on id (explicit, in addition to PK)
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_id ON transactions (id);

-- Index on account — covers both legs of the transfer so lookups by either side are fast
CREATE INDEX IF NOT EXISTS idx_transactions_from_account ON transactions (from_account);
CREATE INDEX IF NOT EXISTS idx_transactions_to_account ON transactions (to_account);

/* User Auth */
-- One row per user tracking their current login/session state, matching the
-- Get/Insert/Delete-by-user_id shape of AuthRepository (auth-service/internal/repository).
CREATE TABLE IF NOT EXISTS user_auth (
    id                      TEXT PRIMARY KEY,
    user_id                 TEXT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    access_token            TEXT,
    is_currently_logged_in  BOOLEAN NOT NULL DEFAULT false,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on id (explicit, in addition to PK)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auth_id ON user_auth (id);

-- Index on user_id — enforces a single auth/session record per user and speeds up token lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auth_user_id ON user_auth (user_id);


-- +goose Down
-- Reverse dependency order: user_auth/transactions/accounts reference users
-- (accounts/transactions also reference each other), so they must drop first.
DROP TABLE IF EXISTS user_auth;
DROP TABLE IF EXISTS transactions;
DROP TYPE IF EXISTS transaction_status;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;
