package memorystorage

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

type Storage struct {
	mu     sync.RWMutex
	events map[string]storage.Event
}

var _ storage.Storage = (*Storage)(nil)

func New() *Storage {
	return &Storage{
		events: make(map[string]storage.Event),
	}
}

func (s *Storage) CreateEvent(ctx context.Context, event storage.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[event.ID]; ok {
		return storage.ErrEventExists
	}
	if s.isDateBusy(event, "") {
		return storage.ErrDateBusy
	}

	s.events[event.ID] = event

	return nil
}

func (s *Storage) UpdateEvent(ctx context.Context, id string, event storage.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[id]; !ok {
		return storage.ErrEventNotFound
	}

	event.ID = id
	if s.isDateBusy(event, id) {
		return storage.ErrDateBusy
	}

	s.events[id] = event

	return nil
}

func (s *Storage) DeleteEvent(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[id]; !ok {
		return storage.ErrEventNotFound
	}

	delete(s.events, id)

	return nil
}

func (s *Storage) ListEvents(ctx context.Context) ([]storage.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]storage.Event, 0, len(s.events))
	for _, event := range s.events {
		events = append(events, event)
	}

	sortEvents(events)

	return events, nil
}

func (s *Storage) ListEventsByPeriod(ctx context.Context, userID string, from, to time.Time) ([]storage.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]storage.Event, 0, len(s.events))
	for _, event := range s.events {
		if event.UserID != userID {
			continue
		}
		if !from.IsZero() && !event.EndAt.After(from) {
			continue
		}
		if !to.IsZero() && !event.StartAt.Before(to) {
			continue
		}

		events = append(events, event)
	}

	sortEvents(events)

	return events, nil
}

func sortEvents(events []storage.Event) {
	sort.Slice(events, func(i, j int) bool {
		if events[i].StartAt.Equal(events[j].StartAt) {
			return events[i].ID < events[j].ID
		}

		return events[i].StartAt.Before(events[j].StartAt)
	})
}

func (s *Storage) isDateBusy(event storage.Event, excludeID string) bool {
	for id, existing := range s.events {
		if id == excludeID || existing.UserID != event.UserID {
			continue
		}
		if event.StartAt.Before(existing.EndAt) && existing.StartAt.Before(event.EndAt) {
			return true
		}
	}

	return false
}
