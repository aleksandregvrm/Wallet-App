package infra

import (
	"go-task-wallet-service/shared/env"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbAuth "go-task-wallet-service/shared/proto/impl/auth"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"
)

var (
	authServiceAddr   = env.GetString("AUTH_SERVICE_ADDR", "localhost:8081")
	walletServiceAddr = env.GetString("WALLET_SERVICE_ADDR", "localhost:8082")
)

type GrpcHandler struct {
	AuthClient   pbAuth.AuthServiceClient
	WalletClient pbWallet.WalletServiceClient

	AuthConn   *grpc.ClientConn
	WalletConn *grpc.ClientConn
}

// gRPC module declaration
func NewGrpcHandler() (*GrpcHandler, error) {
	authConn, err := grpc.NewClient(authServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	walletConn, err := grpc.NewClient(walletServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	// injectable gRPC client implementations
	authClient := pbAuth.NewAuthServiceClient(authConn)
	walletClient := pbWallet.NewWalletServiceClient(walletConn)

	return &GrpcHandler{
		AuthClient:   authClient,
		WalletClient: walletClient,

		AuthConn:   authConn,
		WalletConn: walletConn,
	}, nil
}

func (h *GrpcHandler) Close() {
	h.AuthConn.Close()
	h.WalletConn.Close()
}
