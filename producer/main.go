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
	Quantity       float64   `json:"quantity"` // +buy / -sell
	Price          *float64  `json:"price,omitempty"`
	TS             time.Time `json:"ts"`
}

var (
	entities   = []string{"zurich", "new_york"}
	instrTypes = []string{"stock", "crypto"}

	stocks    = []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "NVDA", "NFLX"}
	stockBase = map[string]float64{"AAPL": 190, "MSFT": 420, "GOOGL": 145, "AMZN": 180, "TSLA": 220, "NVDA": 800, "NFLX": 550}

	cryptos    = []string{"BTC", "ETH", "SOL", "ADA", "XRP"}
	cryptoBase = map[string]float64{"BTC": 60000, "ETH": 3200, "SOL": 150, "ADA": 0.45, "XRP": 0.6}

	// Properly seeded RNG used everywhere in this file
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// --- env helpers ---

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
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		log.Fatal("KAFKA_BROKERS is empty")
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

func parseBoolEnv(k string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func parseDurationEnv(k string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(k))
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("WARN: invalid %s=%q, using default %s", k, raw, def)
		return def
	}
	return d
}

// --- small utils ---

func round(x float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(x*scale) / scale
}

func pick[T any](xs []T) T { return xs[rng.Intn(len(xs))] }

// --- trade generation ---

func genTrade() Trade {
	ent := pick(entities)
	itype := pick(instrTypes)

	var sym string
	var qty float64
	var px float64

	if itype == "stock" {
		sym = pick(stocks)
		base := stockBase[sym]
		px = round(base*(1+(rng.Float64()-0.5)*0.03), 2) // ±1.5%
		q := float64(rng.Intn(50) + 1)                   // 1–50 shares
		if rng.Intn(2) == 0 {
			q = -q // sell
		}
		qty = q
	} else {
		sym = pick(cryptos)
		base := cryptoBase[sym]
		px = round(base*(1+(rng.Float64()-0.5)*0.05), 2) // ±2.5%
		q := 0.001 + rng.Float64()*(0.5-0.001)           // 0.001–0.5
		q = round(q, 4)
		if rng.Intn(2) == 0 {
			q = -q
		}
		qty = q
	}

	price := px
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

// Optionally attempt to create the topic (idempotent; errors ignored if exists).
func ensureTopic(ctx context.Context, broker, topic string) {
	conn, err := kafka.DialContext(ctx, "tcp", broker)
	if err != nil {
		log.Printf("ensureTopic: dial failed: %v", err)
		return
	}
	defer conn.Close()
	if err := conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}); err != nil {
		log.Printf("ensureTopic: create(%s): %v (ok if already exists)", topic, err)
	}
}

// --- main ---

func main() {
	brokers := mustBrokers(envOr("KAFKA_BROKERS", "kafka:9092"))
	topic := envOr("KAFKA_TOPIC", "trades")
	rate := parseRate()

	stayAlive := parseBoolEnv("PRODUCER_STAY_ALIVE", false)
	ttl := parseDurationEnv("PRODUCER_TTL", 1*time.Minute) // default: 1min
	ensure := parseBoolEnv("PRODUCER_ENSURE_TOPIC", true)

	// Base context cancelled by SIGINT/SIGTERM
	baseCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TTL wrapper: only active when not stayAlive and ttl>0
	ctx := baseCtx
	if !stayAlive && ttl > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(baseCtx, ttl) // <— this drives auto-shutdown
		defer cancel()
	}

	// Best-effort topic ensure
	if ensure {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ensureTopic(c, brokers[0], topic)
		cancel()
	}

	// Writer
	dialer := &kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           200 * time.Millisecond,
		Dialer:                 dialer,
	}
	defer func() { _ = w.Close() }()

	log.Printf("producer: brokers=%v topic=%s rate=%d/s stayAlive=%v ttl=%s", brokers, topic, rate, stayAlive, ttl)

	// 1/rate seconds per trade
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
			// Tiny jitter to avoid sync with other producers
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
