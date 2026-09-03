package session

import (
	"fmt"
	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/pkg/domain"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret         = env.GetString("JWT_SECRET", "dev-secret-change-me")
	refreshTokenBytes = env.GetInt("JWT_BYTES", 32)
	accessTokenTTL    = time.Duration(env.GetInt("ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	refreshTokenTTL   = time.Duration(env.GetInt("REFRESH_TOKEN_TTL_MINUTES", 48)) * time.Hour
)

type AccessTokenClaims struct {
	UserId   string
	Username string
	Email    string
	jwt.RegisteredClaims
}

type refreshTokenClaims struct {
	UserId   string
	Username string
	Email    string
	jwt.RegisteredClaims
}

// Access token provided to the actual user, Used for authorization
func GenerateAccessToken(user *domain.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("cannot generate an access token for a nil user")
	}

	now := time.Now()
	claims := AccessTokenClaims{
		UserId:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signed, nil
}

func ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}

	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate access token: %w", err)
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("access token is invalid")
	}

	return claims, nil
}

// Actual persistent token with Much longer TTL, Used to refresh the session of the user
func GenerateRefreshToken(user *domain.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("cannot generate a refresh token for a nil user")
	}

	now := time.Now()
	claims := refreshTokenClaims{
		UserId:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signed, nil
}

func ValidateRefreshToken(tokenString string) (*refreshTokenClaims, error) {
	claims := &refreshTokenClaims{}

	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("refresh token is invalid")
	}

	return claims, nil
}
