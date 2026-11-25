package internalhttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type logRecorder struct {
	messages []string
}

func (l *logRecorder) Info(msg string) {
	l.messages = append(l.messages, msg)
}

func (l *logRecorder) Error(msg string) {
	l.messages = append(l.messages, msg)
}

func TestHelloHandler(t *testing.T) {
	logger := &logRecorder{}
	handler := loggingMiddleware(logger, http.HandlerFunc(helloHandler))
	request := httptest.NewRequest(http.MethodGet, "/hello?q=1", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "hello, calendar!") {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestLoggingMiddleware(t *testing.T) {
	logger := &logRecorder{}
	handler := loggingMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))

	request := httptest.NewRequest(http.MethodPost, "/events?q=1", nil)
	request.RemoteAddr = "203.0.113.10:12345"
	request.Header.Set("User-Agent", "test-agent/1.0")

	handler.ServeHTTP(httptest.NewRecorder(), request)

	if len(logger.messages) != 1 {
		t.Fatalf("got %d log messages, want 1", len(logger.messages))
	}

	message := logger.messages[0]
	expectedParts := []string{
		"203.0.113.10",
		"POST",
		"/events?q=1",
		"HTTP/1.1",
		"201",
		"test-agent/1.0",
	}
	for _, part := range expectedParts {
		if !strings.Contains(message, part) {
			t.Errorf("log message %q does not contain %q", message, part)
		}
	}
}
