package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"msk-delay/pkg/config"
	"msk-delay/pkg/kafka"
	"msk-delay/pkg/logger"
)

func main() {
	// Parse command line flags
	envFlag := flag.String("env", "dev", "Environment to use (dev or prod)")
	flag.Parse()

	log := logger.NewLogger("ConsumerMain")
	log.Info("Starting consumer in %s environment", *envFlag)

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

	// Wait for topic to be ready
	if err := admin.WaitForTopic(ctx, 30*time.Second); err != nil {
		log.Error("Topic not ready: %v", err)
		os.Exit(1)
	}

	// Create and start consumers
	var wg sync.WaitGroup
	consumerCount := cfg.ConsumerCount
	log.Info("Starting %d consumers", consumerCount)

	for i := 0; i < consumerCount; i++ {
		wg.Add(1)
		consumer := kafka.NewConsumer(cfg, i+1)
		go consumer.Start(ctx, &wg)
		
		// Stagger consumer starts to observe rebalancing behavior
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all consumers to finish
	wg.Wait()
	log.Info("All consumers have shut down")
}
