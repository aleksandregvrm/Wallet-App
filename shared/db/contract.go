package db

import "github.com/jackc/pgx/v5/pgxpool"

// General Contract For Database Persistence, Can be extended to multiple DB connections
type Persistence interface {
	Connect(serviceName string) error
	Close() error
	Pool() *pgxpool.Pool
}
