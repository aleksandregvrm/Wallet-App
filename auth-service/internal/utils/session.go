package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"go-task-wallet-service/auth-service/internal/domain"
	"go-task-wallet-service/shared/env"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret         = env.GetString("JWT_SECRET", "dev-secret-change-me")
	refreshTokenBytes = env.GetInt("JWT_BYTES", 64)
	accessTokenTTL    = time.Duration(env.GetInt("ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	refreshTokenTTL   = time.Duration(env.GetInt("REFRESH_TOKEN_TTL_HOURS", 48)) * time.Hour // Refresh token set to have expiration time of hours
)

type accessTokenClaims struct {
	UserId   string
	Username string
	Email    string
	jwt.RegisteredClaims
}

type refreshTokenClaims struct {
	UserId   string
	Username string
	Email    string
}

func GenerateAccessToken(user *domain.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("cannot generate an access token for a nil user")
	}

	now := time.Now()
	claims := accessTokenClaims{
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

func ValidateAccessToken(tokenString string) (*accessTokenClaims, error) {
	claims := &accessTokenClaims{}

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

func GenerateRefreshToken() (string, error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
