package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// KafkaConfig holds the configuration for Kafka
type KafkaConfig struct {
	Brokers               []string `json:"brokers"`
	Topic                 string   `json:"topic"`
	Partitions            int      `json:"partitions"`
	ReplicationFactor     int      `json:"replication_factor"`
	ConsumerGroup         string   `json:"consumer_group"`
	ConsumerCount         int      `json:"consumer_count"`
	MessageSizeKB         int      `json:"message_size_kb"`
	ProducerIntervalMS    int      `json:"producer_interval_ms"`
	ConsumerProcessingMS  int      `json:"consumer_processing_time_ms"`
}

// LoadConfig loads the configuration from the specified environment
func LoadConfig(env string) (*KafkaConfig, error) {
	configPath := filepath.Join("conf", env, "kafka.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config KafkaConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}
