package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Client wraps a Kafka connection
type Client struct {
	brokers []string
	writer  *kafka.Writer
	reader  *kafka.Reader
}

// NewClient creates a new Kafka client
func NewClient(brokers string) *Client {
	return &Client{
		brokers: []string{brokers},
	}
}

// NewWriter creates a new Kafka writer for a topic
func (c *Client) NewWriter(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(c.brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
	}
}

// NewReader creates a new Kafka reader for a topic
func (c *Client) NewReader(topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})
}

// CreateTopic creates a topic if it doesn't exist (for dev/testing)
func (c *Client) CreateTopic(ctx context.Context, topic string, partitions int) error {
	conn, err := kafka.DialContext(ctx, "tcp", c.brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: 1, // For dev
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		// Topic might already exist, which is fine
		return nil
	}

	return nil
}

// Health checks if Kafka is reachable
func (c *Client) Health(ctx context.Context) error {
	conn, err := kafka.DialContext(ctx, "tcp", c.brokers[0])
	if err != nil {
		return fmt.Errorf("kafka unreachable: %w", err)
	}
	defer conn.Close()
	return nil
}
