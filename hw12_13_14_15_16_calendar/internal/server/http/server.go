package internalhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/metrics"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/server/http/openapi"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Logger interface {
	Info(msg string)
	Error(msg string)
}

type Application interface {
	CreateEvent(ctx context.Context, event storage.Event) error
	UpdateEvent(ctx context.Context, id string, event storage.Event) error
	DeleteEvent(ctx context.Context, id string) error
	ListEventsByPeriod(ctx context.Context, userID string, from, to time.Time) ([]storage.Event, error)
}

type Server struct {
	server *http.Server
}

func NewServer(logger Logger, application Application, promMetrics *metrics.Metrics, addr string) *Server {
	return &Server{
		server: &http.Server{
			Addr:              addr,
			Handler:           newHandler(logger, application, promMetrics),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func newHandler(logger Logger, application Application, promMetrics *metrics.Metrics) http.Handler {
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return metricsMiddleware(promMetrics, next)
	})
	router.Get("/", helloHandler)
	router.Get("/hello", helloHandler)
	router.Handle("/metrics", promMetrics.Handler())

	openapi.HandlerWithOptions(newAPI(application), openapi.ChiServerOptions{
		BaseRouter: router,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeError(w, http.StatusBadRequest, err.Error())
		},
	})

	return loggingMiddleware(logger, router)
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := s.Stop(shutdownCtx); err != nil {
			return err
		}

		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	}
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func helloHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintln(w, "hello, calendar!")
}
