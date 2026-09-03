FROM golang:1.26-alpine AS builder

WORKDIR /app

# Downloading dependencies
COPY go.mod go.sum ./
RUN go mod download

# Combining the actual service - api-gateway with shared package to combine it and build it together
COPY shared shared
COPY api-gateway api-gateway

# Removing redundant test files from the image
RUN rm -rf ./api-gateway/tests

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api-gateway ./api-gateway/cmd

FROM alpine:latest

WORKDIR /

# Copying the compiled binary
COPY --from=builder /out/api-gateway /api-gateway

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/api-gateway"]
