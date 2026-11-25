package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/app"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/logger"
	internalhttp "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/server/http"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage"
	memorystorage "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage/memory"
	sqlstorage "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage/sql"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/calendar/config.toml", "Path to configuration file")
}

func main() {
	os.Exit(run())
}

func run() int {
	flag.Parse()

	if flag.Arg(0) == "version" {
		printVersion()
		return 0
	}

	config, err := NewConfig(configFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		return 1
	}

	logg := logger.New(config.Logger.Level)

	store, closeStore, err := buildStorage(config)
	if err != nil {
		logg.Error("failed to init storage: " + err.Error())
		return 1
	}
	defer closeStore()

	calendar := app.New(logg, store)
	address := net.JoinHostPort(config.HTTP.Host, config.HTTP.Port)
	server := internalhttp.NewServer(logg, calendar, address)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logg.Info("calendar is running on " + address)

	if err := server.Start(ctx); err != nil {
		logg.Error("failed to run http server: " + err.Error())
		return 1
	}

	logg.Info("calendar stopped")

	return 0
}

func buildStorage(config Config) (storage.Storage, func(), error) {
	switch config.Storage.Type {
	case storageTypeMemory:
		return memorystorage.New(), func() {}, nil
	case storageTypeSQL:
		store, err := sqlstorage.New(config.Storage.DSN)
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
		return nil, nil, fmt.Errorf("unsupported storage type %q", config.Storage.Type)
	}
}
