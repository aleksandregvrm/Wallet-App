package main

import (
	"context"
	"go-task-wallet-service/api-gateway/controllers"
	infra "go-task-wallet-service/api-gateway/internal/infra/grpc"
	"go-task-wallet-service/api-gateway/internal/middlewares"
	"go-task-wallet-service/api-gateway/internal/services"
	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/logging"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8080")
)

func main() {
	internalLogger := logging.NewInternalLogger()

	ctx, cancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	defer cancel()

	grpcHandler, err := infra.NewGrpcHandler()
	if err != nil {
		internalLogger.LogFatal(ctx, "failed to set up grpc connections: %v", err)
	}
	defer grpcHandler.Close()

	// Declaring services with injected gRPC module
	userSvc := services.NewUserService(grpcHandler)
	accountSvc := services.NewAccountService(grpcHandler)
	walletSvc := services.NewWalletService(grpcHandler)

	// Declaring controllers with injected services
	userCtrl := controllers.NewUserController(userSvc)
	accountCtrl := controllers.NewAccountController(accountSvc)
	configCtrl := controllers.NewConfigController()
	walletCtrl := controllers.NewWalletController(walletSvc)

	// Declared Routes
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

	// Config routes
	mux.HandleFunc("GET /health/liveness", configCtrl.HandleHealthLiveness)
	mux.HandleFunc("GET /health/readiness", configCtrl.HandleHealthReadiness)

	serverErrors := make(chan error, 1)

	go func() {
		internalLogger.LogInfo(ctx, "Http Server listening on port: %s", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	// Specifying the signals to listen for. To trigger graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		internalLogger.LogError(ctx, "Error starting the Http server: %v", err)
	case sig := <-shutdown:
		internalLogger.LogInfo(ctx, "Http Server has been shutdown, due to: %v", sig)

		// Exponential backoff of 10 seconds implemented to make sure there are not handing connections when the server is shutting down
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			internalLogger.LogError(shutdownCtx, "There has been an error with shutting down the Http server gracefully. Reason: %v", err)
			server.Close()
		}
	}
}
