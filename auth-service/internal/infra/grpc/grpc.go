package grpc

import (
	"context"
	"fmt"
	"go-task-wallet-service/auth-service/internal/domain"
	"go-task-wallet-service/auth-service/internal/mapping"
	"go-task-wallet-service/shared/logging"
	pb "go-task-wallet-service/shared/proto/impl/auth"

	"google.golang.org/grpc"
)

type gRPCHandler struct {
	pb.UnimplementedAuthServiceServer
	Service domain.AuthService
	logging.Logger
}

// gRPC module declaration
// Wiring connection to other service gRPC clients. In this case auth-service only communicates front to back to the api-gateway service. So we only need to establish a connection to the api-gateway service.
func NewGrpcHandler(s *grpc.Server, service domain.AuthService) {
	handler := &gRPCHandler{
		Service: service,
		Logger:  logging.NewInternalLogger(),
	}
	pb.RegisterAuthServiceServer(s, handler)
}

func (h *gRPCHandler) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {

	h.LogInfo(ctx, "%v, request has arrived in the auth-service register user handler", req)

	domainUser := mapping.ToRegisterUserDomain(req)

	registeredUserResponse, err := h.Service.RegisterUser(ctx, domainUser)

	if err != nil {
		return nil, err
	}

	protoRegisteredUserResponse := mapping.ToRegisterUserProto(registeredUserResponse)

	return protoRegisteredUserResponse, nil
}

func (h *gRPCHandler) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	h.LogInfo(ctx, "%v, request has arrived in the auth-service login user handler", req)

	domainUser := mapping.ToLoginUserDomain(req)

	loggedInUserResponse, err := h.Service.LoginUser(ctx, domainUser.Username, domainUser.Password)

	if err != nil {
		return nil, err
	}

	protoLoginUserResponse := mapping.ToLoginUserProto(loggedInUserResponse)

	return protoLoginUserResponse, nil
}

func (h *gRPCHandler) RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return nil, fmt.Errorf("something has gone wrong")
}
