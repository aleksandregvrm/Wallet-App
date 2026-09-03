package fixtures

import (
	"context"
	pbAuth "go-task-wallet-service/shared/proto/impl/auth"
)

type FakeAuthServiceServer struct {
	pbAuth.UnimplementedAuthServiceServer
	RegisterUserFunc func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error)
	LoginUserFunc    func(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error)
	RefreshTokenFunc func(ctx context.Context, in *pbAuth.RefreshTokenRequest) (*pbAuth.RefreshTokenResponse, error)
}

var _ pbAuth.AuthServiceServer = (*FakeAuthServiceServer)(nil)

func (f *FakeAuthServiceServer) RegisterUser(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
	if f.RegisterUserFunc == nil {
		return f.UnimplementedAuthServiceServer.RegisterUser(ctx, in)
	}
	return f.RegisterUserFunc(ctx, in)
}

func (f *FakeAuthServiceServer) LoginUser(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error) {
	if f.LoginUserFunc == nil {
		return f.UnimplementedAuthServiceServer.LoginUser(ctx, in)
	}
	return f.LoginUserFunc(ctx, in)
}

func (f *FakeAuthServiceServer) RefreshToken(ctx context.Context, in *pbAuth.RefreshTokenRequest) (*pbAuth.RefreshTokenResponse, error) {
	if f.RefreshTokenFunc == nil {
		return f.UnimplementedAuthServiceServer.RefreshToken(ctx, in)
	}
	return f.RefreshTokenFunc(ctx, in)
}
