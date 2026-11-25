package main

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	storageTypeMemory = "memory"
	storageTypeSQL    = "sql"
	defaultLogLevel   = "info"
)

type Config struct {
	Logger  LoggerConf  `toml:"logger"`
	Storage StorageConf `toml:"storage"`
	HTTP    HTTPConf    `toml:"http"`
}

type LoggerConf struct {
	Level string `toml:"level"`
}

type StorageConf struct {
	Type string `toml:"type"`
	DSN  string `toml:"dsn"`
}

type HTTPConf struct {
	Host string `toml:"host"`
	Port string `toml:"port"`
}

func NewConfig(path string) (Config, error) {
	var config Config

	if _, err := toml.DecodeFile(path, &config); err != nil {
		return Config{}, fmt.Errorf("decode config %s: %w", path, err)
	}

	config.applyDefaults()

	if err := config.validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (c *Config) applyDefaults() {
	c.Logger.Level = strings.ToLower(strings.TrimSpace(c.Logger.Level))
	c.Storage.Type = strings.ToLower(strings.TrimSpace(c.Storage.Type))

	if c.Logger.Level == "" {
		c.Logger.Level = defaultLogLevel
	}
	if c.Storage.Type == "" {
		c.Storage.Type = storageTypeMemory
	}
	if c.HTTP.Host == "" {
		c.HTTP.Host = "127.0.0.1"
	}
	if c.HTTP.Port == "" {
		c.HTTP.Port = "8080"
	}
}

func (c Config) validate() error {
	switch c.Logger.Level {
	case "debug", defaultLogLevel, "warn", "warning", "error":
	default:
		return fmt.Errorf("unsupported log level %q", c.Logger.Level)
	}

	switch c.Storage.Type {
	case storageTypeMemory:
	case storageTypeSQL:
		if c.Storage.DSN == "" {
			return fmt.Errorf("storage dsn is required for sql storage")
		}
	default:
		return fmt.Errorf("unsupported storage type %q", c.Storage.Type)
	}

	return nil
}
