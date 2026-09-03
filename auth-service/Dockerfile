FROM golang:1.26-alpine AS builder

WORKDIR /app

# Downloading dependencies
COPY go.mod go.sum ./
RUN go mod download

# Combining the actual service - auth-service with shared package to combine it and build it together
COPY shared shared
COPY auth-service auth-service

# Removing redundant test files from the image
RUN rm -rf ./auth-service/tests

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/auth-service ./auth-service/cmd


# GRPC_HEALTH_PROBE for gRPC server liveness & healthiness checks
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.24 && \
    wget -qO/bin/grpc_health_probe \
      https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

FROM scratch

WORKDIR /

# Copying the compiled binary
COPY --from=builder /out/auth-service /auth-service
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

USER 65532:65532

EXPOSE 8081

ENTRYPOINT ["/auth-service"]
