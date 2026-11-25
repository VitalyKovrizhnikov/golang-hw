package sqlstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
	"github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

var _ storage.Storage = (*Storage)(nil)

func New(dsn string) (*Storage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Connect(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	return nil
}

func (s *Storage) Close(_ context.Context) error {
	return s.db.Close()
}

func (s *Storage) CreateEvent(ctx context.Context, event storage.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	exists, err := s.eventExists(ctx, event.ID)
	if err != nil {
		return err
	}
	if exists {
		return storage.ErrEventExists
	}

	busy, err := s.isDateBusy(ctx, event, "")
	if err != nil {
		return err
	}
	if busy {
		return storage.ErrDateBusy
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO events (id, title, start_at, end_at, description, user_id, notify_before)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		event.ID,
		event.Title,
		event.StartAt,
		event.EndAt,
		event.Description,
		event.UserID,
		int64(event.NotifyBefore),
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return storage.ErrEventExists
		}

		return fmt.Errorf("create event: %w", err)
	}

	return nil
}

func (s *Storage) UpdateEvent(ctx context.Context, id string, event storage.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	exists, err := s.eventExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return storage.ErrEventNotFound
	}

	busy, err := s.isDateBusy(ctx, event, id)
	if err != nil {
		return err
	}
	if busy {
		return storage.ErrDateBusy
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE events
		SET title = $1,
			start_at = $2,
			end_at = $3,
			description = $4,
			user_id = $5,
			notify_before = $6,
			updated_at = NOW()
		WHERE id = $7`,
		event.Title,
		event.StartAt,
		event.EndAt,
		event.Description,
		event.UserID,
		int64(event.NotifyBefore),
		id,
	)
	if err != nil {
		return fmt.Errorf("update event: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update event rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrEventNotFound
	}

	return nil
}

func (s *Storage) DeleteEvent(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete event rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrEventNotFound
	}

	return nil
}

func (s *Storage) ListEvents(ctx context.Context) ([]storage.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, start_at, end_at, description, user_id, notify_before
		FROM events
		ORDER BY start_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var events []storage.Event
	for rows.Next() {
		var event storage.Event
		var notifyBefore int64

		if err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.StartAt,
			&event.EndAt,
			&event.Description,
			&event.UserID,
			&notifyBefore,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}

		event.NotifyBefore = time.Duration(notifyBefore)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}

func (s *Storage) ListEventsByPeriod(ctx context.Context, userID string, from, to time.Time) ([]storage.Event, error) {
	var fromParam interface{}
	if !from.IsZero() {
		fromParam = from
	}

	var toParam interface{}
	if !to.IsZero() {
		toParam = to
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, start_at, end_at, description, user_id, notify_before
		FROM events
		WHERE user_id = $1
			AND ($2::timestamptz IS NULL OR end_at > $2)
			AND ($3::timestamptz IS NULL OR start_at < $3)
		ORDER BY start_at, id`,
		userID,
		fromParam,
		toParam,
	)
	if err != nil {
		return nil, fmt.Errorf("list events by period: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var events []storage.Event
	for rows.Next() {
		var event storage.Event
		var notifyBefore int64

		if err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.StartAt,
			&event.EndAt,
			&event.Description,
			&event.UserID,
			&notifyBefore,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}

		event.NotifyBefore = time.Duration(notifyBefore)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}

func (s *Storage) eventExists(ctx context.Context, id string) (bool, error) {
	var exists bool

	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check event exists: %w", err)
	}

	return exists, nil
}

func (s *Storage) isDateBusy(ctx context.Context, event storage.Event, excludeID string) (bool, error) {
	var busy bool

	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM events
			WHERE user_id = $1
				AND ($2 = '' OR id <> $2)
				AND start_at < $3
				AND end_at > $4
		)`,
		event.UserID,
		excludeID,
		event.EndAt,
		event.StartAt,
	).Scan(&busy)
	if err != nil {
		return false, fmt.Errorf("check busy date: %w", err)
	}

	return busy, nil
}
