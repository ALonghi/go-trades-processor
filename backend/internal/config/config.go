package config

import (
	"time"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	DatabaseURL  string        `env:"DATABASE_URL,required"`
	KafkaBrokers string        `env:"KAFKA_BROKERS,required"`
	KafkaTopic   string        `env:"KAFKA_TOPIC,required"`
	KafkaGroupID string        `env:"KAFKA_GROUP_ID,required"`
	Port         string        `env:"PORT" envDefault:"8080"`
	CORSOrigin   string        `env:"CORS_ORIGIN" envDefault:"*"`
	CacheTTL     time.Duration `env:"CACHE_TTL" envDefault:"60s"`
}

func Load() (Config, error) {
	var cfg Config
	return cfg, env.Parse(&cfg)
}
