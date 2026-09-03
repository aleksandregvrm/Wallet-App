.PHONY: generate-proto run-gateway dev run-auth run-wallet seed run-gateway createNewMigration migrateup migratedown migrateupLast migrate-status build test-api-gateway test-api-gateway-unit test-api-gateway-integration test-auth-service test-auth-service-unit test-auth-service-integration test-wallet-service test-wallet-service-unit test-wallet-service-integration test-coverage

# Injecting environment variables in the during bootstrap
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

GATEWAY_PORT ?= 8080
AUTH_PORT ?= 8081
WALLET_PORT ?= 8082

PROTO_SRC ?= $(wildcard $(PROTO_DIR)/*.proto)
PROTO_DIR ?= ./shared/proto
PROTO_MODULE ?= go-task-wallet-service
GO_OUT ?= .

DB_USER ?= go_master
DB_PASSWORD ?= test
DB_HOST ?= localhost
DB_PORT ?= 5433

# Directory where migration files will be generated in. And migrations to the DB will be generated from
GOOSE_DIR ?= ./shared/migrations

DB_URL_RAW := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/?sslmode=disable


# Dynamically assigns the out directory based on the package of the proto file
# Example auth -> impl/ait folder, wallet -> impl/wallet folder and etc.
generate-proto:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) --go_opt=module=$(PROTO_MODULE) \
		--go-grpc_out=$(GO_OUT) --go-grpc_opt=module=$(PROTO_MODULE) \
		$(PROTO_SRC)

# Running the all the services in development mode using air with hot refresh (automatic restart on code changes)
# Injecting ENV variables
dev:
	set -a; [ -f .env ] && . ./.env; set +a; \
	air -c api-gateway/.air.toml & \
 	air -c auth-service/.air.toml & \
 	air -c wallet-service/.air.toml & \
	wait

# Running the API Gateway in isolation
run-gateway:
	echo "Running API Gateway on port $(GATEWAY_PORT)..."
	go run ./api-gateway/cmd

# Running the Auth Service in isolation
run-auth:
	echo "Running Auth Service on port $(AUTH_PORT)..."
	go run ./auth-service/cmd

# Running the Wallet Service in isolation
run-wallet:
	echo "Running Wallet Service on port $(WALLET_PORT)..."
	go run ./wallet-service/cmd

# Populating Database with fake data
# Given script is not published on github
seed:
	go run ./scripts/seed $(SEED_ARGS)

# Migration commands
createNewMigration:
	@read -p "Enter migration name: " name; \
	goose -dir $(GOOSE_DIR) create $$name sql

# Applying all pending migrations
migrateup:
	goose -dir $(GOOSE_DIR) postgres "$(DB_URL_RAW)" up

# Rolling back the last made migration
migratedown:
	goose -dir $(GOOSE_DIR) postgres "$(DB_URL_RAW)" down

# Applying only the single next migration
migrateupLast:
	goose -dir $(GOOSE_DIR) postgres "$(DB_URL_RAW)" up-by-one
	
# Checking the migration statuses
migrate-status:
	goose -dir $(GOOSE_DIR) postgres "$(DB_URL_RAW)" status

# Compiling the whole project.
build: 
	go build ./...

# Implementing api-gateway tests
test-api-gateway:
	go test -v ./api-gateway/tests/...

test-api-gateway-unit:
	go test -v ./api-gateway/tests/unit_tests/...

test-api-gateway-integration:
	go test -v ./api-gateway/tests/Integration_tests/...

# Implementing auth-service tests
test-auth-service:
	go test -v ./auth-service/tests/...

test-auth-service-unit:
	go test -v ./auth-service/tests/unit_tests/...

test-auth-service-integration:
	go test -v ./auth-service/tests/Integration_tests/...

# Implementing wallet-service tests
test-wallet-service:
	go test -v ./wallet-service/tests/...

test-wallet-service-unit:
	go test -v ./wallet-service/tests/unit_tests/...

test-wallet-service-integration:
	go test -v ./wallet-service/tests/Integration_tests/...

# Coverage test launches all tests at once
test-coverage:
	go test -v ./wallet-service/tests/...& \
	go test -v ./auth-service/tests/...& \
	go test -v ./api-gateway/tests/...