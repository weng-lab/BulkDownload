package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func useTestRuntime(t *testing.T, zipTTL, cleanupTick, processingDelay time.Duration) string {
	t.Helper()

	t.Cleanup(func() {
		LoadConfig()
	})

	jobsDir := filepath.Join(t.TempDir(), "jobs")
	t.Setenv("JOBS_DIR", jobsDir)
	t.Setenv("PUBLIC_BASE_URL", "https://download.mohd.org")
	t.Setenv("DOWNLOAD_ROOT_DIR", "mohd_data")
	t.Setenv("ZIP_TTL", zipTTL.String())
	t.Setenv("CLEANUP_TICK", cleanupTick.String())
	t.Setenv("PROCESSING_DELAY", processingDelay.String())
	LoadConfig()

	if err := os.MkdirAll(JobsDir, 0o755); err != nil {
		t.Fatalf("create jobs dir: %v", err)
	}

	return jobsDir
}

func waitFor(t *testing.T, timeout, interval time.Duration, check func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(interval)
	}

	t.Fatalf("timed out waiting for %s", message)
}
