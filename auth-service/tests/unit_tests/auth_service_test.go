package unit_tests

import (
	"context"
	"errors"
	"testing"

	"go-task-wallet-service/auth-service/internal/domain"
	"go-task-wallet-service/auth-service/internal/services"
	"go-task-wallet-service/auth-service/internal/utils"
	"go-task-wallet-service/auth-service/tests/fixtures"
	types "go-task-wallet-service/shared/pkg/models"
)

func validRegisterUser() *domain.User {
	return &domain.User{
		Name:     "Jane Doe",
		Username: "janedoe",
		Email:    "jane@example.com",
		Password: "Hunter22!",
	}
}

func TestAuthService_RegisterUser_Success(t *testing.T) {
	var insertCalled bool
	repo := &fixtures.FakeAuthRepository{
		FindByUserIdFunc: func(ctx context.Context, userId string) (*types.UserModel, error) {
			return nil, nil
		},
		InsertUserFunc: func(ctx context.Context, name, email, username, userId, password, refreshToken string) (*types.UserModel, error) {
			insertCalled = true
			if name != "Jane Doe" || email != "jane@example.com" || username != "janedoe" {
				t.Fatalf("unexpected insert args: name=%q email=%q username=%q", name, email, username)
			}
			if password == "Hunter22!" {
				t.Fatalf("expected the password to be hashed before InsertUser, got plaintext")
			}
			return &types.UserModel{ID: userId, Name: name, Email: email, Username: username}, nil
		},
	}
	svc := services.NewAuthService(repo)

	resp, err := svc.RegisterUser(context.Background(), validRegisterUser())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !insertCalled {
		t.Fatalf("expected InsertUser to be called")
	}
	if resp.AccessToken == "" {
		t.Fatalf("expected a non-empty access token")
	}
	if resp.UserId == "" {
		t.Fatalf("expected a non-empty user id")
	}
}

func TestAuthService_RegisterUser_ValidationFailure_RepositoryNeverCalled(t *testing.T) {
	called := false
	repo := &fixtures.FakeAuthRepository{
		FindByUserIdFunc: func(ctx context.Context, userId string) (*types.UserModel, error) {
			called = true
			return nil, nil
		},
	}
	svc := services.NewAuthService(repo)

	user := validRegisterUser()
	user.Password = "weak"
	_, err := svc.RegisterUser(context.Background(), user)
	if err == nil {
		t.Fatalf("expected a validation error")
	}
	if called {
		t.Fatalf("expected the repository never to be called on validation failure")
	}
}

