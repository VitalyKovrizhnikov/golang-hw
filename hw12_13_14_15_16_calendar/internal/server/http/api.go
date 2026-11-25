package internalhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/server/http/openapi"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
	"github.com/google/uuid"
)

type api struct {
	app Application
}

func newAPI(application Application) *api {
	return &api{app: application}
}

func (a *api) ListEvents(w http.ResponseWriter, r *http.Request, params openapi.ListEventsParams) {
	userID, ok := requireUserID(w, params.XUserID)
	if !ok {
		return
	}

	var from, to time.Time
	if params.From != nil {
		from = *params.From
	}
	if params.To != nil {
		to = *params.To
	}
	if !from.IsZero() && !to.IsZero() && !to.After(from) {
		writeError(w, http.StatusBadRequest, "to must be after from")
		return
	}

	events, err := a.app.ListEventsByPeriod(r.Context(), userID, from, to)
	if err != nil {
		writeAppError(w, err)
		return
	}

	result := make([]openapi.Event, 0, len(events))
	for _, event := range events {
		result = append(result, toAPIEvent(event))
	}

	writeJSON(w, http.StatusOK, result)
}

func (a *api) CreateEvent(w http.ResponseWriter, r *http.Request, params openapi.CreateEventParams) {
	userID, ok := requireUserID(w, params.XUserID)
	if !ok {
		return
	}

	var request openapi.CreateEventJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	event, err := eventFromInput(request, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	event.ID = uuid.New().String()

	if err := a.app.CreateEvent(r.Context(), event); err != nil {
		writeAppError(w, err)
		return
	}

	w.Header().Set("Location", "/events/"+event.ID)
	writeJSON(w, http.StatusCreated, toAPIEvent(event))
}

func (a *api) UpdateEvent(w http.ResponseWriter, r *http.Request, id string, params openapi.UpdateEventParams) {
	userID, ok := requireUserID(w, params.XUserID)
	if !ok {
		return
	}

	var request openapi.UpdateEventJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	event, err := eventFromInput(request, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	event.ID = id

	if err := a.app.UpdateEvent(r.Context(), id, event); err != nil {
		writeAppError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIEvent(event))
}

func (a *api) DeleteEvent(w http.ResponseWriter, r *http.Request, id string, params openapi.DeleteEventParams) {
	if _, ok := requireUserID(w, params.XUserID); !ok {
		return
	}

	if err := a.app.DeleteEvent(r.Context(), id); err != nil {
		writeAppError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func requireUserID(w http.ResponseWriter, userID string) (string, bool) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		writeError(w, http.StatusBadRequest, "X-User-ID header is required")
		return "", false
	}

	return userID, true
}

func eventFromInput(input openapi.EventInput, userID string) (storage.Event, error) {
	if strings.TrimSpace(input.Title) == "" {
		return storage.Event{}, errors.New("title is required")
	}
	if !input.EndAt.After(input.StartAt) {
		return storage.Event{}, errors.New("endAt must be after startAt")
	}

	notifyBefore, err := parseNotifyBefore(input.NotifyBefore)
	if err != nil {
		return storage.Event{}, err
	}

	event := storage.Event{
		Title:        input.Title,
		StartAt:      input.StartAt,
		EndAt:        input.EndAt,
		UserID:       userID,
		NotifyBefore: notifyBefore,
	}
	if input.Description != nil {
		event.Description = *input.Description
	}

	return event, nil
}

func parseNotifyBefore(value *string) (time.Duration, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return 0, nil
	}

	duration, err := time.ParseDuration(*value)
	if err != nil {
		return 0, fmt.Errorf("invalid notifyBefore: %w", err)
	}
	if duration < 0 {
		return 0, errors.New("notifyBefore must not be negative")
	}

	return duration, nil
}

func toAPIEvent(event storage.Event) openapi.Event {
	result := openapi.Event{
		Id:      event.ID,
		Title:   event.Title,
		StartAt: event.StartAt,
		EndAt:   event.EndAt,
		UserId:  event.UserID,
	}
	if event.Description != "" {
		description := event.Description
		result.Description = &description
	}
	if event.NotifyBefore > 0 {
		notifyBefore := event.NotifyBefore.String()
		result.NotifyBefore = &notifyBefore
	}

	return result
}

func writeAppError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrEventNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, storage.ErrEventExists), errors.Is(err, storage.ErrDateBusy):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, openapi.Error{
		Code:    int32(status),
		Message: message,
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
