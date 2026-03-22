package core

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig_FromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want Config
	}{
		{
			name: "defaults",
			want: defaultConfig(),
		},
		{
			name: "env overrides",
			env: map[string]string{
				"JOBS_DIR":          "/tmp/bulkdownload-jobs",
				"SOURCE_ROOT_DIR":   "/mnt/source-data",
				"PUBLIC_BASE_URL":   "https://example.com/data",
				"DOWNLOAD_ROOT_DIR": "custom-data",
				"PORT":              "9090",
				"JOB_TTL":           "30s",
				"CLEANUP_TICK":      "5s",
			},
			want: Config{
				JobsDir:         "/tmp/bulkdownload-jobs",
				SourceRootDir:   "/mnt/source-data",
				PublicBaseURL:   "https://example.com/data",
				DownloadRootDir: "custom-data",
				Port:            "9090",
				JobTTL:          30 * time.Second,
				CleanupTick:     5 * time.Second,
			},
		},
		{
			name: "empty env values fall back to defaults",
			env: map[string]string{
				"JOBS_DIR":          "",
				"SOURCE_ROOT_DIR":   "",
				"PUBLIC_BASE_URL":   "",
				"DOWNLOAD_ROOT_DIR": "",
				"PORT":              "",
				"JOB_TTL":           "",
				"CLEANUP_TICK":      "",
			},
			want: defaultConfig(),
		},
		{
			name: "whitespace string values are preserved",
			env: map[string]string{
				"PUBLIC_BASE_URL": " https://example.com/data ",
				"PORT":            " 9090 ",
			},
			want: Config{
				JobsDir:         "./jobs",
				SourceRootDir:   "",
				PublicBaseURL:   " https://example.com/data ",
				DownloadRootDir: "mohd_data",
				Port:            " 9090 ",
				JobTTL:          24 * time.Hour,
				CleanupTick:     5 * time.Minute,
			},
		},
		{
			name: "zero and negative durations are accepted",
			env: map[string]string{
				"JOB_TTL":      "0s",
				"CLEANUP_TICK": "-5s",
			},
			want: Config{
				JobsDir:         "./jobs",
				SourceRootDir:   "",
				PublicBaseURL:   "https://download.mohd.org",
				DownloadRootDir: "mohd_data",
				Port:            "8080",
				JobTTL:          0,
				CleanupTick:     -5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			got, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("LoadConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadConfig_InvalidDuration(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantIs     error
		wantErrMsg string
	}{
		{
			name: "invalid job ttl fails",
			env: map[string]string{
				"JOB_TTL": "nope",
			},
			wantIs:     ErrInvalidJobTTL,
			wantErrMsg: "invalid JOB_TTL: time: invalid duration \"nope\"",
		},
		{
			name: "cleanup tick with whitespace fails",
			env: map[string]string{
				"CLEANUP_TICK": " 5s ",
			},
			wantIs:     ErrInvalidCleanupTick,
			wantErrMsg: "invalid CLEANUP_TICK: time: invalid duration \" 5s \"",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			_, err := LoadConfig()
			if err == nil {
				t.Fatal("LoadConfig() error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantIs) {
				t.Fatalf("LoadConfig() error = %v, want %v", err, tt.wantIs)
			}
			if diff := cmp.Diff(tt.wantErrMsg, err.Error()); diff != "" {
				t.Errorf("LoadConfig() error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
