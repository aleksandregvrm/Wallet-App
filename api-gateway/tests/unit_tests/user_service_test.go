package unit_tests

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-task-wallet-service/api-gateway/internal/domain"
	infra "go-task-wallet-service/api-gateway/internal/infra/grpc"
	"go-task-wallet-service/api-gateway/internal/services"
	"go-task-wallet-service/api-gateway/tests/fixtures"
	pbAuth "go-task-wallet-service/shared/proto/impl/auth"
)

func newUserService(fake *fixtures.FakeAuthServiceClient) *services.UserService {
	return services.NewUserService(&infra.GrpcHandler{AuthClient: fake})
}

// Mocking user service calls with corresponding results
// Implemented tests ensure the validations present in the services execute as expected and cases where they are not enforced we get pass as usual

func TestUserService_RegisterUser_Success(t *testing.T) {
	fake := &fixtures.FakeAuthServiceClient{
		RegisterUserFunc: func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
			if in.GetUsername() != "janedoe" || in.GetEmail() != "jane@example.com" || in.GetPassword() != "hunter22" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbAuth.RegisterUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Name: "Jane Doe", Username: "janedoe", Email: "jane@example.com", Password: "hunter22"}
	auth, err := svc.RegisterUser(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.ID != "user-1" || auth.AccessToken != "token-abc" {
		t.Fatalf("unexpected auth: %+v", auth)
	}
}

func TestUserService_RegisterUser_EmptyUsername_StillCallsGrpc(t *testing.T) {
	called := false
	fake := &fixtures.FakeAuthServiceClient{
		RegisterUserFunc: func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
			called = true
			return &pbAuth.RegisterUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Username: "", Password: "hunter22"}
	auth, err := svc.RegisterUser(context.Background(), user)
	if err == nil {
		t.Fatalf("expected an error for empty username, got auth: %+v", auth)
	}
	if !called {
		t.Fatalf("expected the gRPC call to have been made before validation ran, but it wasn't")
	}
}

func TestUserService_RegisterUser_EmptyPassword(t *testing.T) {
	fake := &fixtures.FakeAuthServiceClient{
		RegisterUserFunc: func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
			return &pbAuth.RegisterUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Username: "janedoe", Password: ""}
	auth, err := svc.RegisterUser(context.Background(), user)
	if err == nil || auth != nil {
		t.Fatalf("expected an error and nil auth, got auth: %+v, err: %v", auth, err)
	}
}

func TestUserService_RegisterUser_PasswordTooShort(t *testing.T) {
	fake := &fixtures.FakeAuthServiceClient{
		RegisterUserFunc: func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
			return &pbAuth.RegisterUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Username: "janedoe", Password: "abc"}
	auth, err := svc.RegisterUser(context.Background(), user)
	if err == nil || auth != nil {
		t.Fatalf("Expected an error and nil auth, got auth: %+v, err: %v", auth, err)
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Fatalf("expected error to mention password length, got: %v", err)
	}
}

func TestUserService_RegisterUser_GrpcError(t *testing.T) {
	wantErr := errors.New("username taken")
	fake := &fixtures.FakeAuthServiceClient{
		RegisterUserFunc: func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
			return nil, wantErr
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Username: "janedoe", Password: "hunter22"}
	auth, err := svc.RegisterUser(context.Background(), user)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if auth != nil {
		t.Fatalf("expected nil auth on error, got: %+v", auth)
	}
}

func TestUserService_LoginUser_Success(t *testing.T) {
	fake := &fixtures.FakeAuthServiceClient{
		LoginUserFunc: func(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error) {
			if in.GetUsername() != "janedoe" || in.GetPassword() != "hunter22" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbAuth.LoginUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Username: "janedoe", Password: "hunter22"}
	auth, err := svc.LoginUser(context.Background(), user)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if auth.ID != "user-1" || auth.AccessToken != "token-abc" {
		t.Fatalf("Unexpected auth: %+v", auth)
	}
}

func TestUserService_LoginUser_EmptyUsername_DoesNotCallGrpc(t *testing.T) {
	called := false
	fake := &fixtures.FakeAuthServiceClient{
		LoginUserFunc: func(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error) {
			called = true
			return &pbAuth.LoginUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Username: "", Password: "hunter22"}
	auth, err := svc.LoginUser(context.Background(), user)
	if err == nil || auth != nil {
		t.Fatalf("expected an error and nil auth, got auth: %+v, err: %v", auth, err)
	}
	if called {
		t.Fatalf("expected the gRPC call to be skipped when username is empty, but it was called")
	}
}

func TestUserService_LoginUser_PasswordTooShort(t *testing.T) {
	called := false
	fake := &fixtures.FakeAuthServiceClient{
		LoginUserFunc: func(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error) {
			called = true
			return &pbAuth.LoginUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Username: "janedoe", Password: "abc"}
	auth, err := svc.LoginUser(context.Background(), user)
	if err == nil || auth != nil {
		t.Fatalf("expected an error and nil auth, got auth: %+v, err: %v", auth, err)
	}
	if called {
		t.Fatalf("expected the gRPC call to be skipped when password is too short, but it was called")
	}
}

func TestUserService_LoginUser_GrpcError(t *testing.T) {
	wantErr := errors.New("invalid credentials")
	fake := &fixtures.FakeAuthServiceClient{
		LoginUserFunc: func(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error) {
			return nil, wantErr
		},
	}
	svc := newUserService(fake)

	user := &domain.User{Username: "janedoe", Password: "hunter22"}
	auth, err := svc.LoginUser(context.Background(), user)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got: %v", err)
	}
	if auth != nil {
		t.Fatalf("expected nil auth on error, got: %+v", auth)
	}
}

func TestUserService_RefreshToken_AlwaysErrors(t *testing.T) {
	svc := newUserService(&fixtures.FakeAuthServiceClient{})

	token, err := svc.RefreshToken(context.Background(), "some-refresh-token")
	if err == nil {
		t.Fatalf("expected an error from the stub, got token: %q", token)
	}
	if token != "" {
		t.Fatalf("expected empty token on error, got: %q", token)
	}
}
