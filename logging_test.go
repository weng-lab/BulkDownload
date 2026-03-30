package main

import (
	"log/slog"
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
