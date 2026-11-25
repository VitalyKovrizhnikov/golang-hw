package kafkabroker

import (
	"context"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/broker"
	"github.com/segmentio/kafka-go"
)

type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

type Producer struct {
	writer *kafka.Writer
}

var _ broker.Producer = (*Producer)(nil)

func NewProducer(config Config) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(config.Brokers...),
			Topic:                  config.Topic,
			Balancer:               &kafka.LeastBytes{},
			RequiredAcks:           kafka.RequireAll,
			AllowAutoTopicCreation: true,
			BatchTimeout:           10 * time.Millisecond,
		},
	}
}

func (p *Producer) Send(ctx context.Context, key string, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

type Consumer struct {
	reader *kafka.Reader
}

var _ broker.Consumer = (*Consumer)(nil)

func NewConsumer(config Config) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     config.Brokers,
			GroupID:     config.GroupID,
			Topic:       config.Topic,
			MinBytes:    1,
			MaxBytes:    10e6,
			StartOffset: kafka.FirstOffset,
		}),
	}
}

func (c *Consumer) Receive(ctx context.Context) (broker.Message, error) {
	message, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return broker.Message{}, err
	}

	return broker.Message{
		Key:   message.Key,
		Value: message.Value,
		Ack: func(ctx context.Context) error {
			return c.reader.CommitMessages(ctx, message)
		},
	}, nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
