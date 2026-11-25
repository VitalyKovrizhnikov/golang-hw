package memorystorage

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

type Storage struct {
	mu            sync.RWMutex
	events        map[string]storage.Event
	notifications map[string]storage.Notification
}

var _ storage.Storage = (*Storage)(nil)

func New() *Storage {
	return &Storage{
		events:        make(map[string]storage.Event),
		notifications: make(map[string]storage.Notification),
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

func (s *Storage) ListEventsForNotification(ctx context.Context, now time.Time) ([]storage.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]storage.Event, 0, len(s.events))
	for _, event := range s.events {
		if event.NotifyBefore <= 0 || !event.NotifiedAt.IsZero() {
			continue
		}
		if !event.StartAt.After(now) {
			continue
		}
		if event.StartAt.Add(-event.NotifyBefore).After(now) {
			continue
		}

		events = append(events, event)
	}

	sortEvents(events)

	return events, nil
}

func (s *Storage) MarkEventNotified(ctx context.Context, id string, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.events[id]
	if !ok {
		return storage.ErrEventNotFound
	}

	event.NotifiedAt = at
	s.events[id] = event

	return nil
}

func (s *Storage) DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64
	for id, event := range s.events {
		if event.EndAt.Before(before) {
			delete(s.events, id)
			deleted++
		}
	}

	return deleted, nil
}

func (s *Storage) SaveNotification(ctx context.Context, notification storage.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := notificationKey(notification.EventID, notification.UserID)
	if _, ok := s.notifications[key]; ok {
		return nil
	}

	s.notifications[key] = notification

	return nil
}

func (s *Storage) ListNotifications(ctx context.Context) ([]storage.Notification, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	notifications := make([]storage.Notification, 0, len(s.notifications))
	for _, notification := range s.notifications {
		notifications = append(notifications, notification)
	}

	sort.Slice(notifications, func(i, j int) bool {
		if notifications[i].EventID == notifications[j].EventID {
			return notifications[i].UserID < notifications[j].UserID
		}

		return notifications[i].EventID < notifications[j].EventID
	})

	return notifications, nil
}

func notificationKey(eventID, userID string) string {
	return eventID + "\x00" + userID
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
