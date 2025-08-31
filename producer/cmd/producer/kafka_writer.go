package main

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// EnsureTopic attempts to create the topic (best-effort).
func EnsureTopic(ctx context.Context, broker, topic string) {
	conn, err := kafka.DialContext(ctx, "tcp", broker)
	if err != nil {
		log.Printf("ensureTopic: dial failed: %v", err)
		return
	}
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		log.Printf("ensureTopic: create(%s): %v (ok if exists)", topic, err)
	}
}

// NewKafkaWriter constructs a kafka.Writer compatible with kafka-go v0.4.x.
func NewKafkaWriter(brokers []string, topic string) (*kafka.Writer, error) {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	wCfg := kafka.WriterConfig{
		Brokers:      brokers,
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		Dialer:       dialer,
		BatchTimeout: 200 * time.Millisecond,
		RequiredAcks: int(kafka.RequireOne),
	}

	w := kafka.NewWriter(wCfg)
	return w, nil
}
