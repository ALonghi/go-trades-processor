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

	"github.com/example/trades-aggregator/internal/cache"
	"github.com/example/trades-aggregator/internal/config"
	"github.com/example/trades-aggregator/internal/db"
	"github.com/example/trades-aggregator/internal/holdings"
	kafkaconsumer "github.com/example/trades-aggregator/internal/kafka"
	httpserver "github.com/example/trades-aggregator/internal/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbpool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db", zap.Error(err))
	}
	defer dbpool.Close()

	svc := holdings.New(dbpool)
	cache, err := cache.New(1<<26 /* ~64MB */, cfg.CacheTTL)
	if err != nil {
		logger.Fatal("cache", zap.Error(err))
	}

	cons := kafkaconsumer.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, svc, logger)
	go func() {
		if err := cons.Run(ctx); err != nil {
			logger.Error("consumer", zap.Error(err))
			cancel()
		}
	}()

	s := httpserver.NewServer(svc, cache, logger, cfg.CORSOrigin)

	server := &http.Server{Addr: ":" + cfg.Port, Handler: s.R}
	go func() {
		logger.Info("http listening", zap.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http", zap.Error(err))
		}
	}()

	// graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctxShut, cancelShut := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShut()
	_ = server.Shutdown(ctxShut)
	logger.Info("shutdown complete")
}
