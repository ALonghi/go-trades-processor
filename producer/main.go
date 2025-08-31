package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Trade struct {
	TradeID        string    `json:"trade_id"`
	Entity         string    `json:"entity"`          // "zurich" | "new_york"
	InstrumentType string    `json:"instrument_type"` // "stock" | "crypto"
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`        // +buy / -sell
	Price          *float64  `json:"price,omitempty"` // quoted currency (USD here)
	TS             time.Time `json:"ts"`              // RFC3339
}

var (
	stocks     = []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "NVDA", "NFLX"}
	stockBase  = map[string]float64{"AAPL": 190, "MSFT": 420, "GOOGL": 145, "AMZN": 180, "TSLA": 220, "NVDA": 800, "NFLX": 550}
	cryptos    = []string{"BTC", "ETH", "SOL", "ADA", "XRP"}
	cryptoBase = map[string]float64{"BTC": 60000, "ETH": 3200, "SOL": 150, "ADA": 0.45, "XRP": 0.6}
	entities   = []string{"zurich", "new_york"}
	instrTypes = []string{"stock", "crypto"}
)

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func mustBrokers(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		log.Fatalf("KAFKA_BROKERS is empty")
	}
	return out
}

func parseRate() int {
	n := 1
	if v := strings.TrimSpace(os.Getenv("TRADES_PER_SEC")); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 && i <= 50 {
			n = i
		}
	}
	return n
}

func round(x float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(x*scale) / scale
}

func pick[T any](xs []T) T {
	return xs[rand.Intn(len(xs))]
}

func genTrade() Trade {
	ent := pick(entities)
	itype := pick(instrTypes)

	var sym string
	var qty float64
	var px float64

	if itype == "stock" {
		sym = pick(stocks)
		base := stockBase[sym]
		// ±1.5% variation, rounded to cents
		px = round(base*(1+(rand.Float64()-0.5)*0.03), 2)
		// integer shares 1–50, random buy/sell
		q := float64(rand.Intn(50) + 1)
		if rand.Intn(2) == 0 {
			q = -q
		}
		qty = q
	} else {
		sym = pick(cryptos)
		base := cryptoBase[sym]
		// ±2.5% variation, 2 decimals (can tweak per symbol)
		px = round(base*(1+(rand.Float64()-0.5)*0.05), 2)
		// fractional size 0.001–0.5, random buy/sell
		q := 0.001 + rand.Float64()*(0.5-0.001)
		q = round(q, 4)
		if rand.Intn(2) == 0 {
			q = -q
		}
		qty = q
	}

	price := px // take address
	return Trade{
		TradeID:        uuid.NewString(),
		Entity:         ent,
		InstrumentType: itype,
		Symbol:         sym,
		Quantity:       qty,
		Price:          &price,
		TS:             time.Now().UTC(),
	}
}

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano())) // nolint:gosec – pseudo RNG is fine here

	brokers := mustBrokers(envOr("KAFKA_BROKERS", "kafka:9092"))
	topic := envOr("KAFKA_TOPIC", "trades")
	rate := parseRate()

	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           200 * time.Millisecond,
	}
	defer func() { _ = w.Close() }()

	log.Printf("producer: brokers=%v topic=%s rate=%d tps", brokers, topic, rate)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1/rate seconds per trade
	period := time.Second / time.Duration(rate)
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("producer: shutting down")
			return
		case <-ticker.C:
			// small jitter to avoid sync with other producers
			time.Sleep(time.Duration(rand.Intn(150)) * time.Millisecond)

			t := genTrade()
			b, _ := json.Marshal(t)
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
