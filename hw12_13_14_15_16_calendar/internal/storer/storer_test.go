package storer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/broker"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

const (
	testEventID = "event-1"
	testUserID  = "user-1"
	testTitle   = "meeting"
)

type fakeLogger struct {
	infos  []string
	errors []string
}

func (l *fakeLogger) Info(msg string) {
	l.infos = append(l.infos, msg)
}

func (l *fakeLogger) Error(msg string) {
	l.errors = append(l.errors, msg)
}

type fakeStorage struct {
	failures int
	attempts int
	saved    []storage.Notification
}

func (s *fakeStorage) SaveNotification(_ context.Context, notification storage.Notification) error {
	s.attempts++
	if s.attempts <= s.failures {
		return errors.New("database is unavailable")
	}

	s.saved = append(s.saved, notification)

	return nil
}

func notificationPayload(t *testing.T, notification storage.Notification) []byte {
	t.Helper()

	payload, err := json.Marshal(notification)
	if err != nil {
		t.Fatal(err)
	}

	return payload
}

func TestHandleStoresNotification(t *testing.T) {
	store := &fakeStorage{}
	logger := &fakeLogger{}
	storer := New(logger, store, nil, Options{})
	acked := 0
	message := broker.Message{
		Value: notificationPayload(t, storage.Notification{
			EventID: testEventID,
			Title:   testTitle,
			StartAt: time.Date(2024, time.January, 1, 10, 0, 0, 0, time.UTC),
			UserID:  testUserID,
		}),
		Ack: func(context.Context) error {
			acked++
			return nil
		},
	}

	if err := storer.handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(store.saved) != 1 || store.saved[0].EventID != testEventID {
		t.Fatalf("saved = %+v", store.saved)
	}
	if acked != 1 {
		t.Fatalf("acked = %d, want 1", acked)
	}
	if len(logger.infos) != 1 {
		t.Fatalf("infos = %v", logger.infos)
	}
}

func TestHandleInvalidNotification(t *testing.T) {
	tests := []struct {
		name  string
		value []byte
	}{
		{name: "invalid json", value: []byte("{")},
		{name: "empty fields", value: []byte(`{"title":"meeting"}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeStorage{}
			storer := New(&fakeLogger{}, store, nil, Options{})
			acked := 0
			message := broker.Message{
				Value: test.value,
				Ack: func(context.Context) error {
					acked++
					return nil
				},
			}

			if err := storer.handle(context.Background(), message); err != nil {
				t.Fatal(err)
			}
			if len(store.saved) != 0 {
				t.Fatalf("saved = %+v", store.saved)
			}
			if acked != 1 {
				t.Fatalf("acked = %d, want 1", acked)
			}
		})
	}
}

func TestHandleRetriesSave(t *testing.T) {
	store := &fakeStorage{failures: 2}
	logger := &fakeLogger{}
	storer := New(logger, store, nil, Options{
		RetryInitial: time.Millisecond,
		RetryMax:     time.Millisecond,
	})
	message := broker.Message{
		Value: notificationPayload(t, storage.Notification{
			EventID: testEventID,
			Title:   testTitle,
			UserID:  testUserID,
		}),
	}

	if err := storer.handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if store.attempts != 3 {
		t.Fatalf("attempts = %d, want 3", store.attempts)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved = %+v", store.saved)
	}
	if len(logger.errors) != 2 {
		t.Fatalf("errors = %v", logger.errors)
	}
}

func TestHandleStopsOnCanceledContext(t *testing.T) {
	store := &fakeStorage{failures: 1000}
	storer := New(&fakeLogger{}, store, nil, Options{
		RetryInitial: time.Millisecond,
		RetryMax:     time.Millisecond,
	})
	message := broker.Message{
		Value: notificationPayload(t, storage.Notification{
			EventID: testEventID,
			Title:   testTitle,
			UserID:  testUserID,
		}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := storer.handle(ctx, message)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if len(store.saved) != 0 {
		t.Fatalf("saved = %+v", store.saved)
	}
}
