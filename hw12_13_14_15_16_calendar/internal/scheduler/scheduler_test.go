package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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

type fakeProducer struct {
	failures int
	attempts int
	keys     []string
	values   [][]byte
}

func (p *fakeProducer) Send(_ context.Context, key string, value []byte) error {
	p.attempts++
	if p.attempts <= p.failures {
		return errors.New("kafka is unavailable")
	}

	p.keys = append(p.keys, key)
	p.values = append(p.values, value)

	return nil
}

type fakeStorage struct {
	events       []storage.Event
	listErr      error
	markErr      error
	deleteErr    error
	deleted      int64
	listNow      time.Time
	deleteBefore time.Time
	marked       map[string]time.Time
}

func (s *fakeStorage) ListEventsForNotification(_ context.Context, now time.Time) ([]storage.Event, error) {
	s.listNow = now
	return s.events, s.listErr
}

func (s *fakeStorage) MarkEventNotified(_ context.Context, id string, at time.Time) error {
	if s.markErr != nil {
		return s.markErr
	}
	if s.marked == nil {
		s.marked = make(map[string]time.Time)
	}
	s.marked[id] = at

	return nil
}

func (s *fakeStorage) DeleteEventsBefore(_ context.Context, before time.Time) (int64, error) {
	s.deleteBefore = before
	return s.deleted, s.deleteErr
}

func TestProcessSendsNotification(t *testing.T) {
	retention := 24 * time.Hour
	startAt := time.Now().UTC().Add(2 * time.Hour)
	store := &fakeStorage{
		events: []storage.Event{{
			ID:           testEventID,
			Title:        testTitle,
			StartAt:      startAt,
			EndAt:        startAt.Add(time.Hour),
			UserID:       testUserID,
			NotifyBefore: time.Hour,
		}},
		deleted: 2,
	}
	producer := &fakeProducer{}
	logger := &fakeLogger{}
	beforeCutoff := time.Now().Add(-retention)

	s := New(logger, store, producer, Options{
		Retention:    retention,
		RetryInitial: time.Millisecond,
		RetryMax:     time.Millisecond,
	})

	if err := s.process(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(producer.values) != 1 || producer.keys[0] != testEventID {
		t.Fatalf("produced keys=%v values=%d", producer.keys, len(producer.values))
	}

	var notification storage.Notification
	if err := json.Unmarshal(producer.values[0], &notification); err != nil {
		t.Fatal(err)
	}
	if notification.EventID != testEventID || notification.Title != testTitle ||
		notification.UserID != testUserID || !notification.StartAt.Equal(startAt) {
		t.Fatalf("notification = %+v", notification)
	}

	if store.marked[testEventID].IsZero() {
		t.Fatal("event was not marked as notified")
	}
	if store.listNow.IsZero() {
		t.Fatal("list time was not set")
	}
	if store.deleteBefore.Before(beforeCutoff.Add(-time.Second)) {
		t.Fatalf("cutoff = %s, too old", store.deleteBefore)
	}
	if len(logger.errors) != 0 {
		t.Fatalf("unexpected errors: %v", logger.errors)
	}
}

func TestProcessRetriesKafka(t *testing.T) {
	startAt := time.Now().UTC().Add(2 * time.Hour)
	store := &fakeStorage{
		events: []storage.Event{{
			ID:           testEventID,
			Title:        testTitle,
			StartAt:      startAt,
			EndAt:        startAt.Add(time.Hour),
			UserID:       testUserID,
			NotifyBefore: time.Hour,
		}},
	}
	producer := &fakeProducer{failures: 2}
	logger := &fakeLogger{}

	s := New(logger, store, producer, Options{
		RetryInitial: time.Millisecond,
		RetryMax:     time.Millisecond,
	})

	if err := s.process(context.Background()); err != nil {
		t.Fatal(err)
	}
	if producer.attempts != 3 {
		t.Fatalf("attempts = %d, want 3", producer.attempts)
	}
	if len(store.marked) != 1 {
		t.Fatalf("marked = %v", store.marked)
	}
	if len(logger.errors) != 2 {
		t.Fatalf("errors = %v", logger.errors)
	}
}

func TestProcessStopsOnCanceledContext(t *testing.T) {
	startAt := time.Now().UTC().Add(2 * time.Hour)
	store := &fakeStorage{
		events: []storage.Event{{
			ID:           testEventID,
			Title:        testTitle,
			StartAt:      startAt,
			EndAt:        startAt.Add(time.Hour),
			UserID:       testUserID,
			NotifyBefore: time.Hour,
		}},
	}
	producer := &fakeProducer{failures: 1000}
	logger := &fakeLogger{}

	s := New(logger, store, producer, Options{
		RetryInitial: time.Millisecond,
		RetryMax:     time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := s.process(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if len(store.marked) != 0 {
		t.Fatalf("marked = %v", store.marked)
	}
}

func TestProcessReturnsStorageError(t *testing.T) {
	store := &fakeStorage{listErr: errors.New("database is unavailable")}
	s := New(&fakeLogger{}, store, &fakeProducer{}, Options{})

	if err := s.process(context.Background()); err == nil {
		t.Fatal("expected storage error")
	}
}
