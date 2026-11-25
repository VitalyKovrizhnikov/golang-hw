package broker

import "context"

type Message struct {
	Key   []byte
	Value []byte
	Ack   func(ctx context.Context) error
}

type Producer interface {
	Send(ctx context.Context, key string, value []byte) error
	Close() error
}

type Consumer interface {
	Receive(ctx context.Context) (Message, error)
	Close() error
}
