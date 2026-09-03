package main

import (
	"context"
	"go-task-wallet-service/shared/logging"
)

func main() {
	logging.NewInternalLogger().LogInfo(context.Background(), "wallet-service running")
}
