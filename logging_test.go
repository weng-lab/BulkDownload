package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    slog.Level
		wantErr string
	}{
		{name: "default info", raw: "", want: slog.LevelInfo},
		{name: "trimmed debug", raw: " debug ", want: slog.LevelDebug},
		{name: "warn", raw: "warn", want: slog.LevelWarn},
		{name: "error", raw: "error", want: slog.LevelError},
		{name: "invalid", raw: "trace", wantErr: `invalid log level: "trace"`},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLogLevel(tc.raw)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("parseLogLevel() error = nil, want non-nil")
				}
				if got != 0 {
					t.Fatalf("parseLogLevel() level = %v, want 0 on error", got)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("parseLogLevel() error = %q, want %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseLogLevel() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("parseLogLevel() level = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewLogger_UsesTextHandlerLevel(t *testing.T) {
	var out bytes.Buffer

	logger, err := newLogger(&out, "info")
	if err != nil {
		t.Fatalf("newLogger() error = %v", err)
	}

	logger.Debug("debug hidden")
	logger.Info("service started", slog.String("port", "8080"))

	got := out.String()
	if strings.Contains(got, "debug hidden") {
		t.Fatal("expected debug log to be filtered at info level")
	}
	if !strings.Contains(got, "level=INFO") {
		t.Fatalf("expected info log level in output, got %q", got)
	}
	if !strings.Contains(got, "msg=\"service started\"") {
		t.Fatalf("expected info message in output, got %q", got)
	}
	if !strings.Contains(got, "port=8080") {
		t.Fatalf("expected structured field in output, got %q", got)
	}
}
