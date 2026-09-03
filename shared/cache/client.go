package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/logging"
	types "go-task-wallet-service/shared/pkg/models"

	"github.com/redis/go-redis/v9"
)

const cacheOpTimeout = 3 * time.Second

const idempotencyTTL = time.Hour * 24 // Pre-defined time to live for idempotency

// Caching expiration time for token - used for user session management
var accessTokenTTL = time.Duration(env.GetInt("ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute

// Caching expiration time for the account balance read-through cache
var balanceTTL = time.Duration(env.GetInt("BALANCE_CACHE_TTL_MINUTES", 5)) * time.Minute

type CacheClient struct {
	Client *redis.Client
	logging.Logger
}

var (
	redisService = "redis"
)

// Abstract client initialization for caching, Bound to functionality not concrete tool
func NewCacheClient(instance Instance) (*CacheClient, error) {
	client := instance.Client()
	if client == nil {
		return nil, fmt.Errorf("%s: instance is not connected, call Connect() first", redisService)
	}

	return &CacheClient{Client: client, Logger: logging.NewInternalLogger()}, nil
}

// Used for idempotency checks of user session
func tokenKey(userId string) string {
	return "session:token:" + userId
}

// Used for idempotency checks for transactions
func idempotencyKey(transactionId string) string {
	return "idempotency:tx:" + transactionId
}

// Used for the account balance read-through cache
func balanceKey(accountId string) string {
	return "balance:account:" + accountId
}

func (c *CacheClient) StoreToken(userId, refreshToken string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()

	now := time.Now()
	userAuth := types.UserAuthModel{
		UserId:              userId,
		RefreshToken:        refreshToken,
		IsCurrentlyLoggedIn: true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	data, err := json.Marshal(userAuth)
	if err != nil {
		return "", fmt.Errorf("cache: encode token for user %s: %w", userId, err)
	}

	if err := c.Client.Set(ctx, tokenKey(userId), data, accessTokenTTL).Err(); err != nil {
		return "", fmt.Errorf("cache: store token for user %s: %w", userId, err)
	}

	return refreshToken, nil
}

func (c *CacheClient) FindTokenByUserId(userId string) (*types.UserAuthModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()

	val, err := c.Client.Get(ctx, tokenKey(userId)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache: get token for user %s: %w", userId, err)
	}

	var userAuth types.UserAuthModel
	if err := json.Unmarshal(val, &userAuth); err != nil {
		return nil, fmt.Errorf("cache: decode token for user %s: %w", userId, err)
	}

	return &userAuth, nil
}

func (c *CacheClient) DeleteTokenByUserId(userId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()

	if err := c.Client.Del(ctx, tokenKey(userId)).Err(); err != nil {
		return fmt.Errorf("cache: delete token for user %s: %w", userId, err)
	}

	return nil
}

func (c *CacheClient) TransactionIdempotencyCheck(transactionId string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()

	data, err := json.Marshal(transactionId)
	if err != nil {
		return false, fmt.Errorf("cache: encode transaction %s: %w", transactionId, err)
	}

	ok, err := c.Client.SetNX(ctx, idempotencyKey(transactionId), data, idempotencyTTL).Result()
	if err != nil {
		return false, fmt.Errorf("cache: insert idempotency key for transaction %s: %w", transactionId, err)
	}

	return ok, nil
}

func (c *CacheClient) TransactionIdempotencyRelease(transactionId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()

	if err := c.Client.Del(ctx, idempotencyKey(transactionId)).Err(); err != nil {
		return fmt.Errorf("cache: release idempotency key for transaction %s: %w", transactionId, err)
	}

	return nil
}

func (c *CacheClient) GetOrStoreAccountBalance(accountId string, amount int64) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()

	val, err := c.Client.Get(ctx, balanceKey(accountId)).Int64()
	if err == nil {
		return val, nil
	}
	if !errors.Is(err, redis.Nil) {
		return 0, fmt.Errorf("cache: get balance for account %s: %w", accountId, err)
	}

	if err := c.Client.Set(ctx, balanceKey(accountId), amount, balanceTTL).Err(); err != nil {
		return 0, fmt.Errorf("cache: store balance for account %s: %w", accountId, err)
	}

	return amount, nil
}

func (c *CacheClient) InvalidateAccountBalance(accountId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()

	if err := c.Client.Del(ctx, balanceKey(accountId)).Err(); err != nil {
		return fmt.Errorf("cache: invalidate balance for account %s: %w", accountId, err)
	}

	return nil
}

func (c *CacheClient) Close() error {
	return c.Client.Close()
}
