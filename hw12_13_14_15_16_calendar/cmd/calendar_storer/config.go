package main

import (
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/config"
)

type Config struct {
	Logger  config.Logger  `toml:"logger"`
	Storage config.Storage `toml:"storage"`
	Kafka   config.Kafka   `toml:"kafka"`
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

	return calendarConfig, nil
}
