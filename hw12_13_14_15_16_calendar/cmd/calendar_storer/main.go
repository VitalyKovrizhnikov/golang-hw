package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/broker/kafka"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/logger"
	storagefactory "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage/factory"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storer"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/calendar/storer_config.toml", "Path to configuration file")
}

func main() {
	os.Exit(run())
}

func run() int {
	flag.Parse()

	calendarConfig, err := NewConfig(configFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		return 1
	}

	logg := logger.New(calendarConfig.Logger.Level)

	store, closeStore, err := storagefactory.Build(storagefactory.Config{
		Type: calendarConfig.Storage.Type,
		DSN:  calendarConfig.Storage.DSN,
	})
	if err != nil {
		logg.Error("failed to init storage: " + err.Error())
		return 1
	}
	defer closeStore()

	consumer := kafkabroker.NewConsumer(kafkabroker.Config{
		Brokers: calendarConfig.Kafka.Brokers,
		Topic:   calendarConfig.Kafka.Topic,
		GroupID: calendarConfig.Kafka.GroupID,
	})
	defer func() {
		if err := consumer.Close(); err != nil {
			logg.Error("failed to close kafka consumer: " + err.Error())
		}
	}()

	process := storer.New(logg, store, consumer, storer.Options{})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logg.Info("storer is running")

	if err := process.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logg.Error("storer failed: " + err.Error())
		return 1
	}

	logg.Info("storer stopped")

	return 0
}
