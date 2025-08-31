package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Brokers             []string
	Topic               string
	Rate                int
	ProducerStayAlive   bool
	ProducerTTL         time.Duration
	ProducerEnsureTopic bool
}

func LoadConfig() Config {
	brokers := parseBrokers(envOr("KAFKA_BROKERS", "kafka:9092"))
	topic := envOr("KAFKA_TOPIC", "trades")

	rate := 1
	if v := strings.TrimSpace(os.Getenv("TRADES_PER_SEC")); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 && i <= 50 {
			rate = i
		}
	}

	stayAlive := parseBoolEnv("PRODUCER_STAY_ALIVE", false)
	ensure := parseBoolEnv("PRODUCER_ENSURE_TOPIC", true)

	ttl := 2 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("PRODUCER_TTL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			ttl = d
		} else {
			log.Printf("WARN: invalid PRODUCER_TTL=%q, using default %s", raw, ttl)
		}
	}

	return Config{
		Brokers:             brokers,
		Topic:               topic,
		Rate:                rate,
		ProducerStayAlive:   stayAlive,
		ProducerTTL:         ttl,
		ProducerEnsureTopic: ensure,
	}
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func parseBrokers(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		// fatal here because producer cannot run
		log.Fatal("KAFKA_BROKERS is empty")
	}
	return out
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
