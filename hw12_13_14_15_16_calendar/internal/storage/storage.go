package storage

import (
	"context"
	"errors"
	"time"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrEventExists   = errors.New("event already exists")
	ErrDateBusy      = errors.New("date is busy")
)

type Storage interface {
	CreateEvent(ctx context.Context, event Event) error
	UpdateEvent(ctx context.Context, id string, event Event) error
	DeleteEvent(ctx context.Context, id string) error
	ListEvents(ctx context.Context) ([]Event, error)
	ListEventsByPeriod(ctx context.Context, userID string, from, to time.Time) ([]Event, error)
	ListEventsForNotification(ctx context.Context, now time.Time) ([]Event, error)
	MarkEventNotified(ctx context.Context, id string, at time.Time) error
	DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error)
	SaveNotification(ctx context.Context, notification Notification) error
}
