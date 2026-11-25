package storagefactory

import (
	"context"
	"fmt"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
	memorystorage "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage/memory"
	sqlstorage "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage/sql"
)

const (
	TypeMemory = "memory"
	TypeSQL    = "sql"
)

type Config struct {
	Type string
	DSN  string
}

func Build(config Config) (storage.Storage, func(), error) {
	switch config.Type {
	case TypeMemory:
		return memorystorage.New(), func() {}, nil
	case TypeSQL:
		store, err := sqlstorage.New(config.DSN)
		if err != nil {
			return nil, nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := store.Connect(ctx); err != nil {
			return nil, nil, err
		}

		return store, func() {
			_ = store.Close(context.Background())
		}, nil
	default:
		return nil, nil, fmt.Errorf("unsupported storage type %q", config.Type)
	}
}
