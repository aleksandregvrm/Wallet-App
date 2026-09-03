package fixtures

import (
	"context"
	"fmt"
	"go-task-wallet-service/auth-service/internal/domain"
	types "go-task-wallet-service/shared/pkg/models"
)

var _ domain.AuthRepository = (*FakeAuthRepository)(nil)

type FakeAuthRepository struct {
	FindByUserIdFunc   func(ctx context.Context, userId string) (*types.UserModel, error)
	FindByUsernameFunc func(ctx context.Context, username string) (*types.UserModel, error)
	InsertUserFunc     func(ctx context.Context, name, email, username, userId, password, refreshToken string) (*types.UserModel, error)
	UpdateUserFunc     func(ctx context.Context, name, email, username, userId string) (*types.UserModel, error)
	GetTokenByUserIdFunc func(ctx context.Context, userId string) (*types.UserAuthModel, error)
	InsertTokenFunc    func(ctx context.Context, userId, refreshToken string) (bool, error)
	DeleteTokenFunc    func(ctx context.Context, userId string) error
}

func (f *FakeAuthRepository) FindByUserId(ctx context.Context, userId string) (*types.UserModel, error) {
	if f.FindByUserIdFunc == nil {
		return nil, fmt.Errorf("fixtures: FindByUserIdFunc not set")
	}
	return f.FindByUserIdFunc(ctx, userId)
}

func (f *FakeAuthRepository) FindByUsername(ctx context.Context, username string) (*types.UserModel, error) {
	if f.FindByUsernameFunc == nil {
		return nil, fmt.Errorf("fixtures: FindByUsernameFunc not set")
	}
	return f.FindByUsernameFunc(ctx, username)
}

func (f *FakeAuthRepository) InsertUser(ctx context.Context, name, email, username, userId, password, refreshToken string) (*types.UserModel, error) {
	if f.InsertUserFunc == nil {
		return nil, fmt.Errorf("fixtures: InsertUserFunc not set")
	}
	return f.InsertUserFunc(ctx, name, email, username, userId, password, refreshToken)
}

func (f *FakeAuthRepository) UpdateUser(ctx context.Context, name, email, username, userId string) (*types.UserModel, error) {
	if f.UpdateUserFunc == nil {
		return nil, fmt.Errorf("fixtures: UpdateUserFunc not set")
	}
	return f.UpdateUserFunc(ctx, name, email, username, userId)
}

func (f *FakeAuthRepository) GetTokenByUserId(ctx context.Context, userId string) (*types.UserAuthModel, error) {
	if f.GetTokenByUserIdFunc == nil {
		return nil, fmt.Errorf("fixtures: GetTokenByUserIdFunc not set")
	}
	return f.GetTokenByUserIdFunc(ctx, userId)
}

func (f *FakeAuthRepository) InsertToken(ctx context.Context, userId, refreshToken string) (bool, error) {
	if f.InsertTokenFunc == nil {
		return false, fmt.Errorf("fixtures: InsertTokenFunc not set")
	}
	return f.InsertTokenFunc(ctx, userId, refreshToken)
}

func (f *FakeAuthRepository) DeleteToken(ctx context.Context, userId string) error {
	if f.DeleteTokenFunc == nil {
		return fmt.Errorf("fixtures: DeleteTokenFunc not set")
	}
	return f.DeleteTokenFunc(ctx, userId)
}
