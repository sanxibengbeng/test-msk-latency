package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"

	"msk-delay/pkg/config"
	"msk-delay/pkg/logger"
)

// ConsumerSarama represents a Kafka message consumer using Sarama library
type ConsumerSarama struct {
	client         sarama.ConsumerGroup
	config         *config.KafkaConfig
	logger         *logger.Logger
	consumerID     int
	processingTime time.Duration
	topic          string
}

// consumerGroupHandler is the implementation of sarama.ConsumerGroupHandler interface
type consumerGroupHandler struct {
	consumerSarama *ConsumerSarama
	ctx            context.Context
}

// NewConsumerSarama creates a new Kafka consumer using Sarama library
func NewConsumerSarama(cfg *config.KafkaConfig, id int) (*ConsumerSarama, error) {
	// Create Sarama config
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_1_0 // Use a compatible version
	
	// Consumer group settings
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
	saramaConfig.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	saramaConfig.Consumer.Group.Rebalance.Timeout = 5 * time.Second
	saramaConfig.Consumer.Group.Session.Timeout = 20 * time.Second
	saramaConfig.Consumer.Group.Heartbeat.Interval = 2 * time.Second
	
	// Create consumer group
	consumerGroup := fmt.Sprintf("%s-sarama", cfg.ConsumerGroup)
	client, err := sarama.NewConsumerGroup(cfg.Brokers, consumerGroup, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating consumer group: %w", err)
	}

	return &ConsumerSarama{
		client:         client,
		config:         cfg,
		logger:         logger.NewLogger(fmt.Sprintf("ConsumerSarama-%d", id)),
		consumerID:     id,
		processingTime: time.Duration(cfg.ConsumerProcessingMS) * time.Millisecond,
		topic:          cfg.Topic,
	}, nil
}

// Start begins consuming messages
func (c *ConsumerSarama) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := c.client.Close(); err != nil {
			c.logger.Error("Error closing consumer group: %v", err)
		}
	}()

	c.logger.Info("Sarama consumer started, processing time: %v", c.processingTime)

	// Create consumer group handler
	handler := &consumerGroupHandler{
		consumerSarama: c,
		ctx:            ctx,
	}

	// Start consuming in a loop
	topics := []string{c.topic}
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Sarama consumer shutting down")
			return
		default:
			// Consume will block until the context is cancelled or an error occurs
			if err := c.client.Consume(ctx, topics, handler); err != nil {
				if err != context.Canceled {
					c.logger.Error("Error from consumer: %v", err)
				}
			}
			// If we reach here, either context was cancelled or there was an error
			// Check if context was cancelled
			if ctx.Err() != nil {
				return
			}
			// Otherwise, try to reconnect after a short delay
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Close closes the consumer
func (c *ConsumerSarama) Close() error {
	c.logger.Info("Closing Sarama consumer")
	return c.client.Close()
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.consumerSarama.logger.Info("Consumer group session setup")
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.consumerSarama.logger.Info("Consumer group session cleanup")
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c := h.consumerSarama
	
	for {
		select {
		case <-h.ctx.Done():
			return nil
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			
			// Extract message metadata
			var messageID string
			var timestamp string
			for _, header := range message.Headers {
				if string(header.Key) == "message_id" {
					messageID = string(header.Value)
				}
				if string(header.Key) == "timestamp" {
					timestamp = string(header.Value)
				}
			}

			// Log detailed information about the message
			c.logger.Info("Received message: ID=%s, Partition=%d, Offset=%d, Size=%d bytes",
				messageID, message.Partition, message.Offset, len(message.Value))

			// Log consumer group information
			c.logger.Debug("Consumer group details: GroupID=%s-sarama, ConsumerID=%d, Topic=%s",
				c.config.ConsumerGroup, c.consumerID, c.topic)

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

			// Mark message as processed
			session.MarkMessage(message, "")
			c.logger.Debug("Processed message: ID=%s, Partition=%d, Offset=%d",
				messageID, message.Partition, message.Offset)
		}
	}
}
