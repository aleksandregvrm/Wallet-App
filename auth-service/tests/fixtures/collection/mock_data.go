package collection

import (
	"go-task-wallet-service/auth-service/internal/domain"
	types "go-task-wallet-service/shared/pkg/models"
	"time"
)

func MockUser() *domain.User {
	return &domain.User{
		ID:        "user-1",
		Name:      "Jane Doe",
		Username:  "janedoe",
		Email:     "jane@example.com",
		Password:  "Hunter22!",
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}
}

func MockUserModel() *types.UserModel {
	return &types.UserModel{
		ID:        "user-1",
		Name:      "Jane Doe",
		Username:  "janedoe",
		Email:     "jane@example.com",
		Password:  "",
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}
}

func MockUserAuthModel() *types.UserAuthModel {
	return &types.UserAuthModel{
		ID:                  "user-auth-1",
		UserId:               "user-1",
		RefreshToken:         "refresh-token-1",
		IsCurrentlyLoggedIn:  false,
		CreatedAt:            time.Unix(0, 0).UTC(),
		UpdatedAt:            time.Unix(0, 0).UTC(),
	}
}
