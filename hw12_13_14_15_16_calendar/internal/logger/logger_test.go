package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input   string
		want    Level
		wantErr bool
	}{
		{input: "debug", want: LevelDebug},
		{input: "INFO", want: LevelInfo},
		{input: "warn", want: LevelWarn},
		{input: "warning", want: LevelWarn},
		{input: "error", want: LevelError},
		{input: "", want: LevelInfo, wantErr: true},
		{input: "verbose", want: LevelInfo, wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			level, err := ParseLevel(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseLevel(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if level != test.want {
				t.Fatalf("ParseLevel(%q) = %v, want %v", test.input, level, test.want)
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	var output bytes.Buffer
	logg := NewWithWriter(&output, "warn")

	logg.Debug("debug message")
	logg.Info("info message")
	logg.Warn("warn message")
	logg.Error("error message")

	logs := output.String()
	if strings.Contains(logs, "debug message") || strings.Contains(logs, "info message") {
		t.Fatalf("unexpected low-level logs: %q", logs)
	}
	if !strings.Contains(logs, "warn message") || !strings.Contains(logs, "error message") {
		t.Fatalf("missing warn/error logs: %q", logs)
	}
	if !strings.Contains(logs, "[WARN]") || !strings.Contains(logs, "[ERROR]") {
		t.Fatalf("missing level prefixes: %q", logs)
	}
}

func TestLoggerDebugLevel(t *testing.T) {
	var output bytes.Buffer
	logg := NewWithWriter(&output, "debug")

	logg.Debug("debug message")

	if !strings.Contains(output.String(), "debug message") {
		t.Fatalf("missing debug log: %q", output.String())
	}
}
