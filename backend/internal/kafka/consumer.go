package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/example/trades-aggregator/internal/holdings"
	"github.com/example/trades-aggregator/internal/models"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Consumer struct {
	Reader *kafka.Reader
	Svc    *holdings.Service
	Logger *zap.Logger
}

func NewConsumer(brokers, topic, groupID string, svc *holdings.Service, logger * /*  */ zap.Logger) *Consumer {
	return &Consumer{
		Reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{brokers},
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 1e3,
			MaxBytes: 1e6,
			MaxWait:  500 * time.Millisecond,
		}),
		Svc:    svc,
		Logger: logger,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	defer c.Reader.Close()
	for {
		m, err := c.Reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		var t models.Trade
		if err := json.Unmarshal(m.Value, &t); err != nil {
			c.Logger.Warn("bad message", zap.Error(err))
			continue
		}
		if t.TS.IsZero() {
			t.TS = time.Now().UTC()
		}
		if err := c.Svc.ApplyTrade(ctx, t); err != nil {
			c.Logger.Error("apply trade", zap.Error(err))
		} else {
			c.Logger.Debug("trade applied", zap.String("trade_id", t.TradeID))
		}
	}
}