func TestAuthService_RegisterUser_FindByUserIdError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	repo := &fixtures.FakeAuthRepository{
		FindByUserIdFunc: func(ctx context.Context, userId string) (*types.UserModel, error) {
			return nil, wantErr
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.RegisterUser(context.Background(), validRegisterUser())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}

func TestAuthService_RegisterUser_UserAlreadyExists(t *testing.T) {
	insertCalled := false
	repo := &fixtures.FakeAuthRepository{
		FindByUserIdFunc: func(ctx context.Context, userId string) (*types.UserModel, error) {
			return &types.UserModel{ID: userId}, nil
		},
		InsertUserFunc: func(ctx context.Context, name, email, username, userId, password, refreshToken string) (*types.UserModel, error) {
			insertCalled = true
			return nil, nil
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.RegisterUser(context.Background(), validRegisterUser())
	if err == nil {
		t.Fatalf("expected an error when the user already exists")
	}
	if insertCalled {
		t.Fatalf("expected InsertUser never to be called when the user already exists")
	}
}

func TestAuthService_RegisterUser_InsertUserError(t *testing.T) {
	wantErr := errors.New("unique constraint violation")
	repo := &fixtures.FakeAuthRepository{
		FindByUserIdFunc: func(ctx context.Context, userId string) (*types.UserModel, error) {
			return nil, nil
		},
		InsertUserFunc: func(ctx context.Context, name, email, username, userId, password, refreshToken string) (*types.UserModel, error) {
			return nil, wantErr
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.RegisterUser(context.Background(), validRegisterUser())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}

func mockUserWithPassword(t *testing.T, plaintext string) *types.UserModel {
	t.Helper()
	hashed, err := utils.HashPassword(plaintext)
	if err != nil {
		t.Fatalf("test setup: failed to hash password: %v", err)
	}
	return &types.UserModel{ID: "user-1", Username: "janedoe", Email: "jane@example.com", Password: hashed}
}

func TestAuthService_LoginUser_Success(t *testing.T) {
	user := mockUserWithPassword(t, "Hunter22!")
	insertTokenCalled := false
	repo := &fixtures.FakeAuthRepository{
		FindByUsernameFunc: func(ctx context.Context, username string) (*types.UserModel, error) {
			return user, nil
		},
		GetTokenByUserIdFunc: func(ctx context.Context, userId string) (*types.UserAuthModel, error) {
			return nil, nil
		},
		InsertTokenFunc: func(ctx context.Context, userId, refreshToken string) (bool, error) {
			insertTokenCalled = true
			if userId != "user-1" {
				t.Fatalf("unexpected userId: %q", userId)
			}
			return true, nil
		},
	}
	svc := services.NewAuthService(repo)

	resp, err := svc.LoginUser(context.Background(), "janedoe", "Hunter22!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !insertTokenCalled {
		t.Fatalf("expected InsertToken to be called")
	}
	if resp.AccessToken == "" || resp.UserId != "user-1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAuthService_LoginUser_FindByUsernameError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	repo := &fixtures.FakeAuthRepository{
		FindByUsernameFunc: func(ctx context.Context, username string) (*types.UserModel, error) {
			return nil, wantErr
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.LoginUser(context.Background(), "janedoe", "Hunter22!")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}

func TestAuthService_LoginUser_UserNotFound(t *testing.T) {
	repo := &fixtures.FakeAuthRepository{
		FindByUsernameFunc: func(ctx context.Context, username string) (*types.UserModel, error) {
			return nil, nil
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.LoginUser(context.Background(), "janedoe", "Hunter22!")
	if err == nil {
		t.Fatalf("expected an error for a nonexistent user")
	}
}

func TestAuthService_LoginUser_WrongPassword(t *testing.T) {
	user := mockUserWithPassword(t, "Hunter22!")
	repo := &fixtures.FakeAuthRepository{
		FindByUsernameFunc: func(ctx context.Context, username string) (*types.UserModel, error) {
			return user, nil
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.LoginUser(context.Background(), "janedoe", "WrongPassword1!")
	if err == nil {
		t.Fatalf("expected an error for a wrong password")
	}
}

func TestAuthService_LoginUser_GetTokenByUserIdError(t *testing.T) {
	user := mockUserWithPassword(t, "Hunter22!")
	wantErr := errors.New("cache unavailable")
	repo := &fixtures.FakeAuthRepository{
		FindByUsernameFunc: func(ctx context.Context, username string) (*types.UserModel, error) {
			return user, nil
		},
		GetTokenByUserIdFunc: func(ctx context.Context, userId string) (*types.UserAuthModel, error) {
			return nil, wantErr
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.LoginUser(context.Background(), "janedoe", "Hunter22!")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}

func TestAuthService_LoginUser_AlreadyLoggedIn(t *testing.T) {
	user := mockUserWithPassword(t, "Hunter22!")
	insertTokenCalled := false
	repo := &fixtures.FakeAuthRepository{
		FindByUsernameFunc: func(ctx context.Context, username string) (*types.UserModel, error) {
			return user, nil
		},
		GetTokenByUserIdFunc: func(ctx context.Context, userId string) (*types.UserAuthModel, error) {
			return &types.UserAuthModel{UserId: userId, IsCurrentlyLoggedIn: true}, nil
		},
		InsertTokenFunc: func(ctx context.Context, userId, refreshToken string) (bool, error) {
			insertTokenCalled = true
			return true, nil
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.LoginUser(context.Background(), "janedoe", "Hunter22!")
	if err == nil {
		t.Fatalf("expected an error when the user is already logged in")
	}
	if insertTokenCalled {
		t.Fatalf("expected InsertToken never to be called when already logged in")
	}
}

func TestAuthService_LoginUser_PreviouslyLoggedOut_CanLoginAgain(t *testing.T) {
	user := mockUserWithPassword(t, "Hunter22!")
	repo := &fixtures.FakeAuthRepository{
		FindByUsernameFunc: func(ctx context.Context, username string) (*types.UserModel, error) {
			return user, nil
		},
		GetTokenByUserIdFunc: func(ctx context.Context, userId string) (*types.UserAuthModel, error) {
			return &types.UserAuthModel{UserId: userId, IsCurrentlyLoggedIn: false}, nil
		},
		InsertTokenFunc: func(ctx context.Context, userId, refreshToken string) (bool, error) {
			return true, nil
		},
	}
	svc := services.NewAuthService(repo)

	resp, err := svc.LoginUser(context.Background(), "janedoe", "Hunter22!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserId != "user-1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAuthService_LoginUser_InsertTokenError(t *testing.T) {
	user := mockUserWithPassword(t, "Hunter22!")
	wantErr := errors.New("cache write failed")
	repo := &fixtures.FakeAuthRepository{
		FindByUsernameFunc: func(ctx context.Context, username string) (*types.UserModel, error) {
			return user, nil
		},
		GetTokenByUserIdFunc: func(ctx context.Context, userId string) (*types.UserAuthModel, error) {
			return nil, nil
		},
		InsertTokenFunc: func(ctx context.Context, userId, refreshToken string) (bool, error) {
			return true, wantErr
		},
	}
	svc := services.NewAuthService(repo)

	_, err := svc.LoginUser(context.Background(), "janedoe", "Hunter22!")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
}

func TestAuthService_AuthorizeUser_AlwaysErrors(t *testing.T) {
	svc := services.NewAuthService(&fixtures.FakeAuthRepository{})

	if err := svc.AuthorizeUser(context.Background(), "user-1"); err == nil {
		t.Fatalf("expected an error from the stub")
	}
}

func TestAuthService_RefreshToken_AlwaysEmpty(t *testing.T) {
	svc := services.NewAuthService(&fixtures.FakeAuthRepository{})

	if got := svc.RefreshToken(context.Background(), "user-1"); got != "" {
		t.Fatalf("expected an empty string from the stub, got: %q", got)
	}
}
