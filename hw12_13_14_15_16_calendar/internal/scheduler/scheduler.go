package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/retry"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

const (
	defaultInterval     = 10 * time.Second
	defaultRetention    = 365 * 24 * time.Hour
	defaultRetryInitial = time.Second
	defaultRetryMax     = 30 * time.Second
)

type Logger interface {
	Info(msg string)
	Error(msg string)
}

type Storage interface {
	ListEventsForNotification(ctx context.Context, now time.Time) ([]storage.Event, error)
	MarkEventNotified(ctx context.Context, id string, at time.Time) error
	DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error)
}

type Producer interface {
	Send(ctx context.Context, key string, value []byte) error
}

type Options struct {
	Interval     time.Duration
	Retention    time.Duration
	RetryInitial time.Duration
	RetryMax     time.Duration
}

type Scheduler struct {
	logger   Logger
	storage  Storage
	producer Producer
	options  Options
}

func New(logger Logger, storage Storage, producer Producer, options Options) *Scheduler {
	if options.Interval <= 0 {
		options.Interval = defaultInterval
	}
	if options.Retention <= 0 {
		options.Retention = defaultRetention
	}
	if options.RetryInitial <= 0 {
		options.RetryInitial = defaultRetryInitial
	}
	if options.RetryMax <= 0 {
		options.RetryMax = defaultRetryMax
	}

	return &Scheduler{
		logger:   logger,
		storage:  storage,
		producer: producer,
		options:  options,
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.options.Interval)
	defer ticker.Stop()

	for {
		if err := s.process(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			s.logger.Error("scheduler cycle failed: " + err.Error())
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) process(ctx context.Context) error {
	now := time.Now().UTC()

	events, err := s.storage.ListEventsForNotification(ctx, now)
	if err != nil {
		return fmt.Errorf("list events for notification: %w", err)
	}

	for _, event := range events {
		if err := s.notify(ctx, event); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			s.logger.Error("failed to send notification for event " + event.ID + ": " + err.Error())
			continue
		}

		if err := s.storage.MarkEventNotified(ctx, event.ID, now); err != nil {
			s.logger.Error("failed to mark event " + event.ID + " as notified: " + err.Error())
			continue
		}

		s.logger.Info("notification sent for event " + event.ID)
	}

	cutoff := now.Add(-s.options.Retention)
	deleted, err := s.storage.DeleteEventsBefore(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("delete old events: %w", err)
	}
	if deleted > 0 {
		s.logger.Info(fmt.Sprintf("deleted %d old events", deleted))
	}

	return nil
}

func (s *Scheduler) notify(ctx context.Context, event storage.Event) error {
	notification := storage.Notification{
		EventID: event.ID,
		Title:   event.Title,
		StartAt: event.StartAt,
		UserID:  event.UserID,
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	return retry.Do(ctx, s.logger, retry.Options{
		Initial: s.options.RetryInitial,
		Max:     s.options.RetryMax,
	}, func() error {
		return s.producer.Send(ctx, event.ID, payload)
	})
}
