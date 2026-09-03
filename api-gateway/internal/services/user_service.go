package services

import (
	"context"
	"fmt"
	"go-task-wallet-service/api-gateway/internal/domain"
	infra "go-task-wallet-service/api-gateway/internal/infra/grpc"
	"go-task-wallet-service/api-gateway/internal/mapping"
	"go-task-wallet-service/shared/logging"
)

type UserService struct {
	grpc *infra.GrpcHandler
	logging.Logger
}

func NewUserService(grpc *infra.GrpcHandler) *UserService {
	return &UserService{grpc: grpc, Logger: logging.NewInternalLogger()}
}

func (s *UserService) RegisterUser(ctx context.Context, user *domain.User) (*domain.UserAuth, error) {
	pbRegisterUser := mapping.ToRegisterUserProto(user)

	registeredUser, err := s.grpc.AuthClient.RegisterUser(ctx, pbRegisterUser)

	if user.Username == "" || user.Password == "" {
		return nil, fmt.Errorf("Invalid credentials")
	}

	if len(user.Password) < 6 {
		return nil, fmt.Errorf("Provided password is too short - %s", len(user.Password))
	}

	if err != nil {
		s.LogInfo(ctx, "Registering user has failed, with username:%s", user.Username)
		return nil, err
	}

	authUser := mapping.ToRegisterUserAuthDomain(registeredUser)

	return authUser, nil
}

func (s *UserService) LoginUser(ctx context.Context, user *domain.User) (*domain.UserAuth, error) {
	username, password := user.Username, user.Password

	pbLoginUser := mapping.ToLoginUserProto(user)

	// Checks to prevent useless rpc calls and cache db lookups
	if username == "" || password == "" {
		return nil, fmt.Errorf("Invalid credentials")
	}

	if len(password) < 6 {
		return nil, fmt.Errorf("Invalid Credentials")
	}

	loggedInUserResponse, err := s.grpc.AuthClient.LoginUser(ctx, pbLoginUser)

	if err != nil {
		s.LogInfo(ctx, "Authenticating user has failed, with username:%s, reason:%v", user.Username, err)
		return nil, err
	}

	authUser := mapping.ToLoginUserDomain(loggedInUserResponse)

	return authUser, nil
}

func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	return "", fmt.Errorf("Something has gone terribly wrong")
}
