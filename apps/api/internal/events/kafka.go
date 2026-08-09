package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/agungardiyanta/UrlShorter/apps/api/internal/domain"
	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
		},
	}
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}

func (p *KafkaPublisher) PublishClick(ctx context.Context, event domain.ClickEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Code),
		Value: payload,
		Time:  event.ClickedAt,
	})
}
