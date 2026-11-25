//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/server/http/openapi"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

const (
	defaultCalendarURL  = "http://localhost:8888"
	defaultPostgresDSN  = "postgres://calendar:calendar@localhost:5432/calendar?sslmode=disable"
	defaultKafkaBroker  = "localhost:9092"
	defaultKafkaTopic   = "calendar.notifications"
	requestTimeout      = 10 * time.Second
	integrationTimeout  = 30 * time.Second
	integrationPollTime = 500 * time.Millisecond
)

type eventInput struct {
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	StartAt      time.Time `json:"startAt"`
	EndAt        time.Time `json:"endAt"`
	NotifyBefore string    `json:"notifyBefore,omitempty"`
}

func TestCreateEventAndErrors(t *testing.T) {
	userID := "it-user-" + uuid.New().String()
	start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	input := eventInput{
		Title:        "integration meeting",
		Description:  "created by integration test",
		StartAt:      start,
		EndAt:        start.Add(time.Hour),
		NotifyBefore: "15m",
	}

	event := createEvent(t, userID, input)
	if event.Id == "" || event.UserId != userID {
		t.Fatalf("created event = %+v", event)
	}

	status, body := apiRequest(t, http.MethodPost, "/events", userID, input)
	if status != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want %d, body = %s", status, http.StatusConflict, body)
	}

	status, body = apiRequest(t, http.MethodPost, "/events", "", input)
	if status != http.StatusBadRequest {
		t.Fatalf("missing user status = %d, want %d, body = %s", status, http.StatusBadRequest, body)
	}

	invalid := input
	invalid.EndAt = start.Add(-time.Hour)
	status, body = apiRequest(t, http.MethodPost, "/events", userID, invalid)
	if status != http.StatusBadRequest {
		t.Fatalf("invalid period status = %d, want %d, body = %s", status, http.StatusBadRequest, body)
	}
}

func TestListEventsPeriods(t *testing.T) {
	userID := "it-list-user-" + uuid.New().String()
	now := time.Now().UTC()
	base := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.UTC).AddDate(0, 0, 2)

	offsets := []time.Duration{0, 48 * time.Hour, 10 * 24 * time.Hour}
	for i, offset := range offsets {
		start := base.Add(offset)
		createEvent(t, userID, eventInput{
			Title:   fmt.Sprintf("integration list %d", i),
			StartAt: start,
			EndAt:   start.Add(time.Hour),
		})
	}

	dayStart := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		from time.Time
		to   time.Time
		want int
	}{
		{name: "day", from: dayStart, to: dayStart.Add(24 * time.Hour), want: 1},
		{name: "week", from: dayStart, to: dayStart.Add(7 * 24 * time.Hour), want: 2},
		{name: "month", from: dayStart, to: dayStart.Add(31 * 24 * time.Hour), want: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := listEvents(t, userID, test.from, test.to)
			if len(events) != test.want {
				t.Fatalf("events = %d, want %d", len(events), test.want)
			}
		})
	}
}

func TestStorerSavesNotification(t *testing.T) {
	eventID := "it-storer-" + uuid.New().String()
	userID := "it-storer-user-" + uuid.New().String()
	notification := storage.Notification{
		EventID: eventID,
		Title:   "storer integration",
		StartAt: time.Now().UTC().Add(time.Hour),
		UserID:  userID,
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		t.Fatal(err)
	}

	brokers := strings.Split(env("KAFKA_BROKERS", defaultKafkaBroker), ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  env("KAFKA_TOPIC", defaultKafkaTopic),
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           10 * time.Millisecond,
	}
	defer func() {
		_ = writer.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	if err := writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(eventID),
		Value: payload,
	}); err != nil {
		t.Fatal(err)
	}

	db := openDB(t)
	waitForNotification(t, db, eventID, notification.Title, notification.UserID)
}

func TestSchedulerSendsNotification(t *testing.T) {
	userID := "it-scheduler-user-" + uuid.New().String()
	title := "scheduler integration"
	start := time.Now().UTC().Add(30 * time.Minute).Truncate(time.Second)
	db := openDB(t)

	event := createEvent(t, userID, eventInput{
		Title:        title,
		StartAt:      start,
		EndAt:        start.Add(time.Hour),
		NotifyBefore: "1h",
	})

	waitForNotification(t, db, event.Id, title, userID)

	var notified sql.NullTime
	if err := db.QueryRowContext(context.Background(),
		`SELECT notified_at FROM events WHERE id = $1`, event.Id).Scan(&notified); err != nil {
		t.Fatal(err)
	}
	if !notified.Valid {
		t.Fatal("event was not marked as notified")
	}
}

func createEvent(t *testing.T, userID string, input eventInput) openapi.Event {
	t.Helper()

	status, body := apiRequest(t, http.MethodPost, "/events", userID, input)
	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body = %s", status, http.StatusCreated, body)
	}

	var event openapi.Event
	if err := json.Unmarshal(body, &event); err != nil {
		t.Fatal(err)
	}

	return event
}

func listEvents(t *testing.T, userID string, from, to time.Time) []openapi.Event {
	t.Helper()

	path := fmt.Sprintf(
		"/events?from=%s&to=%s",
		url.QueryEscape(from.Format(time.RFC3339)),
		url.QueryEscape(to.Format(time.RFC3339)),
	)
	status, body := apiRequest(t, http.MethodGet, path, userID, nil)
	if status != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body = %s", status, http.StatusOK, body)
	}

	var events []openapi.Event
	if err := json.Unmarshal(body, &events); err != nil {
		t.Fatal(err)
	}

	return events
}

func apiRequest(t *testing.T, method, path, userID string, body interface{}) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(
		ctx,
		method,
		env("CALENDAR_URL", defaultCalendarURL)+path,
		reader,
	)
	if err != nil {
		t.Fatal(err)
	}
	if userID != "" {
		request.Header.Set("X-User-ID", userID)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := (&http.Client{Timeout: requestTimeout}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	return response.StatusCode, payload
}

func openDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", env("POSTGRES_DSN", defaultPostgresDSN))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func waitForNotification(t *testing.T, db *sql.DB, eventID, title, userID string) {
	t.Helper()

	deadline := time.Now().Add(integrationTimeout)
	for time.Now().Before(deadline) {
		var gotTitle, gotUser string
		err := db.QueryRowContext(
			context.Background(),
			`SELECT title, user_id FROM notifications WHERE event_id = $1`,
			eventID,
		).Scan(&gotTitle, &gotUser)

		if err == nil {
			if gotTitle != title || gotUser != userID {
				t.Fatalf("notification = %q/%q, want %q/%q", gotTitle, gotUser, title, userID)
			}

			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("query notification: %v", err)
		}

		time.Sleep(integrationPollTime)
	}

	t.Fatalf("notification for event %s was not saved", eventID)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
