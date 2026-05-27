package kafka

import (
	"github.com/segmentio/kafka-go"
)

func NewWriter(broker, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(broker),
		Topic:                  topic,
		AllowAutoTopicCreation: true,
		Balancer:               &kafka.LeastBytes{},
	}
}
