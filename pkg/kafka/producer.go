package kafka

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/protocol"

	"msk-delay/pkg/config"
	"msk-delay/pkg/logger"
)

// Producer represents a Kafka message producer
type Producer struct {
	writer *kafka.Writer
	config *config.KafkaConfig
	logger *logger.Logger
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *config.KafkaConfig) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}

	return &Producer{
		writer: writer,
		config: cfg,
		logger: logger.NewLogger("Producer"),
	}
}

// generateMessage creates a message of the specified size
func (p *Producer) generateMessage(sizeKB int) []byte {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	size := sizeKB * 1024
	
	message := make([]byte, size)
	for i := range message {
		message[i] = chars[rand.Intn(len(chars))]
	}
	
	return message
}

// Start begins producing messages at the configured interval
func (p *Producer) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(p.config.ProducerIntervalMS) * time.Millisecond)
	defer ticker.Stop()
	
	messageCount := 0
	
	p.logger.Info("Starting producer with interval %dms and message size %dKB",
		p.config.ProducerIntervalMS, p.config.MessageSizeKB)
	
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Producer shutting down")
			return p.writer.Close()
		case <-ticker.C:
			messageCount++
			messageID := fmt.Sprintf("msg-%d", messageCount)
			message := p.generateMessage(p.config.MessageSizeKB)
			
			err := p.writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(messageID),
				Value: message,
				Headers: []protocol.Header{
					{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339Nano))},
					{Key: "message_id", Value: []byte(messageID)},
				},
			})
			
			if err != nil {
				p.logger.Error("Failed to write message: %v", err)
			} else {
				p.logger.Debug("Produced message %s (%d bytes)", messageID, len(message))
			}
		}
	}
}

// Close closes the producer
func (p *Producer) Close() error {
	p.logger.Info("Closing producer")
	return p.writer.Close()
}
