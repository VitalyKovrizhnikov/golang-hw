package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

type fakeLogger struct{}

func (fakeLogger) Debug(string) {}
func (fakeLogger) Info(string)  {}
func (fakeLogger) Error(string) {}

type fakeMetrics struct {
	created int
	updated int
	deleted int
	listed  int
	errors  []string
}

func (m *fakeMetrics) EventCreated() {
	m.created++
}

func (m *fakeMetrics) EventUpdated() {
	m.updated++
}

func (m *fakeMetrics) EventDeleted() {
	m.deleted++
}

func (m *fakeMetrics) EventsListed(count int) {
	m.listed += count
}

func (m *fakeMetrics) EventOperationFailed(operation string) {
	m.errors = append(m.errors, operation)
}

type fakeStorage struct {
	storage.Storage

	createErr error
	updateErr error
	deleteErr error
	listErr   error
	events    []storage.Event
}

func (s *fakeStorage) CreateEvent(context.Context, storage.Event) error {
	return s.createErr
}

func (s *fakeStorage) UpdateEvent(context.Context, string, storage.Event) error {
	return s.updateErr
}

func (s *fakeStorage) DeleteEvent(context.Context, string) error {
	return s.deleteErr
}

func (s *fakeStorage) ListEventsByPeriod(context.Context, string, time.Time, time.Time) ([]storage.Event, error) {
	return s.events, s.listErr
}

func TestAppRecordsSuccessfulOperations(t *testing.T) {
	metrics := &fakeMetrics{}
	store := &fakeStorage{events: []storage.Event{{ID: "event-1"}, {ID: "event-2"}}}
	app := New(fakeLogger{}, store, metrics)
	ctx := context.Background()

	if err := app.CreateEvent(ctx, storage.Event{ID: "event-1"}); err != nil {
		t.Fatal(err)
	}
	if err := app.UpdateEvent(ctx, "event-1", storage.Event{ID: "event-1"}); err != nil {
		t.Fatal(err)
	}
	if err := app.DeleteEvent(ctx, "event-1"); err != nil {
		t.Fatal(err)
	}
	events, err := app.ListEventsByPeriod(ctx, "user-1", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}

	if metrics.created != 1 || metrics.updated != 1 || metrics.deleted != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
	if metrics.listed != 2 {
		t.Fatalf("listed = %d, want 2", metrics.listed)
	}
	if len(metrics.errors) != 0 {
		t.Fatalf("errors = %v", metrics.errors)
	}
}

func TestAppRecordsFailedOperations(t *testing.T) {
	operationErr := errors.New("storage is unavailable")
	metrics := &fakeMetrics{}
	store := &fakeStorage{
		createErr: operationErr,
		updateErr: operationErr,
		deleteErr: operationErr,
		listErr:   operationErr,
	}
	app := New(fakeLogger{}, store, metrics)
	ctx := context.Background()

	if err := app.CreateEvent(ctx, storage.Event{}); !errors.Is(err, operationErr) {
		t.Fatalf("create error = %v", err)
	}
	if err := app.UpdateEvent(ctx, "event-1", storage.Event{}); !errors.Is(err, operationErr) {
		t.Fatalf("update error = %v", err)
	}
	if err := app.DeleteEvent(ctx, "event-1"); !errors.Is(err, operationErr) {
		t.Fatalf("delete error = %v", err)
	}
	if _, err := app.ListEventsByPeriod(ctx, "user-1", time.Time{}, time.Time{}); !errors.Is(err, operationErr) {
		t.Fatalf("list error = %v", err)
	}

	want := []string{"create", "update", "delete", "list"}
	if len(metrics.errors) != len(want) {
		t.Fatalf("errors = %v, want %v", metrics.errors, want)
	}
	for i := range want {
		if metrics.errors[i] != want[i] {
			t.Fatalf("errors = %v, want %v", metrics.errors, want)
		}
	}
	if metrics.created != 0 || metrics.updated != 0 || metrics.deleted != 0 || metrics.listed != 0 {
		t.Fatalf("success metrics = %+v", metrics)
	}
}
