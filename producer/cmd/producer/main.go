package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := LoadConfig()

	// Base context canceled by SIGINT/SIGTERM
	baseCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Apply TTL unless stay-alive requested or TTL <= 0
	ctx := baseCtx
	if !cfg.ProducerStayAlive && cfg.ProducerTTL > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(baseCtx, cfg.ProducerTTL)
		defer cancel()
	}

	// Best-effort ensure topic exists (short timeout)
	if cfg.ProducerEnsureTopic {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		EnsureTopic(c, cfg.Brokers[0], cfg.Topic)
		cancel()
	}

	// Create writer
	writer, err := NewKafkaWriter(cfg.Brokers, cfg.Topic)
	if err != nil {
		log.Fatalf("producer: failed to create kafka writer: %v", err)
	}
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("producer: writer close error: %v", err)
		}
	}()

	log.Printf("producer: brokers=%v topic=%s rate=%d/s stayAlive=%v ttl=%s", cfg.Brokers, cfg.Topic, cfg.Rate, cfg.ProducerStayAlive, cfg.ProducerTTL)

	// production loop
	runProducerLoop(ctx, cfg, writer)
}
