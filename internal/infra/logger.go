package infra

import (
	"io"
	"log/slog"
)

func NewLogger(writer io.Writer, verbose bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
	}

	if verbose {
		opts.Level = slog.LevelDebug
		opts.AddSource = true
	}

	handler := slog.NewTextHandler(writer, opts)

	logger := slog.New(handler)

	return logger
}
