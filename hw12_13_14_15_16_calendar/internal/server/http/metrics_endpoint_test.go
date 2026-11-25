package internalhttp

import (
	"net/http"
	"strings"
	"testing"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/metrics"
)

func TestMetricsEndpointAndHTTPMetrics(t *testing.T) {
	app := &fakeApp{}
	handler := newHandler(&logRecorder{}, app, metrics.New())

	response := doRequest(t, handler, http.MethodGet, "/events", testUserID, "")
	if response.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", response.Code, http.StatusOK)
	}

	response = doRequest(t, handler, http.MethodGet, "/metrics", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, part := range []string{
		"calendar_http_requests_total",
		`route="/events"`,
		`status="200"`,
		"calendar_http_request_duration_seconds",
		"calendar_http_requests_in_flight",
		"calendar_events_created_total",
		"calendar_events_updated_total",
		"calendar_events_deleted_total",
		"calendar_events_listed_total",
	} {
		if !strings.Contains(body, part) {
			t.Errorf("metrics body does not contain %q", part)
		}
	}

	if strings.Contains(body, `route="/metrics"`) {
		t.Error("metrics endpoint must not instrument itself")
	}
}
