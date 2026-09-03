package container

import (
	"context"
	"database/sql"
	"testing"

	"go-task-wallet-service/shared/cache"
	"go-task-wallet-service/shared/db"
	"go-task-wallet-service/shared/logging"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// Migrations file location
const migrationsDir = "../../../shared/migrations"

type TestEnv struct {
	DB    *db.DBClient
	Cache cache.Cache
}

// For some of the testing we do we need to spin up an actual instances of Database and Caching.
// In this case as we use PotgreSQL and Redis. To ensure the testing can fully mock the flow.
// We spin up the lightweight test container setup via testcontainers-go library
// Subsequently test containers are teared down in - t.Cleanup(func() { redisClient.Close() })
// and err := pgContainer.Terminate(context.Background()); err != ni

func SetupTestContainerEnv(t *testing.T) *TestEnv {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:latest",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_pass"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("container: failed to start postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(context.Background()); err != nil {
			t.Logf("container: failed to terminate postgres: %v", err)
		}
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("container: failed to get postgres connection string: %v", err)
	}

	migrationDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("container: failed to open postgres connection for migrations: %v", err)
	}
	defer migrationDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("container: failed to set goose dialect: %v", err)
	}
	if err := goose.RunContext(ctx, "up", migrationDB, migrationsDir); err != nil {
		t.Fatalf("container: failed to run migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("container: failed to create pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)

	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("container: failed to start redis: %v", err)
	}
	t.Cleanup(func() {
		if err := redisContainer.Terminate(context.Background()); err != nil {
			t.Logf("container: failed to terminate redis: %v", err)
		}
	})

	redisConnStr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("container: failed to get redis connection string: %v", err)
	}
	redisOpts, err := redis.ParseURL(redisConnStr)
	if err != nil {
		t.Fatalf("container: failed to parse redis connection string: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)
	t.Cleanup(func() { redisClient.Close() })

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("container: failed to ping redis: %v", err)
	}

	return &TestEnv{
		DB:    &db.DBClient{Pool: pool},
		Cache: &cache.CacheClient{Client: redisClient, Logger: logging.NewInternalLogger()},
	}
}
