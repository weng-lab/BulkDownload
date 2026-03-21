package core

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
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
			name: "invalid duration fails",
			env: map[string]string{
				"JOB_TTL": "nope",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			got, err := LoadConfig()
			if tt.wantErr {
				if err == nil {
					t.Fatal("LoadConfig() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("LoadConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"JOBS_DIR",
		"SOURCE_ROOT_DIR",
		"PUBLIC_BASE_URL",
		"DOWNLOAD_ROOT_DIR",
		"PORT",
		"JOB_TTL",
		"CLEANUP_TICK",
	} {
		t.Setenv(key, "")
	}
}
