package ingest

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/aegis-decision-engine/ade/internal/storage/postgres"
	"github.com/segmentio/kafka-go"
)

// Consumer consumes events from Kafka
type Consumer struct {
	reader      *kafka.Reader
	eventStore  *postgres.EventStore
	logger      *slog.Logger
	stopChan    chan struct{}
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(reader *kafka.Reader, eventStore *postgres.EventStore, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{
		reader:     reader,
		eventStore: eventStore,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

// Start begins consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("starting kafka consumer")
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopChan:
			return nil
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Error("failed to read message", "error", err)
			time.Sleep(time.Second)
			continue
		}

		if err := c.processMessage(ctx, msg); err != nil {
			c.logger.Error("failed to process message", "error", err, "offset", msg.Offset)
		}
	}
}

// Stop stops the consumer
func (c *Consumer) Stop() error {
	close(c.stopChan)
	return c.reader.Close()
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var event models.Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	// Only process if we have an event store
	if c.eventStore == nil {
		c.logger.Debug("skipping persistence, no event store")
		return nil
	}

	// Check if already processed (idempotency)
	existing, err := c.eventStore.GetByID(ctx, event.EventID)
	if err == nil && existing != nil {
		c.logger.Debug("event already processed", "event_id", event.EventID)
		return nil
	}

	// Store event
	if err := c.eventStore.Store(ctx, &event); err != nil {
		return err
	}

	c.logger.Info("event consumed from kafka",
		"event_id", event.EventID,
		"service_id", event.ServiceID,
		"offset", msg.Offset,
	)

	return nil
}
