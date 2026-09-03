FROM golang:1.26-alpine AS builder

WORKDIR /app

# Downloading dependencies
COPY go.mod go.sum ./
RUN go mod download

# Combining the topic-init runner with shared (env/events/retry packagess)
COPY shared shared
COPY scripts/topic scripts/topic

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/topic-init ./scripts/topic

FROM scratch

WORKDIR /

COPY --from=builder /out/topic-init /topic-init

USER 65532:65532

ENTRYPOINT ["/topic-init"]
