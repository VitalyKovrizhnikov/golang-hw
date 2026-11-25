package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	logger *log.Logger
	level  Level
}

func New(level string) *Logger {
	return NewWithWriter(os.Stdout, level)
}

func NewWithWriter(writer io.Writer, level string) *Logger {
	parsedLevel, err := ParseLevel(level)
	if err != nil {
		parsedLevel = LevelInfo
	}

	return &Logger{
		logger: log.New(writer, "", log.LstdFlags),
		level:  parsedLevel,
	}
}

func ParseLevel(level string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level %q", level)
	}
}

func (l *Logger) Debug(msg string) {
	l.log(LevelDebug, "DEBUG", msg)
}

func (l *Logger) Info(msg string) {
	l.log(LevelInfo, "INFO", msg)
}

func (l *Logger) Warn(msg string) {
	l.log(LevelWarn, "WARN", msg)
}

func (l *Logger) Error(msg string) {
	l.log(LevelError, "ERROR", msg)
}

func (l *Logger) log(level Level, name, msg string) {
	if level < l.level {
		return
	}

	l.logger.Printf("[%s] %s", name, msg)
}
