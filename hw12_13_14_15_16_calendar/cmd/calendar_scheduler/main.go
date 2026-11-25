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
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/scheduler"
	storagefactory "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage/factory"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/calendar/scheduler_config.toml", "Path to configuration file")
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

	producer := kafkabroker.NewProducer(kafkabroker.Config{
		Brokers: calendarConfig.Kafka.Brokers,
		Topic:   calendarConfig.Kafka.Topic,
	})
	defer func() {
		if err := producer.Close(); err != nil {
			logg.Error("failed to close kafka producer: " + err.Error())
		}
	}()

	process := scheduler.New(logg, store, producer, scheduler.Options{
		Interval:  calendarConfig.Interval,
		Retention: calendarConfig.Retention,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logg.Info("scheduler is running")

	if err := process.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logg.Error("scheduler failed: " + err.Error())
		return 1
	}

	logg.Info("scheduler stopped")

	return 0
}
