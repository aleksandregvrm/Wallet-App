package cache

// This module defines a common ground through which each service would be able to connect to the cache backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/logging"

	"github.com/redis/go-redis/v9"
)

var (
	redisHost     = env.GetString("REDIS_HOST", "localhost")
	redisPort     = env.GetString("REDIS_PORT", "6379")
	redisPassword = env.GetString("REDIS_PASSWORD", "test")
	redisDB       = env.GetInt("REDIS_DB", 0)
)

var _ Instance = (*RedisInstance)(nil) // Interface implementation check

type RedisInstance struct {
	mu     sync.Mutex // Mutex locking for thread-safe access
	client *redis.Client
	logging.Logger
}

func NewRedisInstance() Instance {
	return &RedisInstance{
		Logger: logging.NewInternalLogger(),
	}
}

func (r *RedisInstance) Connect(serviceName string) error {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password:     redisPassword, // At this point setup only requires password for connection
		DB:           redisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("connecting to redis: %w", err)
	}

	r.mu.Lock()
	r.client = client
	r.mu.Unlock()

	r.LogDebug(ctx, "%s - Connected to redis at %s:%s", serviceName, redisHost, redisPort)

	return nil
}

func (r *RedisInstance) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.client != nil {
		return r.client.Close()
	}

	return nil
}

func (r *RedisInstance) Client() *redis.Client {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.client
}
