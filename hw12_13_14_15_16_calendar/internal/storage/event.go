package storage

import "time"

type Event struct {
	ID           string
	Title        string
	StartAt      time.Time
	EndAt        time.Time
	Description  string
	UserID       string
	NotifyBefore time.Duration
	NotifiedAt   time.Time
}
