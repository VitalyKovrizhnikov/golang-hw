package storer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/broker"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/retry"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

const (
	defaultRetryInitial = time.Second
	defaultRetryMax     = 30 * time.Second
)

type Logger interface {
	Info(msg string)
	Error(msg string)
}

type Storage interface {
	SaveNotification(ctx context.Context, notification storage.Notification) error
}

type Consumer interface {
	Receive(ctx context.Context) (broker.Message, error)
}

type Options struct {
	RetryInitial time.Duration
	RetryMax     time.Duration
}

type Storer struct {
	logger   Logger
	storage  Storage
	consumer Consumer
	options  Options
}

func New(logger Logger, storage Storage, consumer Consumer, options Options) *Storer {
	if options.RetryInitial <= 0 {
		options.RetryInitial = defaultRetryInitial
	}
	if options.RetryMax <= 0 {
		options.RetryMax = defaultRetryMax
	}

	return &Storer{
		logger:   logger,
		storage:  storage,
		consumer: consumer,
		options:  options,
	}
}

func (s *Storer) Run(ctx context.Context) error {
	for {
		var message broker.Message
		err := retry.Do(ctx, s.logger, s.retryOptions(), func() error {
			var receiveErr error
			message, receiveErr = s.consumer.Receive(ctx)

			return receiveErr
		})
		if err != nil {
			return err
		}

		if err := s.handle(ctx, message); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			s.logger.Error("failed to process notification: " + err.Error())
		}
	}
}

func (s *Storer) handle(ctx context.Context, message broker.Message) error {
	var notification storage.Notification
	if err := json.Unmarshal(message.Value, &notification); err != nil {
		s.logger.Error("invalid notification: " + err.Error())

		return s.ack(ctx, message)
	}
	if notification.EventID == "" || notification.UserID == "" {
		s.logger.Error("invalid notification: empty event_id or user_id")

		return s.ack(ctx, message)
	}

	err := retry.Do(ctx, s.logger, s.retryOptions(), func() error {
		return s.storage.SaveNotification(ctx, notification)
	})
	if err != nil {
		return err
	}

	if err := s.ack(ctx, message); err != nil {
		return fmt.Errorf("commit notification: %w", err)
	}

	s.logger.Info("notification stored for event " + notification.EventID)

	return nil
}

func (s *Storer) ack(ctx context.Context, message broker.Message) error {
	if message.Ack == nil {
		return nil
	}

	return message.Ack(ctx)
}

func (s *Storer) retryOptions() retry.Options {
	return retry.Options{
		Initial: s.options.RetryInitial,
		Max:     s.options.RetryMax,
	}
}
