package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestResolveConfig(t *testing.T) {
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
			name: "blank values fall back to defaults",
			env: map[string]string{
				"JOBS_DIR":          "",
				"SOURCE_ROOT_DIR":   "   ",
				"PUBLIC_BASE_URL":   "",
				"DOWNLOAD_ROOT_DIR": " ",
				"PORT":              "",
				"JOB_TTL":           "  ",
				"CLEANUP_TICK":      "",
			},
			want: defaultConfig(),
		},
		{
			name: "string values are trimmed",
			env: map[string]string{
				"PUBLIC_BASE_URL": " https://example.com/data ",
				"PORT":            " 9090 ",
			},
			want: Config{
				JobsDir:         "./jobs",
				SourceRootDir:   "",
				PublicBaseURL:   "https://example.com/data",
				DownloadRootDir: "mohd_data",
				Port:            "9090",
				JobTTL:          24 * time.Hour,
				CleanupTick:     5 * time.Minute,
			},
		},
		{
			name: "duration values are trimmed and parsed",
			env: map[string]string{
				"JOB_TTL":      " 45s ",
				"CLEANUP_TICK": " 10s ",
			},
			want: Config{
				JobsDir:         "./jobs",
				SourceRootDir:   "",
				PublicBaseURL:   "https://download.mohd.org",
				DownloadRootDir: "mohd_data",
				Port:            "8080",
				JobTTL:          45 * time.Second,
				CleanupTick:     10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveConfig(tt.env)
			if err != nil {
				t.Fatalf("resolveConfig() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("resolveConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveConfig_InvalidDuration(t *testing.T) {
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
			name: "zero job ttl fails",
			env: map[string]string{
				"JOB_TTL": "0s",
			},
			wantIs:     ErrInvalidJobTTL,
			wantErrMsg: "invalid JOB_TTL: must be greater than 0",
		},
		{
			name: "negative cleanup tick fails",
			env: map[string]string{
				"CLEANUP_TICK": "-5s",
			},
			wantIs:     ErrInvalidCleanupTick,
			wantErrMsg: "invalid CLEANUP_TICK: must be greater than 0",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveConfig(tt.env)
			if err == nil {
				t.Fatal("resolveConfig() error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantIs) {
				t.Fatalf("resolveConfig() error = %v, want %v", err, tt.wantIs)
			}
			if diff := cmp.Diff(tt.wantErrMsg, err.Error()); diff != "" {
				t.Errorf("resolveConfig() error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	withWorkingDir(t, dir)

	const envFile = `# local overrides
JOBS_DIR=./custom-jobs
export SOURCE_ROOT_DIR=./source
PUBLIC_BASE_URL="http://localhost:9000"
DOWNLOAD_ROOT_DIR='downloads'
PORT=9090
JOB_TTL=5m
CLEANUP_TICK=30s
`
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envFile), 0o644); err != nil {
		t.Fatalf("WriteFile(.env) error = %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := Config{
		JobsDir:         "./custom-jobs",
		SourceRootDir:   "./source",
		PublicBaseURL:   "http://localhost:9000",
		DownloadRootDir: "downloads",
		Port:            "9090",
		JobTTL:          5 * time.Minute,
		CleanupTick:     30 * time.Second,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LoadConfig() mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadConfig_MissingDotEnvUsesDefaults(t *testing.T) {
	clearConfigEnv(t)
	withWorkingDir(t, t.TempDir())

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if diff := cmp.Diff(defaultConfig(), got); diff != "" {
		t.Errorf("LoadConfig() mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadConfig_EnvOverridesDotEnv(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	withWorkingDir(t, dir)

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("PORT=9090\nJOB_TTL=5m\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.env) error = %v", err)
	}
	t.Setenv("PORT", "8081")
	t.Setenv("JOB_TTL", "45s")

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if diff := cmp.Diff("8081", got.Port); diff != "" {
		t.Errorf("LoadConfig() port mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(45*time.Second, got.JobTTL); diff != "" {
		t.Errorf("LoadConfig() job ttl mismatch (-want +got):\n%s", diff)
	}
}
