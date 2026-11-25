package internalhttp

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/metrics"
	"github.com/go-chi/chi/v5"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	return w.ResponseWriter.Write(data)
}

func metricsMiddleware(promMetrics *metrics.Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		promMetrics.RequestStarted()
		writer := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		defer func() {
			route := "unknown"
			routeContext := chi.RouteContext(r.Context())
			if routeContext != nil && routeContext.RoutePattern() != "" {
				route = routeContext.RoutePattern()
			}

			promMetrics.RequestFinished(r.Method, route, writer.status, time.Since(start))
		}()

		next.ServeHTTP(writer, r)
	})
}

func loggingMiddleware(logger Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(writer, r)

		logger.Info(fmt.Sprintf(
			"%s [%s] %s %s %s %d %s %q",
			clientIP(r),
			start.Format("02/Jan/2006:15:04:05 -0700"),
			r.Method,
			r.URL.RequestURI(),
			r.Proto,
			writer.status,
			time.Since(start).String(),
			r.UserAgent(),
		))
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
