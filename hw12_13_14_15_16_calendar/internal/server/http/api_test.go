package internalhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/server/http/openapi"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
)

const (
	testUserID  = "user-1"
	testEventID = "event-1"
)

type fakeApp struct {
	createErr error
	updateErr error
	deleteErr error
	listErr   error

	created storage.Event
	updated storage.Event
	deleted string

	listUserID string
	listFrom   time.Time
	listTo     time.Time
	events     []storage.Event
}

func (f *fakeApp) CreateEvent(_ context.Context, event storage.Event) error {
	f.created = event
	return f.createErr
}

func (f *fakeApp) UpdateEvent(_ context.Context, _ string, event storage.Event) error {
	f.updated = event
	return f.updateErr
}

func (f *fakeApp) DeleteEvent(_ context.Context, id string) error {
	f.deleted = id
	return f.deleteErr
}

func (f *fakeApp) ListEventsByPeriod(_ context.Context, userID string, from, to time.Time) ([]storage.Event, error) {
	f.listUserID = userID
	f.listFrom = from
	f.listTo = to
	return f.events, f.listErr
}

func doRequest(t *testing.T, handler http.Handler, method, target, userID, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if userID != "" {
		request.Header.Set("X-User-ID", userID)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}

func TestCreateEvent(t *testing.T) {
	app := &fakeApp{}
	handler := newHandler(&logRecorder{}, app)

	body := `{"title":"meeting","description":"sync",` +
		`"startAt":"2024-01-01T10:00:00Z","endAt":"2024-01-01T11:00:00Z",` +
		`"notifyBefore":"15m"}`
	response := doRequest(t, handler, http.MethodPost, "/events", testUserID, body)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var event openapi.Event
	if err := json.NewDecoder(response.Body).Decode(&event); err != nil {
		t.Fatal(err)
	}
	if event.Id == "" {
		t.Fatal("event id is empty")
	}
	if app.created.ID != event.Id || app.created.UserID != testUserID {
		t.Fatalf("created event = %+v", app.created)
	}
	if app.created.NotifyBefore != 15*time.Minute {
		t.Fatalf("notifyBefore = %s, want 15m", app.created.NotifyBefore)
	}
	if response.Header().Get("Location") != "/events/"+event.Id {
		t.Fatalf("Location = %q", response.Header().Get("Location"))
	}
}

func TestCreateEventValidation(t *testing.T) {
	validBody := `{"title":"meeting","startAt":"2024-01-01T10:00:00Z","endAt":"2024-01-01T11:00:00Z"}`
	tests := []struct {
		name   string
		userID string
		body   string
		appErr error
		status int
	}{
		{name: "missing user", body: validBody, status: http.StatusBadRequest},
		{name: "invalid json", userID: testUserID, body: "{", status: http.StatusBadRequest},
		{
			name:   "empty title",
			userID: testUserID,
			body:   `{"title":" ","startAt":"2024-01-01T10:00:00Z","endAt":"2024-01-01T11:00:00Z"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "bad period",
			userID: testUserID,
			body:   `{"title":"meeting","startAt":"2024-01-01T11:00:00Z","endAt":"2024-01-01T10:00:00Z"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "bad duration",
			userID: testUserID,
			body:   `{"title":"meeting","startAt":"2024-01-01T10:00:00Z","endAt":"2024-01-01T11:00:00Z","notifyBefore":"soon"}`,
			status: http.StatusBadRequest,
		},
		{name: "busy", userID: testUserID, body: validBody, appErr: storage.ErrDateBusy, status: http.StatusConflict},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := &fakeApp{createErr: test.appErr}
			handler := newHandler(&logRecorder{}, app)
			response := doRequest(t, handler, http.MethodPost, "/events", test.userID, test.body)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, test.status, response.Body.String())
			}
		})
	}
}

func TestListEvents(t *testing.T) {
	from := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	app := &fakeApp{
		events: []storage.Event{{
			ID:      testEventID,
			Title:   "meeting",
			StartAt: from.Add(10 * time.Hour),
			EndAt:   from.Add(11 * time.Hour),
			UserID:  testUserID,
		}},
	}
	handler := newHandler(&logRecorder{}, app)

	target := "/events?from=2024-01-01T00:00:00Z" +
		"&to=2024-01-02T00:00:00Z"
	response := doRequest(t, handler, http.MethodGet, target, testUserID, "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if app.listUserID != testUserID || !app.listFrom.Equal(from) || !app.listTo.Equal(to) {
		t.Fatalf("list args = user=%q from=%s to=%s", app.listUserID, app.listFrom, app.listTo)
	}

	var events []openapi.Event
	if err := json.NewDecoder(response.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Id != testEventID {
		t.Fatalf("events = %+v", events)
	}

	response = doRequest(t, handler, http.MethodGet, "/events?from=nope", testUserID, "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid from status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestUpdateEvent(t *testing.T) {
	app := &fakeApp{}
	handler := newHandler(&logRecorder{}, app)
	body := `{"title":"updated","startAt":"2024-01-01T12:00:00Z","endAt":"2024-01-01T13:00:00Z"}`

	response := doRequest(t, handler, http.MethodPut, "/events/event-1", testUserID, body)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if app.updated.ID != testEventID || app.updated.UserID != testUserID || app.updated.Title != "updated" {
		t.Fatalf("updated event = %+v", app.updated)
	}

	app.updateErr = storage.ErrEventNotFound
	response = doRequest(t, handler, http.MethodPut, "/events/missing", testUserID, body)
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestDeleteEvent(t *testing.T) {
	app := &fakeApp{}
	handler := newHandler(&logRecorder{}, app)

	response := doRequest(t, handler, http.MethodDelete, "/events/event-1", testUserID, "")
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusNoContent, response.Body.String())
	}
	if app.deleted != testEventID {
		t.Fatalf("deleted = %q", app.deleted)
	}

	app.deleteErr = errors.New("boom")
	response = doRequest(t, handler, http.MethodDelete, "/events/event-1", testUserID, "")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("error status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
