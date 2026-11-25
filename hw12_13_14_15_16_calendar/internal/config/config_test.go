package config

import (
	"testing"
	"time"
)

func TestNormalizeLogger(t *testing.T) {
	logger := Logger{}
	if err := NormalizeLogger(&logger); err != nil {
		t.Fatal(err)
	}
	if logger.Level != DefaultLogLevel {
		t.Fatalf("level = %q, want %q", logger.Level, DefaultLogLevel)
	}

	logger = Logger{Level: " WARN "}
	if err := NormalizeLogger(&logger); err != nil {
		t.Fatal(err)
	}
	if logger.Level != "warn" {
		t.Fatalf("level = %q, want warn", logger.Level)
	}

	logger = Logger{Level: "verbose"}
	if err := NormalizeLogger(&logger); err == nil {
		t.Fatal("expected error for unsupported level")
	}
}

func TestNormalizeStorage(t *testing.T) {
	storage := Storage{}
	if err := NormalizeStorage(&storage); err != nil {
		t.Fatal(err)
	}
	if storage.Type != StorageSQL || storage.DSN != DefaultStorageDSN {
		t.Fatalf("storage = %+v", storage)
	}

	storage = Storage{Type: StorageMemory, DSN: "ignored"}
	if err := NormalizeStorage(&storage); err != nil {
		t.Fatal(err)
	}
	if storage.Type != StorageMemory {
		t.Fatalf("storage = %+v", storage)
	}

	storage = Storage{Type: "unknown"}
	if err := NormalizeStorage(&storage); err == nil {
		t.Fatal("expected error for unsupported storage")
	}
}

func TestNormalizeKafka(t *testing.T) {
	kafka := Kafka{}
	NormalizeKafka(&kafka)

	if len(kafka.Brokers) != 1 || kafka.Brokers[0] != DefaultBroker {
		t.Fatalf("brokers = %v", kafka.Brokers)
	}
	if kafka.Topic != DefaultTopic || kafka.GroupID != DefaultGroupID {
		t.Fatalf("kafka = %+v", kafka)
	}
}

func TestParseDuration(t *testing.T) {
	duration, err := ParseDuration("interval", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if duration != time.Second {
		t.Fatalf("duration = %s, want 1s", duration)
	}

	duration, err = ParseDuration("interval", "2m", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if duration != 2*time.Minute {
		t.Fatalf("duration = %s, want 2m", duration)
	}

	if _, err := ParseDuration("interval", "soon", time.Second); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := ParseDuration("interval", "0s", time.Second); err == nil {
		t.Fatal("expected positive duration error")
	}
}
