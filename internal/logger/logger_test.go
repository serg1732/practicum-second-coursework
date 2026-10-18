package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected slog.Level
	}{
		{
			name:     "DEBUG",
			value:    "DEBUG",
			expected: slog.LevelDebug,
		},
		{
			name:     "debug в нижнем регистре",
			value:    "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "INFO",
			value:    "INFO",
			expected: slog.LevelInfo,
		},
		{
			name:     "WARN",
			value:    "WARN",
			expected: slog.LevelWarn,
		},
		{
			name:     "ERROR",
			value:    "ERROR",
			expected: slog.LevelError,
		},
		{
			name:     "уровень со смещением",
			value:    "ERROR+2",
			expected: slog.LevelError + 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseLevel(tt.value)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestParseLevelErrorUnknownLevel(t *testing.T) {
	actual, err := ParseLevel("UNKNOWN")

	assert.Error(t, err)
	assert.Equal(t, slog.LevelInfo, actual)
}

func TestNewSlogLogger(t *testing.T) {
	log := NewSlogLogger(slog.LevelWarn)

	if !assert.NotNil(t, log) {
		return
	}

	ctx := context.Background()
	assert.False(t, log.Enabled(ctx, slog.LevelInfo))
	assert.True(t, log.Enabled(ctx, slog.LevelWarn))
	assert.True(t, log.Enabled(ctx, slog.LevelError))
}

func TestNewSlogLoggerWriteJSONToStdout(t *testing.T) {
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	assert.NoError(t, err)
	defer func() {
		os.Stdout = oldStdout
		assert.NoError(t, reader.Close())
	}()

	os.Stdout = writer
	log := NewSlogLogger(slog.LevelInfo)
	log.Info("test message", "key", "value")

	assert.NoError(t, writer.Close())

	data := make([]byte, 1024)
	n, err := reader.Read(data)
	assert.NoError(t, err)

	line := strings.TrimSpace(string(data[:n]))
	if !assert.NotEmpty(t, line) {
		return
	}

	var actual map[string]any
	assert.NoError(t, json.Unmarshal([]byte(line), &actual))

	assert.Equal(t, "INFO", actual["level"])
	assert.Equal(t, "test message", actual["msg"])
	assert.Equal(t, "value", actual["key"])
}
