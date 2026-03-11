package core

import (
	"os"
	"testing"
	"time"
)

func useTestRuntime(t *testing.T, zipTTL, cleanupTick, processingDelay time.Duration) {
	t.Helper()

	prevZipTTL := ZipTTL
	prevCleanupTick := CleanupTick
	prevProcessingDelay := ProcessingDelay
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	tempDir := t.TempDir()

	ZipTTL = zipTTL
	CleanupTick = cleanupTick
	ProcessingDelay = processingDelay

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	if err := os.MkdirAll(OutputDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}

	t.Cleanup(func() {
		ZipTTL = prevZipTTL
		CleanupTick = prevCleanupTick
		ProcessingDelay = prevProcessingDelay
		_ = os.Chdir(prevWD)
	})
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
