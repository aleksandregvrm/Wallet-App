package cache

import (
	types "go-task-wallet-service/shared/pkg/models"

	"github.com/redis/go-redis/v9"
)

// General contract for caching in RAM
// To be used not as a persistent state but rather as an Idempotency check
// And user session management
type Cache interface {
	// Session management

	// Token caching functionality is designed for user session management
	// internally it uses access tokens TTL for automatic cleanup and proceeding session termination
	StoreToken(userId, refreshToken string) (string, error)
	FindTokenByUserId(userId string) (*types.UserAuthModel, error)
	DeleteTokenByUserId(userId string) error

	// Transaction idempotency related functionality. Designed to prevent processing of the same transaction twice

	TransactionIdempotencyCheck(transactionId string) (bool, error) // idempotency check prevents processing of same transaction twice
	// After finalizing transaction processing we release the idempotency hence deleting it from cache
	TransactionIdempotencyRelease(transactionId string) error

	// Balance related functionality. Designed to cache the balance of an account to prevent stampede

	// Getting the newly updated account balance which in case not present is being stored in cache for future requests
	GetOrStoreAccountBalance(accountId string, amount int64) (int64, error)

	// Invalidating the account balance which is explicitly called after a transaction is processed to prevent invalid data being served from cache
	InvalidateAccountBalance(accountId string) error

	Close() error
}

// Instance is the low-level connection contract for a caching backend.
//
//	NewCacheClient depends on this interface rather
//
// than a concrete Redis setup, so the caching backend is swappable.
type Instance interface {
	Connect(serviceName string) error
	Close() error
	Client() *redis.Client
}
