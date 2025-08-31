package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

func runProducerLoop(ctx context.Context, cfg Config, w *kafka.Writer) {
	rate := cfg.Rate
	if rate <= 0 {
		rate = 1
	}
	period := time.Second / time.Duration(rate)
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				log.Println("producer: TTL reached; exiting")
			} else {
				log.Println("producer: shutting down (signal)")
			}
			return
		case <-ticker.C:
			// jitter
			time.Sleep(time.Duration(rng.Intn(150)) * time.Millisecond)

			t := genTrade()
			b, err := json.Marshal(t)
			if err != nil {
				log.Printf("marshal error: %v", err)
				continue
			}

			key := []byte(fmt.Sprintf("%s|%s", t.Entity, t.Symbol))
			msg := kafka.Message{Key: key, Value: b, Time: t.TS}

			if err := w.WriteMessages(ctx, msg); err != nil {
				log.Printf("write error: %v", err)
				continue
			}
			log.Printf("sent: %s %s %s %s qty=%v price=%v",
				t.TradeID, t.Entity, t.InstrumentType, t.Symbol, t.Quantity, *t.Price)
		}
	}
}
