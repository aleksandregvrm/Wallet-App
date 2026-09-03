package db

// This module defines a common ground through which each service would be able to connect to DB through DB pool

import (
	"context"
	"fmt"
	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/logging"
	"go-task-wallet-service/shared/retry"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var postgresDSN = env.GetString("POSTGRES_DSN", "postgres://go_master:test@localhost:5433/?sslmode=disable")

var logger = logging.NewInternalLogger()

// Implementation check
var _ Persistence = (*PostgreSqlInstance)(nil)

type PostgreSqlInstance struct {
	mu   sync.Mutex
	pool *pgxpool.Pool // connection client through which we interact with the DB
}

func NewPostgreSqlInstance() *PostgreSqlInstance {
	return &PostgreSqlInstance{}
}

func (d *PostgreSqlInstance) Connect(serviceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.LogDebug(ctx, "%s - Connecting to postgres DSN: %s", serviceName, postgresDSN)

	err := retry.WithBackoff(ctx, retry.DefaultConfig(), func() error {
		pool, poolErr := pgxpool.New(ctx, postgresDSN)
		if poolErr != nil {
			return fmt.Errorf("create pool: %w", poolErr)
		}

		if pingErr := pool.Ping(ctx); pingErr != nil {
			pool.Close()
			return fmt.Errorf("ping: %w", pingErr)
		}

		d.mu.Lock()
		d.pool = pool
		d.mu.Unlock()

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s, connection to postgres has failed: %w", serviceName, err)
	}

	return nil
}

func (d *PostgreSqlInstance) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.pool != nil {
		d.pool.Close()
	}

	return nil
}

func (d *PostgreSqlInstance) Pool() *pgxpool.Pool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.pool
}
