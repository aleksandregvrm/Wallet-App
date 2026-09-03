package fixtures

import (
	"context"
	"fmt"
	pbAuth "go-task-wallet-service/shared/proto/impl/auth"

	"google.golang.org/grpc"
)

// Registering the three gRPC methods for authentication authorization related calls
// Mocking the api-gateways functionality of gRPC client
type FakeAuthServiceClient struct {
	RegisterUserFunc func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error)
	LoginUserFunc    func(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error)
	RefreshTokenFunc func(ctx context.Context, in *pbAuth.RefreshTokenRequest) (*pbAuth.RefreshTokenResponse, error)
}

var _ pbAuth.AuthServiceClient = (*FakeAuthServiceClient)(nil)

func (f *FakeAuthServiceClient) RegisterUser(ctx context.Context, in *pbAuth.RegisterUserRequest, _ ...grpc.CallOption) (*pbAuth.RegisterUserResponse, error) {
	if f.RegisterUserFunc == nil {
		return nil, fmt.Errorf("fixtures: RegisterUserFunc not set")
	}
	return f.RegisterUserFunc(ctx, in)
}

func (f *FakeAuthServiceClient) LoginUser(ctx context.Context, in *pbAuth.LoginUserRequest, _ ...grpc.CallOption) (*pbAuth.LoginUserResponse, error) {
	if f.LoginUserFunc == nil {
		return nil, fmt.Errorf("fixtures: LoginUserFunc not set")
	}
	return f.LoginUserFunc(ctx, in)
}

func (f *FakeAuthServiceClient) RefreshToken(ctx context.Context, in *pbAuth.RefreshTokenRequest, _ ...grpc.CallOption) (*pbAuth.RefreshTokenResponse, error) {
	if f.RefreshTokenFunc == nil {
		return nil, fmt.Errorf("fixtures: RefreshTokenFunc not set")
	}
	return f.RefreshTokenFunc(ctx, in)
}
