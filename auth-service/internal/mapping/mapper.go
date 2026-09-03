package mapping

import (
	"go-task-wallet-service/auth-service/internal/domain"
	"go-task-wallet-service/auth-service/internal/pkg"
	pb "go-task-wallet-service/shared/proto/impl/auth"
)

// Mapper functions to map from Proto to Domain and from Domain to Proto
// Essential when using gRPC

func ToRegisterUserDomain(req *pb.RegisterUserRequest) *domain.User {
	if req == nil {
		return nil
	}

	return &domain.User{
		ID:       req.GetId(),
		Name:     req.GetName(),
		Username: req.GetUsername(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}
}

func ToRegisterUserProto(registerUserResponse *pkg.AuthenticateUserResponse) *pb.RegisterUserResponse {
	if registerUserResponse == nil {
		return nil
	}

	return &pb.RegisterUserResponse{
		Id:          registerUserResponse.UserId,
		AccessToken: registerUserResponse.AccessToken,
	}
}

func ToLoginUserDomain(req *pb.LoginUserRequest) *domain.User {
	if req == nil {
		return nil
	}

	return &domain.User{
		Username: req.GetUsername(),
		Password: req.GetPassword(),
	}
}

func ToLoginUserProto(auth *pkg.AuthenticateUserResponse) *pb.LoginUserResponse {
	if auth == nil {
		return nil
	}

	return &pb.LoginUserResponse{
		Id:          auth.UserId,
		AccessToken: auth.AccessToken,
	}
}

func ToRefreshTokenDomain(req *pb.RefreshTokenRequest) *domain.UserAuth {
	if req == nil {
		return nil
	}

	return &domain.UserAuth{
		AccessToken: req.GetRefreshToken(),
		UserId:      req.GetUserId(),
	}
}

func ToRefreshTokenProto(auth *domain.UserAuth) *pb.RefreshTokenResponse {
	if auth == nil {
		return nil
	}

	return &pb.RefreshTokenResponse{
		RefreshToken: auth.AccessToken,
		UserId:       auth.UserId,
	}
}
