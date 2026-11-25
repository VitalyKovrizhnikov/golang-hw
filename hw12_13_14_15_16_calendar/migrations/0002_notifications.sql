ALTER TABLE events ADD COLUMN IF NOT EXISTS notified_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL,
    title TEXT NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    user_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS notifications_event_user_idx
    ON notifications (event_id, user_id);
