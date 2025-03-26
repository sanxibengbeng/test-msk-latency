package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"msk-delay/pkg/config"
	"msk-delay/pkg/logger"
)

// Admin provides administrative functions for Kafka
type Admin struct {
	config *config.KafkaConfig
	logger *logger.Logger
}

// NewAdmin creates a new Kafka admin client
func NewAdmin(cfg *config.KafkaConfig) *Admin {
	return &Admin{
		config: cfg,
		logger: logger.NewLogger("KafkaAdmin"),
	}
}

// CreateTopic creates a topic with the specified configuration
func (a *Admin) CreateTopic(ctx context.Context) error {
	conn, err := kafka.Dial("tcp", a.config.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	a.logger.Info("Creating topic %s with %d partitions and replication factor %d",
		a.config.Topic, a.config.Partitions, a.config.ReplicationFactor)

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             a.config.Topic,
			NumPartitions:     a.config.Partitions,
			ReplicationFactor: a.config.ReplicationFactor,
			ConfigEntries: []kafka.ConfigEntry{
				{
					ConfigName:  "retention.ms",
					ConfigValue: "86400000", // 24 hours
				},
			},
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	a.logger.Info("Topic %s created successfully", a.config.Topic)
	return nil
}

// DeleteTopic deletes the specified topic
func (a *Admin) DeleteTopic(ctx context.Context) error {
	conn, err := kafka.Dial("tcp", a.config.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	a.logger.Info("Deleting topic %s", a.config.Topic)
	err = controllerConn.DeleteTopics(a.config.Topic)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	a.logger.Info("Topic %s deleted successfully", a.config.Topic)
	return nil
}

// WaitForTopic waits for a topic to be ready
func (a *Admin) WaitForTopic(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		conn, err := kafka.Dial("tcp", a.config.Brokers[0])
		if err != nil {
			a.logger.Warn("Failed to connect to Kafka: %v, retrying...", err)
			time.Sleep(1 * time.Second)
			continue
		}
		
		partitions, err := conn.ReadPartitions(a.config.Topic)
		conn.Close()
		
		if err != nil {
			a.logger.Warn("Topic %s not ready yet: %v", a.config.Topic, err)
			time.Sleep(1 * time.Second)
			continue
		}
		
		if len(partitions) == a.config.Partitions {
			a.logger.Info("Topic %s is ready with %d partitions", a.config.Topic, len(partitions))
			return nil
		}
		
		a.logger.Warn("Topic %s has %d partitions, expected %d", 
			a.config.Topic, len(partitions), a.config.Partitions)
		time.Sleep(1 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for topic %s to be ready", a.config.Topic)
}
