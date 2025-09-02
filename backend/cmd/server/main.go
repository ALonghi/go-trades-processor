package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/example/trades-aggregator/internal/config"
	"github.com/example/trades-aggregator/internal/db"
	"github.com/example/trades-aggregator/internal/holdings"
	httpserver "github.com/example/trades-aggregator/internal/http"
	kafkaconsumer "github.com/example/trades-aggregator/internal/kafka"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Logger
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	// Root context (cancellable; used by consumer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// DB
	dbpool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db_connect_failed", zap.Error(err))
	}
	defer dbpool.Close()

	// Domain services
	svc := holdings.New(dbpool)

	// Kafka consumer (lifecycle tied to ctx)
	consumer := kafkaconsumer.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, svc, logger)
	go func() {
		if err := consumer.Run(ctx); err != nil {
			logger.Error("kafka_consumer_error", zap.Error(err))
			cancel() // propagate failure
		}
	}()

	// HTTP server (the HTTP package now constructs its own typed caches)
	router := httpserver.NewServer(svc, logger, cfg.CORSOrigin)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.R,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start HTTP listener
	go func() {
		logger.Info("http_listen_start", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http_listen_failed", zap.Error(err))
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("shutdown_signal", zap.String("signal", sig.String()))

	// Cancel background work (consumer) and shutdown HTTP
	cancel()

	ctxShut, cancelShut := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShut()
	if err := srv.Shutdown(ctxShut); err != nil {
		logger.Warn("http_shutdown_error", zap.Error(err))
	}

	logger.Info("shutdown_complete")
}
