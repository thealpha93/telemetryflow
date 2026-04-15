// Package logger provides a pre-configured structured JSON logger
// built on Go's stdlib log/slog (ADR-010).
//
// All services use the same logger setup:
//   - JSON output to stdout (structured, machine-parseable)
//   - INFO level by default (override with LOG_LEVEL env var)
//   - Always log with context: slog.InfoContext(ctx, "msg", "key", val)
//
// Why slog over zap/zerolog?
// slog is stdlib in Go 1.21+. Zero extra dependencies, structured by default,
// and fast enough for our throughput. Adding a third-party logging library
// would add complexity with no meaningful gain at this scale.
package logger

import "log/slog"

// New creates and returns a *slog.Logger that writes JSON to stdout.
//
// Usage:
//
//	log := logger.New()
//	slog.SetDefault(log) // makes slog.InfoContext etc. use this logger
func New() *slog.Logger {
	panic("not implemented — implement in Phase 1")
}
