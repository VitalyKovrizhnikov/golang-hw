package memorystorage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

const (
	firstUserID   = "user-1"
	secondUserID  = "user-2"
	firstEventID  = "event-1"
	secondEventID = "event-2"
	missingID     = "missing"
	eventTitle    = "meeting"
)

func at(hour, minute int) time.Time {
	return time.Date(2024, time.January, 1, hour, minute, 0, 0, time.UTC)
}

func testEvent(id, userID string, start time.Time) storage.Event {
	return storage.Event{
		ID:      id,
		Title:   eventTitle,
		StartAt: start,
		EndAt:   start.Add(time.Hour),
		UserID:  userID,
	}
}

func TestCreateAndList(t *testing.T) {
	store := New()
	ctx := context.Background()

	first := testEvent(firstEventID, firstUserID, at(10, 0))
	second := testEvent(secondEventID, firstUserID, at(12, 0))
	second.Description = "details"
	second.NotifyBefore = 15 * time.Minute

	if err := store.CreateEvent(ctx, first); err != nil {
		t.Fatalf("create first event: %v", err)
	}
	if err := store.CreateEvent(ctx, second); err != nil {
		t.Fatalf("create second event: %v", err)
	}

	events, err := store.ListEvents(ctx)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0] != first || events[1] != second {
		t.Fatalf("events = %+v, want [%+v %+v]", events, first, second)
	}
}

func TestCreateDuplicateAndBusyDate(t *testing.T) {
	store := New()
	ctx := context.Background()
	base := testEvent(firstEventID, firstUserID, at(10, 0))

	if err := store.CreateEvent(ctx, base); err != nil {
		t.Fatalf("create base event: %v", err)
	}

	duplicate := testEvent(firstEventID, secondUserID, at(18, 0))
	if err := store.CreateEvent(ctx, duplicate); !errors.Is(err, storage.ErrEventExists) {
		t.Fatalf("duplicate error = %v, want %v", err, storage.ErrEventExists)
	}

	overlap := testEvent(secondEventID, firstUserID, at(10, 30))
	if err := store.CreateEvent(ctx, overlap); !errors.Is(err, storage.ErrDateBusy) {
		t.Fatalf("overlap error = %v, want %v", err, storage.ErrDateBusy)
	}

	otherUser := testEvent(secondEventID, secondUserID, at(10, 30))
	if err := store.CreateEvent(ctx, otherUser); err != nil {
		t.Fatalf("other user event: %v", err)
	}

	adjacent := testEvent("event-3", firstUserID, at(11, 0))
	if err := store.CreateEvent(ctx, adjacent); err != nil {
		t.Fatalf("adjacent event: %v", err)
	}
}

func TestUpdateEvent(t *testing.T) {
	store := New()
	ctx := context.Background()

	first := testEvent(firstEventID, firstUserID, at(10, 0))
	second := testEvent(secondEventID, firstUserID, at(12, 0))
	if err := store.CreateEvent(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateEvent(ctx, second); err != nil {
		t.Fatal(err)
	}

	busyUpdate := testEvent("other-id", firstUserID, at(11, 30))
	busyUpdate.EndAt = at(12, 30)
	if err := store.UpdateEvent(ctx, firstEventID, busyUpdate); !errors.Is(err, storage.ErrDateBusy) {
		t.Fatalf("busy update error = %v, want %v", err, storage.ErrDateBusy)
	}

	updated := testEvent("other-id", firstUserID, at(10, 15))
	updated.EndAt = at(10, 45)
	updated.Title = "updated"
	if err := store.UpdateEvent(ctx, firstEventID, updated); err != nil {
		t.Fatalf("update event: %v", err)
	}

	events, err := store.ListEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if events[0].ID != firstEventID || events[0].Title != "updated" {
		t.Fatalf("updated event = %+v", events[0])
	}

	missing := testEvent(missingID, firstUserID, at(20, 0))
	if err := store.UpdateEvent(ctx, missingID, missing); !errors.Is(err, storage.ErrEventNotFound) {
		t.Fatalf("missing update error = %v, want %v", err, storage.ErrEventNotFound)
	}
}

func TestDeleteEvent(t *testing.T) {
	store := New()
	ctx := context.Background()
	event := testEvent(firstEventID, firstUserID, at(10, 0))

	if err := store.CreateEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteEvent(ctx, firstEventID); err != nil {
		t.Fatalf("delete event: %v", err)
	}

	events, err := store.ListEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events after delete, want 0", len(events))
	}

	if err := store.DeleteEvent(ctx, firstEventID); !errors.Is(err, storage.ErrEventNotFound) {
		t.Fatalf("second delete error = %v, want %v", err, storage.ErrEventNotFound)
	}
}

func TestListEventsReturnsCopy(t *testing.T) {
	store := New()
	ctx := context.Background()
	event := testEvent(firstEventID, firstUserID, at(10, 0))

	if err := store.CreateEvent(ctx, event); err != nil {
		t.Fatal(err)
	}

	events, err := store.ListEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	events[0].Title = "changed"

	events, err = store.ListEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if events[0].Title != eventTitle {
		t.Fatalf("stored title changed to %q", events[0].Title)
	}
}

func TestCanceledContext(t *testing.T) {
	store := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event := testEvent(firstEventID, firstUserID, at(10, 0))
	if err := store.CreateEvent(ctx, event); !errors.Is(err, context.Canceled) {
		t.Fatalf("create error = %v, want context.Canceled", err)
	}

	if _, err := store.ListEvents(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("list error = %v, want context.Canceled", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := New()
	const (
		writers = 100
		readers = 10
	)

	errCh := make(chan error, writers+readers)
	var wg sync.WaitGroup

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			event := testEvent(
				fmt.Sprintf("event-%d", i),
				fmt.Sprintf("user-%d", i),
				at(0, 0).Add(time.Duration(i)*time.Hour),
			)
			if err := store.CreateEvent(context.Background(), event); err != nil {
				errCh <- err
			}
		}(i)
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.ListEvents(context.Background()); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent error: %v", err)
	}

	events, err := store.ListEvents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != writers {
		t.Fatalf("got %d events, want %d", len(events), writers)
	}
}
