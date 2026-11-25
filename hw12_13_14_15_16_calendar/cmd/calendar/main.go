package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/app"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/logger"
	"github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/metrics"
	internalhttp "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/server/http"
	storagefactory "github.com/VitalyKovrizhnikov/hw12_13_14_15_16_calendar/internal/storage/factory"
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

	store, closeStore, err := storagefactory.Build(storagefactory.Config{
		Type: config.Storage.Type,
		DSN:  config.Storage.DSN,
	})
	if err != nil {
		logg.Error("failed to init storage: " + err.Error())
		return 1
	}
	defer closeStore()

	promMetrics := metrics.New()
	calendar := app.New(logg, store, promMetrics)
	address := net.JoinHostPort(config.HTTP.Host, config.HTTP.Port)
	server := internalhttp.NewServer(logg, calendar, promMetrics, address)

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
