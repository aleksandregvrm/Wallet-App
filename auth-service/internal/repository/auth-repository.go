package repository

import (
	"context"
	"errors"
	"fmt"
	"go-task-wallet-service/shared/cache"
	"go-task-wallet-service/shared/db"
	"go-task-wallet-service/shared/logging"
	types "go-task-wallet-service/shared/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Db/Cache access layer. No business operations should be done in repository
type AuthRepository struct {
	db *db.DBClient
	ch cache.Cache
	logging.Logger
}

func NewAuthRepository(db *db.DBClient, ch cache.Cache) *AuthRepository {
	return &AuthRepository{
		db:     db,
		ch:     ch,
		Logger: logging.NewInternalLogger(),
	}
}

// User related
func (r *AuthRepository) FindByUserId(ctx context.Context, userId string) (*types.UserModel, error) {
	query := `
		SELECT id, name, username, email, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user types.UserModel
	err := r.db.Pool.QueryRow(ctx, query, userId).Scan(
		&user.ID,
		&user.Name,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user %s in DB: %w", userId, err)
	}

	return &user, nil
}

func (r *AuthRepository) FindByUsername(ctx context.Context, username string) (*types.UserModel, error) {
	query := `
		SELECT id, name, username, email, password, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user types.UserModel
	err := r.db.Pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Name,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find username %s in DB: %w", username, err)
	}

	return &user, nil

}

func (r *AuthRepository) InsertUser(ctx context.Context, name, email, username, userId, password, refreshToken string) (*types.UserModel, error) {

	query := `
		INSERT INTO users (id, name, email, username, password) VALUES ($1, $2, $3, $4, $5) RETURNING id, name, username, email
	`

	var user types.UserModel

	err := r.db.Pool.QueryRow(ctx, query, userId, name, email, username, password).Scan(
		&user.ID,
		&user.Name,
		&user.Username,
		&user.Email,
	)

	if err != nil {
		r.Logger.LogWarn(ctx, "Failed to create user: %s", userId)
		return nil, fmt.Errorf("failed to insert user in DB: %w", err)
	}

	// Caching the user token for session management
	r.ch.StoreToken(userId, refreshToken)

	return &user, nil
}

func (r *AuthRepository) UpdateUser(ctx context.Context, name, email, username, userId string) (*types.UserModel, error) {
	query := `
		UPDATE users
		SET email = $1, username = $2, name = $3
		WHERE id = $4
		RETURNING id, name, username, email, updated_at
	`

	var user types.UserModel

	err := r.db.Pool.QueryRow(ctx, query, email, username, name, userId).Scan(
		&user.ID,
		&user.Name,
		&user.Username,
		&user.Email,
		&user.UpdatedAt,
	)

	if err != nil {
		r.Logger.LogWarn(ctx, "Failed to update user: %s", userId)
		return nil, fmt.Errorf("failed to update user: %s in DB: %w", username, err)
	}

	return nil, nil
}

// User auth data related
func (r *AuthRepository) GetTokenByUserId(ctx context.Context, userId string) (*types.UserAuthModel, error) {
	query := `
		SELECT id, user_id, refresh_token, is_currently_logged_in, created_at, updated_at
		FROM user_auth
		WHERE user_id = $1
	`

	var userAuth types.UserAuthModel

	err := r.db.Pool.QueryRow(ctx, query, userId).Scan(
		&userAuth.ID,
		&userAuth.UserId,
		&userAuth.RefreshToken,
		&userAuth.IsCurrentlyLoggedIn,
		&userAuth.CreatedAt,
		&userAuth.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		r.Logger.LogWarn(ctx, "Failed to retrieve userAuth for user: %s", userId)
		return nil, fmt.Errorf("failed to retrieve userAuth: %s in DB: %w", userId, err)
	}

	return &userAuth, nil
}

func (r *AuthRepository) InsertToken(ctx context.Context, userId, refreshToken string) (bool, error) {
	// Atomic token insert ensures idempotency
	query := `
		INSERT INTO user_auth (id, user_id, refresh_token, created_at, is_currently_logged_in)
		VALUES ($1, $2, $3, now(), true)
		ON CONFLICT (user_id) DO UPDATE
		SET refresh_token = EXCLUDED.refresh_token, created_at = now(), is_currently_logged_in = true
		WHERE user_auth.is_currently_logged_in = false
		RETURNING id
	`

	var insertedId string
	err := r.db.Pool.QueryRow(ctx, query, uuid.NewString(), userId, refreshToken).Scan(&insertedId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to insert token for user %s: %w", userId, err)
	}

	if _, cacheErr := r.ch.StoreToken(userId, refreshToken); cacheErr != nil {
		r.Logger.LogWarn(ctx, "failed to cache token for user %s: %v", userId, cacheErr)
	}

	return true, nil
}

func (r *AuthRepository) DeleteToken(ctx context.Context, userId string) error {
	query := `
		DELETE FROM user_auth
		WHERE user_id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query, userId)
	if err != nil {
		return fmt.Errorf("failed to delete token for user %s: %w", userId, err)
	}

	return nil
}
