package services

import (
	"context"
	"errors"
	"fmt"
	"go-task-wallet-service/auth-service/internal/domain"
	"go-task-wallet-service/auth-service/internal/pkg"
	"go-task-wallet-service/auth-service/internal/utils"
	"go-task-wallet-service/shared/logging"
	"go-task-wallet-service/shared/pkg/session"

	"github.com/google/uuid"
)

type AuthService struct {
	authRepository domain.AuthRepository
	logging.Logger
}

func NewAuthService(authRepsotiory domain.AuthRepository) *AuthService {
	return &AuthService{
		authRepository: authRepsotiory,
		Logger:         logging.NewInternalLogger(),
	}
}

func (s *AuthService) generateSessionTokens(ctx context.Context, user *domain.User) (*pkg.UserSessionTokens, error) {
	accessToken, err := session.GenerateAccessToken(user)

	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := session.GenerateRefreshToken(user)

	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &pkg.UserSessionTokens{
		SessionToken: accessToken,
		RefreshToken: refreshToken,
	}, nil

}

func (s *AuthService) RegisterUser(ctx context.Context, user *domain.User) (*pkg.AuthenticateUserResponse, error) {
	ctx = logging.WithRequestID(ctx, user.Username)

	if err := utils.ValidateUser(user); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Generating the user Id manually with standard UUID library
	user.ID = uuid.NewString()

	existing, err := s.authRepository.FindByUserId(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existing != nil {
		return nil, errors.New("user already exists")
	}

	hashedPassword, err := utils.HashPassword(user.Password)

	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = hashedPassword

	tokens, err := s.generateSessionTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session tokens: %w", err)
	}

	// Persisting User in the database and caching user refresh token for session management in RAM
	inserted, err := s.authRepository.InsertUser(ctx, user.Name, user.Email, user.Username, user.ID, user.Password, tokens.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	s.LogInfo(ctx, "User created with userId: %s", user.ID)

	return &pkg.AuthenticateUserResponse{
		AccessToken: tokens.SessionToken,
		UserId:      inserted.ID,
	}, nil
}

func (s *AuthService) LoginUser(ctx context.Context, username, password string) (*pkg.AuthenticateUserResponse, error) {
	ctx = logging.WithRequestID(ctx, username)

	user, err := s.authRepository.FindByUsername(ctx, username)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid username or password")
	}
	if err = utils.ComparePassword(user.Password, password); err != nil {
		return nil, fmt.Errorf("invalid password for username: %s", username)
	}

	existingAuth, err := s.authRepository.GetTokenByUserId(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}
	if existingAuth != nil && existingAuth.IsCurrentlyLoggedIn {
		return nil, fmt.Errorf("user %s is already logged in", username)
	}

	domainUser := domain.User{
		Username: user.Username,
		ID:       user.ID,
		Email:    user.Email,
	}

	tokens, err := s.generateSessionTokens(ctx, &domainUser)

	if err != nil {
		return nil, fmt.Errorf("failed to generate session tokens: %w", err)

	}

	inserted, err := s.authRepository.InsertToken(ctx, domainUser.ID, tokens.RefreshToken)
	if err != nil {
		s.LogWarn(ctx, "Token inserting (session initialization) failed for - %s, reason: %v", username, err)
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}
	if !inserted {
		return nil, fmt.Errorf("user %s is already logged in", username)
	}

	s.LogInfo(ctx, "User logged in with username: %s", username)

	return &pkg.AuthenticateUserResponse{
		AccessToken: tokens.SessionToken,
		UserId:      domainUser.ID,
	}, nil

}

func (s *AuthService) AuthorizeUser(ctx context.Context, userId string) error {
	return fmt.Errorf("something has gone wrong")
}

func (s *AuthService) RefreshToken(ctx context.Context, userId string) string {
	return ""
}
