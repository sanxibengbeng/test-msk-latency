package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"msk-delay/pkg/config"
	"msk-delay/pkg/logger"
)

// Consumer represents a Kafka message consumer
type Consumer struct {
	reader         *kafka.Reader
	config         *config.KafkaConfig
	logger         *logger.Logger
	consumerID     int
	processingTime time.Duration
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.KafkaConfig, id int) *Consumer {
	// Set default rebalance timeout to 5 seconds
	rebalanceTimeout := 5 * time.Second

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
		GroupID: cfg.ConsumerGroup,
		// MinBytes:    5e3,  // 5KB
		// MaxBytes:    5e6,  // 5MB
		StartOffset: kafka.LastOffset,
		// Optimized consumer group settings for rebalancing tests
		// HeartbeatInterval: 2 * time.Second,
		// SessionTimeout:    20 * time.Second,
		RebalanceTimeout: rebalanceTimeout,
		// Set maximum wait time for fetching messages
		// MaxWait: 500 * time.Millisecond,
		// Commit messages automatically
		CommitInterval: 1 * time.Second,
		// Enable auto commit
		// AutoCommit: true,
		// Enable partition watching for rebalancing tests
		WatchPartitionChanges:  true,
		PartitionWatchInterval: 30 * time.Second,
		// Enable verbose logging for consumer group events
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			fmt.Printf("[KAFKA-INTERNAL][Consumer-%d] %s\n", id, fmt.Sprintf(msg, args...))
		}),
	})

	return &Consumer{
		reader:         reader,
		config:         cfg,
		logger:         logger.NewLogger(fmt.Sprintf("Consumer-%d", id)),
		consumerID:     id,
		processingTime: time.Duration(cfg.ConsumerProcessingMS) * time.Millisecond,
	}
}

// Start begins consuming messages
func (c *Consumer) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer c.reader.Close()

	c.logger.Info("Consumer started, processing time: %v", c.processingTime)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer shutting down")
			return
		default:
			message, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err.Error() != "context canceled" {
					c.logger.Error("Error fetching message: %v", err)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Extract message metadata
			var messageID string
			var timestamp string
			for _, header := range message.Headers {
				if header.Key == "message_id" {
					messageID = string(header.Value)
				}
				if header.Key == "timestamp" {
					timestamp = string(header.Value)
				}
			}

			// Log detailed information about the message
			c.logger.Info("Received message: ID=%s, Partition=%d, Offset=%d, Size=%d bytes",
				messageID, message.Partition, message.Offset, len(message.Value))

			// Log consumer group information
			c.logger.Debug("Consumer group details: GroupID=%s, ConsumerID=%d, Topic=%s",
				c.config.ConsumerGroup, c.consumerID, c.config.Topic)

			// Parse the producer timestamp if available
			if timestamp != "" {
				if producerTime, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
					latency := time.Since(producerTime)
					c.logger.Info("Message latency: %v", latency)
				}
			}

			// Simulate processing time
			c.logger.Debug("Processing message for %v...", c.processingTime)
			time.Sleep(c.processingTime)
			c.logger.Debug("Finished processing message %s", messageID)

			// With AutoCommit enabled, we don't need to manually commit
			// But we can still track when messages are processed
			c.logger.Debug("Processed message: ID=%s, Partition=%d, Offset=%d",
				messageID, message.Partition, message.Offset)
		}
	}
}

// Close closes the consumer
func (c *Consumer) Close() error {
	c.logger.Info("Closing consumer")
	return c.reader.Close()
}
