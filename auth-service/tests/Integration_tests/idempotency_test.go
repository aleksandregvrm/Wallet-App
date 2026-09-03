package integration_tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"go-task-wallet-service/auth-service/internal/repository"
	"go-task-wallet-service/auth-service/internal/services"
	"go-task-wallet-service/auth-service/internal/utils"
	"go-task-wallet-service/auth-service/tests/container"

	"github.com/google/uuid"
)

func TestAuthRepository_InsertUser_ConcurrentSameUsername_OnlyOneSucceeds(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	const attempts = 10
	var succeeded, failed int64
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", uuid.NewString(), "hashed-password", "refresh-token")
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			atomic.AddInt64(&succeeded, 1)
		}(i)
	}
	wg.Wait()

	if succeeded != 1 {
		t.Fatalf("expected exactly 1 successful insert for a racing username, got %d succeeded, %d failed", succeeded, failed)
	}
	if failed != attempts-1 {
		t.Fatalf("expected %d failed inserts, got %d", attempts-1, failed)
	}
}

func TestAuthRepository_InsertToken_ConcurrentUpsert_AllSucceed(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	if _, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", userId, "hashed-password", "refresh-token-0"); err != nil {
		t.Fatalf("setup: failed to insert user: %v", err)
	}

	const attempts = 10
	tokens := make([]string, attempts)
	var wg sync.WaitGroup
	var errCount int64
	for i := 0; i < attempts; i++ {
		tokens[i] = uuid.NewString()
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			if _, err := repo.InsertToken(ctx, userId, token); err != nil {
				atomic.AddInt64(&errCount, 1)
			}
		}(tokens[i])
	}
	wg.Wait()

	if errCount != 0 {
		t.Fatalf("expected all concurrent upserts to succeed (ON CONFLICT DO UPDATE), got %d errors", errCount)
	}

	auth, err := repo.GetTokenByUserId(ctx, userId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil {
		t.Fatalf("expected a token row to exist after concurrent upserts")
	}
	found := false
	for _, token := range tokens {
		if auth.RefreshToken == token {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected the surviving token to be one of the racing writes, got: %q", auth.RefreshToken)
	}
}

func TestAuthService_LoginUser_ConcurrentSameUser_DoubleLoginGuardUnderRace(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewAuthRepository(env.DB, env.Cache)
	svc := services.NewAuthService(repo)
	ctx := context.Background()

	hashed, err := utils.HashPassword("Hunter22!")
	if err != nil {
		t.Fatalf("setup: failed to hash password: %v", err)
	}
	if _, err := repo.InsertUser(ctx, "Jane Doe", "jane@example.com", "janedoe", uuid.NewString(), hashed, "refresh-token-0"); err != nil {
		t.Fatalf("setup: failed to insert user: %v", err)
	}

	const attempts = 10
	var succeeded, failed int64
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.LoginUser(ctx, "janedoe", "Hunter22!")
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			atomic.AddInt64(&succeeded, 1)
		}()
	}
	wg.Wait()

	t.Logf("concurrent logins: succeeded=%d failed=%d (out of %d)", succeeded, failed, attempts)

	if succeeded > 1 {
		t.Fatalf("the \"already logged in\" guard is a read-then-write race with no locking or unique constraint: "+
			"%d/%d concurrent logins for the same user all succeeded, when at most 1 should have (TOCTOU race between GetTokenByUserId and InsertToken)", succeeded, attempts)
	}
}
