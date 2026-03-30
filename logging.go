package main

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

func newLogger(w io.Writer, rawLevel string) (*slog.Logger, error) {
	level, err := parseLogLevel(rawLevel)
	if err != nil {
		return nil, err
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler), nil
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level: %q", raw)
	}
}
