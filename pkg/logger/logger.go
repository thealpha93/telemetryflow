// Package logger provides a pre-configured structured JSON logger
// built on Go's stdlib log/slog (ADR-010).
//
// All services use the same logger setup:
//   - JSON output to stdout (structured, machine-parseable)
//   - INFO level by default (override with LOG_LEVEL env var)
//   - Always log with context: slog.InfoContext(ctx, "msg", "key", val)
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a *slog.Logger that writes JSON to stdout.
// Log level is read from the LOG_LEVEL environment variable.
// Valid values: DEBUG, INFO, WARN, ERROR (case-insensitive). Defaults to INFO.
//
// Call slog.SetDefault(logger.New()) in main() so package-level slog
// functions (slog.InfoContext, slog.ErrorContext, etc.) use this logger.
func New() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,

		// AddSource adds "source":{"function":"...","file":"...","line":N}
		// to every log entry — useful in production for tracing errors back
		// to the exact line without a stack trace.
		AddSource: level == slog.LevelDebug,
	})

	return slog.New(handler)
}

// parseLevel converts a string to a slog.Level.
// Unrecognised values fall back to INFO so a misconfigured LOG_LEVEL
// never silently disables logging.
func parseLevel(s string) slog.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
