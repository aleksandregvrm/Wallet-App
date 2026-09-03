package main

import (
	"context"
	"go-task-wallet-service/shared/cache"
	"go-task-wallet-service/shared/db"
	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/events"
	"go-task-wallet-service/shared/logging"
	infra "go-task-wallet-service/wallet-service/internal/infra/grpc"
	kafkainfra "go-task-wallet-service/wallet-service/internal/infra/kafka"
	"go-task-wallet-service/wallet-service/internal/repository"
	"go-task-wallet-service/wallet-service/internal/services"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Outbox configurations
const outboxBatchSize int = 100
const outboxPollInterval int = 2

const serverShutdownTimeout int = 30

var (
	gRPCAddr           = env.GetString("GRPC_ADDR_WALLET", ":8082")
	environment        = env.GetString("ENV", "development")
	serviceName        = "Wallet Service"
	kafkaBrokers       = strings.Split(env.GetString("KAFKA_BROKERS", "localhost:9092"), ",")
	kafkaConsumerGroup = env.GetString("KAFKA_CONSUMER_GROUP", "wallet-service")
	walletEventsTopic  = env.GetString("KAFKA_WALLET_EVENTS_TOPIC", "wallet.events.v1")
)

func main() {
	// Logging declaration
	internalLogger := logging.NewInternalLogger()

	// SIGINT/SIGTERM notifications arrive through this signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	lis, err := net.Listen("tcp", gRPCAddr)
	if err != nil {
		internalLogger.LogFatal(ctx, "An error has occurred when starting a Wallet Service gRPC server: %v", err)
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

	// Wiring the repository with persistence and caching
	walletRepository := repository.NewWalletRepository(postgreSqlDbClient, cacheClient)

	// Connecting to Kafka, then building the producer/consumer from the
	// ConnectMessaging contract (decoupled from concrete implementation)
	messaging, err := events.NewKafkaMessagingInstance(kafkaConsumerGroup, kafkaBrokers)
	if err != nil {
		internalLogger.LogFatal(ctx, "An error has occurred when building the kafka messaging client: %v", err)
	}

	if err := messaging.Connect(); err != nil {
		internalLogger.LogFatal(ctx, "An error has occurred when connecting to kafka: %v", err)
	}
	defer messaging.CloseConnection()

	producer := kafkainfra.NewProducer(messaging, walletEventsTopic)

	// Service declaration
	service := services.NewWalletService(walletRepository, producer)

	// background poller polling transaction_outbox and
	// publishing to Kafka via the same producer, decoupled from the gRPC
	outboxRelay := services.NewOutboxRelayService(walletRepository, producer, walletEventsTopic, outboxBatchSize, time.Duration(outboxPollInterval)*time.Second)

	go outboxRelay.CheckAndRelay(ctx)

	// Dedicated event dispatcher which delegates a handler function to fired event
	dispatcher := services.NewEventDispatcherService()
	dispatcher.Register(events.TransactionDepositRequested, service.HandleDepositDispatched)
	dispatcher.Register(events.TransactionWithdrawalRequested, service.HandleWithdrawalDispatched)
	dispatcher.Register(events.TransactionTransferRequested, service.HandleTransferDispatched)

	consumer := kafkainfra.NewConsumer(messaging, walletEventsTopic)
	if err := consumer.Consume(ctx, dispatcher.Handle); err != nil {
		internalLogger.LogFatal(ctx, "An error occurred when starting the kafka consumer: %v", err)
	}

	grpcServer := grpcserver.NewServer()

	// Depending on the provided gRPC call we either forward it to the walletService or to outboxRelayService for processing
	infra.NewGrpcHandler(grpcServer, service, outboxRelay)

	// Registering the gRPC health checks
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Initializing gRPC ui for the ease of testing in development
	// In Production environment we obviously won't have such UI
	if environment == "development" {
		reflection.Register(grpcServer)
	}

	internalLogger.LogInfo(ctx, "Starting gRPC server Wallet Service on port: %s", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			internalLogger.LogError(ctx, "failed to server: %v", err)
			// cancel()
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
