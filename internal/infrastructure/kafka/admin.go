package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// EnsureTopics creates the given topics if they don't already exist.
func EnsureTopics(ctx context.Context, broker string, topics []string) error {
	conn, err := kafka.DialContext(ctx, "tcp", broker)
	if err != nil {
		return fmt.Errorf("kafka dial: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("kafka controller: %w", err)
	}

	ctrlConn, err := kafka.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("kafka controller dial: %w", err)
	}
	defer ctrlConn.Close()

	configs := make([]kafka.TopicConfig, len(topics))
	for i, t := range topics {
		configs[i] = kafka.TopicConfig{
			Topic:             t,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}
	}

	if err = ctrlConn.CreateTopics(configs...); err != nil {
		if kafkaErr, ok := err.(kafka.Error); !ok || kafkaErr != kafka.TopicAlreadyExists {
			return fmt.Errorf("create topics: %w", err)
		}
	}
	return nil
}
