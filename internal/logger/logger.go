package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a structured logger writing JSON records to standard error.
// The level argument is a case-insensitive name ("debug", "info", "warn",
// "error"); unknown values fall back to info.
func New(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	})
	return slog.New(handler)
}
