package logger

import (
	"log/slog"
	"os"
)

// Init sets up the global slog logger with JSON output.
// Call this once in main() before any other package logs.
func Init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
