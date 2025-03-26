package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"msk-delay/pkg/config"
	"msk-delay/pkg/kafka"
	"msk-delay/pkg/logger"
)

func main() {
	// Set up random seed
	rand.Seed(time.Now().UnixNano())

	// Parse command line flags
	envFlag := flag.String("env", "dev", "Environment to use (dev or prod)")
	flag.Parse()

	log := logger.NewLogger("ProducerMain")
	log.Info("Starting producer in %s environment", *envFlag)

	// Load configuration
	cfg, err := config.LoadConfig(*envFlag)
	if err != nil {
		log.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info("Received signal %v, shutting down", sig)
		cancel()
	}()

	// Create and initialize Kafka admin
	admin := kafka.NewAdmin(cfg)

	// Create topic if it doesn't exist
	if err := admin.CreateTopic(ctx); err != nil {
		log.Warn("Error creating topic (it may already exist): %v", err)
	}

	// Wait for topic to be ready
	if err := admin.WaitForTopic(ctx, 30*time.Second); err != nil {
		log.Error("Topic not ready: %v", err)
		os.Exit(1)
	}

	// Create and start producer
	producer := kafka.NewProducer(cfg)
	if err := producer.Start(ctx); err != nil {
		log.Error("Producer error: %v", err)
		os.Exit(1)
	}
}
