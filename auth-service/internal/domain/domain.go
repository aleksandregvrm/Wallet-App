package domain

import (
	"context"
	"go-task-wallet-service/auth-service/internal/pkg"
	shareddomain "go-task-wallet-service/shared/pkg/domain"
	types "go-task-wallet-service/shared/pkg/models"
	"time"
)

// Domain UserAuth
type UserAuth struct {
	ID                  string
	AccessToken         string
	UserId              string
	IsCurrentlyLoggedIn bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Type alias for User struct from shared domain
type User = shareddomain.User

type AuthService interface {
	RegisterUser(ctx context.Context, user *User) (*pkg.AuthenticateUserResponse, error)
	LoginUser(ctx context.Context, username, password string) (*pkg.AuthenticateUserResponse, error)
	AuthorizeUser(ctx context.Context, userId string) error
	RefreshToken(ctx context.Context, userId string) string
}

type AuthRepository interface {
	FindByUserId(ctx context.Context, userId string) (*types.UserModel, error)
	FindByUsername(ctx context.Context, username string) (*types.UserModel, error)
	InsertUser(ctx context.Context, name, email, username, userId, password, refreshToken string) (*types.UserModel, error)
	UpdateUser(ctx context.Context, name, email, username, userId string) (*types.UserModel, error)
	GetTokenByUserId(ctx context.Context, userId string) (*types.UserAuthModel, error)
	InsertToken(ctx context.Context, userId, refreshToken string) (bool, error)
	DeleteToken(ctx context.Context, userId string) error
}
