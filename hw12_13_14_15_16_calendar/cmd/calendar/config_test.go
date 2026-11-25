package main

import (
	"os"
	"path/filepath"
	"testing"
)

const testDSN = "postgres://calendar:calendar@localhost:5432/calendar?sslmode=disable"

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestConfigDefaults(t *testing.T) {
	config, err := NewConfig(writeConfig(t, ""))
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}

	if config.Logger.Level != "info" {
		t.Errorf("logger level = %q, want info", config.Logger.Level)
	}
	if config.Storage.Type != "memory" {
		t.Errorf("storage type = %q, want memory", config.Storage.Type)
	}
	if config.HTTP.Host != "127.0.0.1" || config.HTTP.Port != "8080" {
		t.Errorf("http address = %s:%s, want 127.0.0.1:8080", config.HTTP.Host, config.HTTP.Port)
	}
}

func TestConfigCustom(t *testing.T) {
	content := `
[logger]
level = "DEBUG"

[storage]
type = "SQL"
dsn = "` + testDSN + `"

[http]
host = "0.0.0.0"
port = "9090"
`

	config, err := NewConfig(writeConfig(t, content))
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}

	if config.Logger.Level != "debug" {
		t.Errorf("logger level = %q, want debug", config.Logger.Level)
	}
	if config.Storage.Type != "sql" || config.Storage.DSN != testDSN {
		t.Errorf("storage = %+v", config.Storage)
	}
	if config.HTTP.Host != "0.0.0.0" || config.HTTP.Port != "9090" {
		t.Errorf("http address = %s:%s", config.HTTP.Host, config.HTTP.Port)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "unsupported storage",
			content: "[storage]\ntype = \"file\"\n",
		},
		{
			name:    "sql without dsn",
			content: "[storage]\ntype = \"sql\"\n",
		},
		{
			name:    "unsupported log level",
			content: "[logger]\nlevel = \"verbose\"\n",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewConfig(writeConfig(t, test.content)); err == nil {
				t.Fatal("NewConfig() error = nil, want error")
			}
		})
	}
}

func TestConfigMissingFile(t *testing.T) {
	if _, err := NewConfig(filepath.Join(t.TempDir(), "missing.toml")); err == nil {
		t.Fatal("NewConfig() error = nil, want error")
	}
}
