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

	outputDir := filepath.Join(t.TempDir(), "zips")
	t.Setenv("OUTPUT_DIR", outputDir)
	t.Setenv("ZIP_TTL", zipTTL.String())
	t.Setenv("CLEANUP_TICK", cleanupTick.String())
	t.Setenv("PROCESSING_DELAY", processingDelay.String())
	LoadConfig()

	if err := os.MkdirAll(OutputDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}

	return outputDir
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
