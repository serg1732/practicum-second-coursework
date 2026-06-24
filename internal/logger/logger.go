package logger

import (
	"log/slog"
	"os"
)

// NewSlogLogger инициализация логера.
func NewSlogLogger(level slog.Level) *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		}),
	)
}

// ParseLevel - получение уровня логирования из текста.
func ParseLevel(s string) (slog.Level, error) {
	var level slog.Level
	err := level.UnmarshalText([]byte(s))
	return level, err
}
