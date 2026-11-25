package main

import (
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/config"
)

const (
	defaultInterval  = 10 * time.Second
	defaultRetention = 365 * 24 * time.Hour
)

type Config struct {
	Logger    config.Logger    `toml:"logger"`
	Storage   config.Storage   `toml:"storage"`
	Kafka     config.Kafka     `toml:"kafka"`
	Scheduler config.Scheduler `toml:"scheduler"`

	Interval  time.Duration
	Retention time.Duration
}

func NewConfig(path string) (Config, error) {
	var calendarConfig Config

	if err := config.Load(path, &calendarConfig); err != nil {
		return Config{}, err
	}

	if err := config.NormalizeLogger(&calendarConfig.Logger); err != nil {
		return Config{}, err
	}
	if err := config.NormalizeStorage(&calendarConfig.Storage); err != nil {
		return Config{}, err
	}
	config.NormalizeKafka(&calendarConfig.Kafka)

	interval, err := config.ParseDuration(
		"scheduler.interval",
		calendarConfig.Scheduler.Interval,
		defaultInterval,
	)
	if err != nil {
		return Config{}, err
	}

	retention, err := config.ParseDuration(
		"scheduler.retention",
		calendarConfig.Scheduler.Retention,
		defaultRetention,
	)
	if err != nil {
		return Config{}, err
	}

	calendarConfig.Interval = interval
	calendarConfig.Retention = retention

	return calendarConfig, nil
}
