package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-task-wallet-service/api-gateway/controllers"
	infra "go-task-wallet-service/api-gateway/internal/infra/grpc"
	"go-task-wallet-service/api-gateway/internal/middlewares"
	"go-task-wallet-service/api-gateway/internal/services"
	"go-task-wallet-service/api-gateway/tests/fixtures"
	pbAuth "go-task-wallet-service/shared/proto/impl/auth"
	pbWallet "go-task-wallet-service/shared/proto/impl/wallet"

	"google.golang.org/grpc"
)

func newTestServer(t *testing.T, authSrv pbAuth.AuthServiceServer, walletSrv pbWallet.WalletServiceServer) *httptest.Server {
	t.Helper()

	authConn := fixtures.DialBufconn(t, func(s *grpc.Server) {
		pbAuth.RegisterAuthServiceServer(s, authSrv)
	})
	walletConn := fixtures.DialBufconn(t, func(s *grpc.Server) {
		pbWallet.RegisterWalletServiceServer(s, walletSrv)
	})

	grpcHandler := &infra.GrpcHandler{
		AuthClient:   pbAuth.NewAuthServiceClient(authConn),
		WalletClient: pbWallet.NewWalletServiceClient(walletConn),
	}

	userSvc := services.NewUserService(grpcHandler)
	accountSvc := services.NewAccountService(grpcHandler)
	walletSvc := services.NewWalletService(grpcHandler)

	userCtrl := controllers.NewUserController(userSvc)
	accountCtrl := controllers.NewAccountController(accountSvc)
	walletCtrl := controllers.NewWalletController(walletSvc)
	configCtrl := controllers.NewConfigController()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /user/register", userCtrl.HandleUserRegister)
	mux.HandleFunc("POST /user/login", userCtrl.HandleUserLogin)
	mux.HandleFunc("POST /user/refresh", userCtrl.HandleRefreshToken)
	mux.HandleFunc("POST /account/create", middlewares.AuthorizeUser(accountCtrl.HandleAccountCreate))
	mux.HandleFunc("UPDATE /account/update", middlewares.AuthorizeUser(accountCtrl.HandleAccountUpdate))
	mux.HandleFunc("POST /wallet/deposit", middlewares.AuthorizeUser(walletCtrl.HandleDeposit))
	mux.HandleFunc("POST /wallet/withdraw", middlewares.AuthorizeUser(walletCtrl.HandleWithdraw))
	mux.HandleFunc("POST /wallet/transfer", middlewares.AuthorizeUser(walletCtrl.HandleTransferFunds))
	mux.HandleFunc("GET /wallet/transactions", middlewares.AuthorizeUser(walletCtrl.HandleListTransactions))
	mux.HandleFunc("GET /wallet/balance", middlewares.AuthorizeUser(walletCtrl.HandleGetBalance))
	mux.HandleFunc("GET /health/liveness", configCtrl.HandleHealthLiveness)
	mux.HandleFunc("GET /health/readiness", configCtrl.HandleHealthReadiness)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}
