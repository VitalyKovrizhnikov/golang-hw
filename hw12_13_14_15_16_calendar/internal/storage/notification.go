package storage

import "time"

type Notification struct {
	EventID string    `json:"eventId"`
	Title   string    `json:"title"`
	StartAt time.Time `json:"startAt"`
	UserID  string    `json:"userId"`
}
