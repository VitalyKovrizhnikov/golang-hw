package app

import (
	"context"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

type Logger interface {
	Debug(msg string)
	Info(msg string)
	Error(msg string)
}

type Metrics interface {
	EventCreated()
	EventUpdated()
	EventDeleted()
	EventsListed(count int)
	EventOperationFailed(operation string)
}

type App struct {
	logger  Logger
	storage storage.Storage
	metrics Metrics
}

func New(logger Logger, store storage.Storage, metrics Metrics) *App {
	return &App{
		logger:  logger,
		storage: store,
		metrics: metrics,
	}
}

func (a *App) CreateEvent(ctx context.Context, event storage.Event) error {
	if err := a.storage.CreateEvent(ctx, event); err != nil {
		a.metrics.EventOperationFailed("create")
		a.logger.Error("failed to create event: " + err.Error())
		return err
	}

	a.metrics.EventCreated()
	a.logger.Debug("event created: " + event.ID)

	return nil
}

func (a *App) UpdateEvent(ctx context.Context, id string, event storage.Event) error {
	if err := a.storage.UpdateEvent(ctx, id, event); err != nil {
		a.metrics.EventOperationFailed("update")
		a.logger.Error("failed to update event: " + err.Error())
		return err
	}

	a.metrics.EventUpdated()
	a.logger.Debug("event updated: " + id)

	return nil
}

func (a *App) DeleteEvent(ctx context.Context, id string) error {
	if err := a.storage.DeleteEvent(ctx, id); err != nil {
		a.metrics.EventOperationFailed("delete")
		a.logger.Error("failed to delete event: " + err.Error())
		return err
	}

	a.metrics.EventDeleted()
	a.logger.Debug("event deleted: " + id)

	return nil
}

func (a *App) ListEvents(ctx context.Context) ([]storage.Event, error) {
	events, err := a.storage.ListEvents(ctx)
	if err != nil {
		a.metrics.EventOperationFailed("list_all")
		a.logger.Error("failed to list events: " + err.Error())
		return nil, err
	}

	a.metrics.EventsListed(len(events))

	return events, nil
}

func (a *App) ListEventsByPeriod(ctx context.Context, userID string, from, to time.Time) ([]storage.Event, error) {
	events, err := a.storage.ListEventsByPeriod(ctx, userID, from, to)
	if err != nil {
		a.metrics.EventOperationFailed("list")
		a.logger.Error("failed to list events: " + err.Error())
		return nil, err
	}

	a.metrics.EventsListed(len(events))

	return events, nil
}
