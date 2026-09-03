FROM golang:1.26-alpine AS builder

WORKDIR /app

# Downloading dependencies
COPY go.mod go.sum ./
RUN go mod download

# Combining the migration runner with shared (migration SQL files + db/env packages)
COPY shared shared
COPY scripts/migrations scripts/migrations

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./scripts/migrations

FROM scratch

WORKDIR /

# Copying all the migration sql files to apply them to the db instance
COPY --from=builder /out/migrate /migrate
COPY shared/migrations /shared/migrations

USER 65532:65532

ENTRYPOINT ["/migrate"]
