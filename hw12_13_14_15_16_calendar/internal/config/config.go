package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	StorageMemory = "memory"
	StorageSQL    = "sql"

	DefaultLogLevel   = "info"
	DefaultStorageDSN = "postgres://calendar:calendar@localhost:5432/calendar?sslmode=disable"
	DefaultBroker     = "localhost:9092"
	DefaultTopic      = "calendar.notifications"
	DefaultGroupID    = "calendar-storer"
)

type Logger struct {
	Level string `toml:"level"`
}

type Storage struct {
	Type string `toml:"type"`
	DSN  string `toml:"dsn"`
}

type Kafka struct {
	Brokers []string `toml:"brokers"`
	Topic   string   `toml:"topic"`
	GroupID string   `toml:"group_id"`
}

type Scheduler struct {
	Interval  string `toml:"interval"`
	Retention string `toml:"retention"`
}

func Load(path string, target interface{}) error {
	if _, err := toml.DecodeFile(path, target); err != nil {
		return fmt.Errorf("decode config %s: %w", path, err)
	}

	return nil
}

func NormalizeLogger(logger *Logger) error {
	logger.Level = strings.ToLower(strings.TrimSpace(logger.Level))
	if logger.Level == "" {
		logger.Level = DefaultLogLevel
	}

	switch logger.Level {
	case "debug", DefaultLogLevel, "warn", "warning", "error":
		return nil
	default:
		return fmt.Errorf("unsupported log level %q", logger.Level)
	}
}

func NormalizeStorage(storage *Storage) error {
	storage.Type = strings.ToLower(strings.TrimSpace(storage.Type))
	if storage.Type == "" {
		storage.Type = StorageSQL
	}

	switch storage.Type {
	case StorageMemory:
		return nil
	case StorageSQL:
		if strings.TrimSpace(storage.DSN) == "" {
			storage.DSN = DefaultStorageDSN
		}

		return nil
	default:
		return fmt.Errorf("unsupported storage type %q", storage.Type)
	}
}

func NormalizeKafka(kafka *Kafka) {
	brokers := make([]string, 0, len(kafka.Brokers))
	for _, broker := range kafka.Brokers {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}
	if len(brokers) == 0 {
		brokers = []string{DefaultBroker}
	}
	kafka.Brokers = brokers

	kafka.Topic = strings.TrimSpace(kafka.Topic)
	if kafka.Topic == "" {
		kafka.Topic = DefaultTopic
	}

	kafka.GroupID = strings.TrimSpace(kafka.GroupID)
	if kafka.GroupID == "" {
		kafka.GroupID = DefaultGroupID
	}
}

func ParseDuration(name, value string, defaultValue time.Duration) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return defaultValue, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}

	return duration, nil
}
