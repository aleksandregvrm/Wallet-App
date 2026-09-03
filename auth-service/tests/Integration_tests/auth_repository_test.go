package integration_tests

import (
	"context"
	"testing"

	"go-task-wallet-service/auth-service/internal/repository"
	"go-task-wallet-service/auth-service/tests/container"

	"github.com/google/uuid"
)

func TestAuthRepository_InsertUser_And_FindByUserId(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	inserted, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", userId, "hashed-password", "refresh-token-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted.ID != userId || inserted.Name != "Jane Doe" || inserted.Username != "janedoe" || inserted.Email != "jane@example.com" {
		t.Fatalf("unexpected inserted user: %+v", inserted)
	}

	found, err := repo.FindByUserId(ctx, userId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatalf("expected to find the inserted user, got nil")
	}
	if found.ID != userId || found.Username != "janedoe" {
		t.Fatalf("unexpected found user: %+v", found)
	}
}

func TestAuthRepository_FindByUserId_NotFound(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	found, err := repo.FindByUserId(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatalf("expected nil for a non-existent user, got: %+v", found)
	}
}

func TestAuthRepository_FindByUsername(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	if _, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", userId, "hashed-password", "refresh-token-1"); err != nil {
		t.Fatalf("setup: failed to insert user: %v", err)
	}

	found, err := repo.FindByUsername(ctx, "janedoe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatalf("expected to find the inserted user, got nil")
	}
	if found.ID != userId || found.Password != "hashed-password" {
		t.Fatalf("unexpected found user: %+v", found)
	}
}

func TestAuthRepository_FindByUsername_NotFound(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	found, err := repo.FindByUsername(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatalf("expected nil for a non-existent username, got: %+v", found)
	}
}

func TestAuthRepository_InsertToken_And_GetTokenByUserId(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	if _, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", userId, "hashed-password", "refresh-token-1"); err != nil {
		t.Fatalf("setup: failed to insert user: %v", err)
	}

	if _, err := repo.InsertToken(ctx, userId, "refresh-token-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	auth, err := repo.GetTokenByUserId(ctx, userId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil {
		t.Fatalf("expected to find the inserted token, got nil")
	}
	if auth.UserId != userId || auth.RefreshToken != "refresh-token-1" || !auth.IsCurrentlyLoggedIn {
		t.Fatalf("unexpected auth: %+v", auth)
	}
}

func TestAuthRepository_InsertToken_UpsertsOnConflict(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	if _, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", userId, "hashed-password", "refresh-token-1"); err != nil {
		t.Fatalf("setup: failed to insert user: %v", err)
	}

	if inserted, err := repo.InsertToken(ctx, userId, "refresh-token-1"); err != nil || !inserted {
		t.Fatalf("unexpected result on first insert: inserted=%v err=%v", inserted, err)
	}

	// InsertToken guards against clobbering an active session (is_currently_logged_in = false
	// in the WHERE clause), so the conflict path only takes effect after a logout.
	if err := repo.DeleteToken(ctx, userId); err != nil {
		t.Fatalf("setup: failed to log out: %v", err)
	}

	if inserted, err := repo.InsertToken(ctx, userId, "refresh-token-2"); err != nil || !inserted {
		t.Fatalf("unexpected result on second insert: inserted=%v err=%v", inserted, err)
	}

	auth, err := repo.GetTokenByUserId(ctx, userId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.RefreshToken != "refresh-token-2" {
		t.Fatalf("expected the token to be upserted to refresh-token-2, got: %q", auth.RefreshToken)
	}
}

func TestAuthRepository_GetTokenByUserId_NotFound(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	auth, err := repo.GetTokenByUserId(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth != nil {
		t.Fatalf("expected nil for a user with no token, got: %+v", auth)
	}
}

func TestAuthRepository_DeleteToken(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	if _, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", userId, "hashed-password", "refresh-token-1"); err != nil {
		t.Fatalf("setup: failed to insert user: %v", err)
	}
	if _, err := repo.InsertToken(ctx, userId, "refresh-token-1"); err != nil {
		t.Fatalf("setup: failed to insert token: %v", err)
	}

	if err := repo.DeleteToken(ctx, userId); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	auth, err := repo.GetTokenByUserId(ctx, userId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth != nil {
		t.Fatalf("expected token to be deleted, got: %+v", auth)
	}
}

func TestAuthRepository_InsertUser_DuplicateUsername_Errors(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	if _, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", uuid.NewString(), "hashed-password", "refresh-token-1"); err != nil {
		t.Fatalf("setup: failed to insert first user: %v", err)
	}

	_, err := repo.InsertUser(ctx, "Jane Doe 2", "jane2@example.com", "janedoe", uuid.NewString(), "hashed-password", "refresh-token-2")
	if err == nil {
		t.Fatalf("expected a unique-constraint error on duplicate username, got nil")
	}
}
