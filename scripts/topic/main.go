package main

// Script that connects to the kafka instance. And initializes it's configuration with topics named in the /shared folder
// The script checks for the health status of kafka. In case kafka has not booted up yet retries connection.
// Once connection is established by default 3 partition and 2 topics are initialized for wallet events and wallet events DLQ

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/events"
	"go-task-wallet-service/shared/retry"

	"github.com/segmentio/kafka-go"
)

func main() {
	brokers := strings.Split(env.GetString("KAFKA_BROKERS", "localhost:9092"), ",")
	partitions := env.GetInt("KAFKA_TOPIC_PARTITIONS", 3)
	replicationFactor := env.GetInt("KAFKA_TOPIC_REPLICATION_FACTOR", 1)
	dialTimeout := time.Duration(env.GetInt("KAFKA_DIAL_TIMEOUT_SECONDS", 30)) * time.Second

	// Define topics in the shared events
	topics := []string{
		events.WalletEventsTopic,
		events.WalletEventsTopic + events.DlqTopicSuffix,
	}

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	// Healthcheck retry. Ensuring stable connection
	var conn *kafka.Conn
	err := retry.WithBackoff(ctx, retry.DefaultConfig(), func() error {
		c, dialErr := kafka.Dial("tcp", brokers[0])
		if dialErr != nil {
			return fmt.Errorf("dial broker %s: %w", brokers[0], dialErr)
		}
		conn = c
		return nil
	})
	if err != nil {
		log.Fatalf("failed to connect to kafka broker %s: %v", brokers[0], err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		log.Fatalf("failed to discover kafka controller: %v", err)
	}

	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerConn, err := kafka.Dial("tcp", controllerAddr)
	if err != nil {
		log.Fatalf("failed to connect to kafka controller at %s: %v", controllerAddr, err)
	}
	defer controllerConn.Close()

	topicConfigs := make([]kafka.TopicConfig, 0, len(topics))
	for _, topic := range topics {
		topicConfigs = append(topicConfigs, kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: replicationFactor,
		})
	}

	// Topic creation which will append the given topics to the kafka instance. In case topics already exist
	if err := controllerConn.CreateTopics(topicConfigs...); err != nil {
		log.Fatalf("failed to create topics: %v", err)
	}

	log.Printf("topic-init: ensured topics %v (partitions=%d, replication_factor=%d)", topics, partitions, replicationFactor)
}
