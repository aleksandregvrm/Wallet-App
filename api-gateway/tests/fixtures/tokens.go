package fixtures

import (
	sharedDomain "go-task-wallet-service/shared/pkg/domain"
	"go-task-wallet-service/shared/pkg/session"
	"testing"
)

// Test to validate the access token creation, - validation and invalidation
func ValidAccessToken(t *testing.T, userId, username, email string) string {
	t.Helper()

	token, err := session.GenerateAccessToken(&sharedDomain.User{
		ID:       userId,
		Username: username,
		Email:    email,
	})
	if err != nil {
		t.Fatalf("fixtures: failed to generate access token: %v", err)
	}
	return token
}

const InvalidAccessToken = "not-a-real-jwt-token"
