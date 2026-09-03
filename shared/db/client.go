package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DbClient is the entrypoint services use to run queries against
// DB (Postgres in this case). Embedding *pgxpool.Pool gives callers Exec/Query/QueryRow
// directly, on top of the transaction helper below.
type DBClient struct {
	*pgxpool.Pool
}

var (
	postgres = "postgres"
)

// NewDBClient builds a client from an already-connected instance.
// DB agnostic client initiator, Extendable to every DB which provides a pool functionality to connect to it
func NewDBClient(instance Persistence) (*DBClient, error) {
	pool := instance.Pool()
	if pool == nil {
		return nil, fmt.Errorf("%s: instance is not connected, call Connect() first", postgres)
	}

	return &DBClient{Pool: pool}, nil
}

// WithTransaction runs fn inside a single Postgres transaction, committing on
// success and rolling back if fn returns an error or fn panics.
func (c *DBClient) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
