package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestBusinessAndHTTPMetrics(t *testing.T) {
	m := New()

	m.EventCreated()
	m.EventUpdated()
	m.EventDeleted()
	m.EventsListed(3)
	m.EventOperationFailed("create")
	m.RequestStarted()
	m.RequestFinished(http.MethodGet, "/events", http.StatusOK, 25*time.Millisecond)

	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{name: "created", value: testutil.ToFloat64(m.eventsCreated), want: 1},
		{name: "updated", value: testutil.ToFloat64(m.eventsUpdated), want: 1},
		{name: "deleted", value: testutil.ToFloat64(m.eventsDeleted), want: 1},
		{name: "listed", value: testutil.ToFloat64(m.eventsListed), want: 3},
		{
			name:  "operation errors",
			value: testutil.ToFloat64(m.operationErrors.WithLabelValues("create")),
			want:  1,
		},
		{name: "in flight", value: testutil.ToFloat64(m.httpInFlight), want: 0},
		{
			name:  "http requests",
			value: testutil.ToFloat64(m.httpRequests.WithLabelValues(http.MethodGet, "/events", "200")),
			want:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.value != test.want {
				t.Fatalf("value = %v, want %v", test.value, test.want)
			}
		})
	}

	durationCount := testutil.CollectAndCount(
		m.httpRequestDuration,
		"calendar_http_request_duration_seconds",
	)
	if durationCount != 1 {
		t.Fatalf("duration metric count = %d, want 1", durationCount)
	}
}

func TestMetricsHandler(t *testing.T) {
	m := New()
	m.RequestStarted()
	m.RequestFinished(http.MethodGet, "/events", http.StatusOK, time.Millisecond)
	m.EventOperationFailed("create")

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()

	m.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, name := range []string{
		"calendar_http_requests_total",
		"calendar_http_request_duration_seconds",
		"calendar_http_requests_in_flight",
		"calendar_events_created_total",
		"calendar_events_updated_total",
		"calendar_events_deleted_total",
		"calendar_events_listed_total",
		"calendar_event_operation_errors_total",
	} {
		if !strings.Contains(body, name) {
			t.Errorf("metrics body does not contain %q", name)
		}
	}
}
