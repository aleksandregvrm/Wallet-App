package main

import (
	"context"
	infra "go-task-wallet-service/auth-service/internal/infra/grpc"
	"go-task-wallet-service/auth-service/internal/repository"
	services "go-task-wallet-service/auth-service/internal/services"
	"go-task-wallet-service/shared/cache"
	"go-task-wallet-service/shared/db"
	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/logging"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var (
	gRPCAddr    = env.GetString("GRPC_ADDR_AUTH", ":8081")
	environment = env.GetString("ENV", "development")
	serviceName = "Auth Service"
)

const serverShutdownTimeout int = 30

func main() {
	internalLogger := logging.NewInternalLogger()

	// Cancel ctx on SIGINT/SIGTERM so the graceful shutdown path actually runs
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	lis, err := net.Listen("tcp", gRPCAddr)
	if err != nil {
		internalLogger.LogFatal(ctx, "An error has occurred when starting a Auth Service gRPC server: %v", err)
	}

	dbInstance := db.NewPostgreSqlInstance()

	if err := dbInstance.Connect(serviceName); err != nil {
		internalLogger.LogFatal(ctx, "An error has occurred when connecting to postgres: %v", err)
	}
	defer dbInstance.Close()

	// Defining the Database in use since newDBClient also extends to using other SQL databases
	postgreSqlDbClient, err := db.NewDBClient(dbInstance)
	if err != nil {
		internalLogger.LogFatal(ctx, "An error has occurred when building the postgres client: %v", err)
	}

	// Initializing the redis instance, then building the cache client from it
	// (mirrors the Postgres block above: build from the Instance interface,
	// not a concrete Redis setup).
	redisInstance := cache.NewRedisInstance()

	if err := redisInstance.Connect(serviceName); err != nil {
		internalLogger.LogFatal(ctx, "An error has occurred when connecting to redis: %v", err)
	}
	defer redisInstance.Close()

	cacheClient, err := cache.NewCacheClient(redisInstance)
	if err != nil {
		internalLogger.LogFatal(ctx, "An error has occurred when building the redis cache client: %v", err)
	}

	authRepository := repository.NewAuthRepository(postgreSqlDbClient, cacheClient)

	// Service declaration
	service := services.NewAuthService(authRepository)

	// Launching the Grpc server with the Driver service as a dependency
	grpcServer := grpcserver.NewServer()

	infra.NewGrpcHandler(grpcServer, service)

	// Registering the gRPC health checks
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Initializing gRPC ui for the ease of testing in development
	// In Production environment we obviously won't have such UI
	if environment == "development" {
		reflection.Register(grpcServer)
	}
	internalLogger.LogInfo(ctx, "Starting gRPC server Auth Service on port: %s", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			internalLogger.LogError(ctx, "failed to server: %v", err)
		}
	}()

	// Wait for the shutdown signal
	<-ctx.Done()
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// Ensuring shutting down cannot hang in case we have a stuck call. In case timeout is exceeded the server is shutdown anyway.
	select {
	case <-stopped:
	case <-time.After(time.Duration(serverShutdownTimeout) * time.Second):
		internalLogger.LogWarn(ctx, "graceful stop exceeded %ds, forcing shutdown", serverShutdownTimeout)
		grpcServer.Stop()
	}

}
