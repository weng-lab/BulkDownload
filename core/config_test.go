package core

import (
	"testing"
	"time"
)

func TestLoadConfigUsesDefaultsWithoutEnv(t *testing.T) {
	t.Cleanup(func() {
		LoadConfig()
	})
	t.Setenv("OUTPUT_DIR", "")
	t.Setenv("PORT", "")
	t.Setenv("ZIP_TTL", "")
	t.Setenv("CLEANUP_TICK", "")
	t.Setenv("PROCESSING_DELAY", "")

	LoadConfig()

	if OutputDir != "./zips" {
		t.Fatalf("expected default OutputDir, got %s", OutputDir)
	}
	if Port != "8080" {
		t.Fatalf("expected default Port, got %s", Port)
	}
	if ZipTTL != 24*time.Hour {
		t.Fatalf("expected default ZipTTL, got %s", ZipTTL)
	}
	if CleanupTick != 5*time.Minute {
		t.Fatalf("expected default CleanupTick, got %s", CleanupTick)
	}
	if ProcessingDelay != 0 {
		t.Fatalf("expected default ProcessingDelay, got %s", ProcessingDelay)
	}
}

func TestLoadConfigUsesEnvOverrides(t *testing.T) {
	t.Cleanup(func() {
		LoadConfig()
	})
	t.Setenv("OUTPUT_DIR", "/tmp/bulkdownload-test")
	t.Setenv("PORT", "9090")
	t.Setenv("ZIP_TTL", "30s")
	t.Setenv("CLEANUP_TICK", "5s")
	t.Setenv("PROCESSING_DELAY", "2s")

	LoadConfig()

	if OutputDir != "/tmp/bulkdownload-test" {
		t.Fatalf("expected OutputDir override, got %s", OutputDir)
	}
	if Port != "9090" {
		t.Fatalf("expected Port override, got %s", Port)
	}
	if ZipTTL != 30*time.Second {
		t.Fatalf("expected ZipTTL override, got %s", ZipTTL)
	}
	if CleanupTick != 5*time.Second {
		t.Fatalf("expected CleanupTick override, got %s", CleanupTick)
	}
	if ProcessingDelay != 2*time.Second {
		t.Fatalf("expected ProcessingDelay override, got %s", ProcessingDelay)
	}
}

func TestLoadConfigFallsBackOnInvalidEnv(t *testing.T) {
	t.Cleanup(func() {
		LoadConfig()
	})
	t.Setenv("OUTPUT_DIR", "")
	t.Setenv("PORT", "")
	t.Setenv("ZIP_TTL", "nope")
	t.Setenv("CLEANUP_TICK", "still-nope")
	t.Setenv("PROCESSING_DELAY", "bad")

	LoadConfig()

	if OutputDir != "./zips" {
		t.Fatalf("expected default OutputDir after invalid env, got %s", OutputDir)
	}
	if Port != "8080" {
		t.Fatalf("expected default Port after invalid env, got %s", Port)
	}
	if ZipTTL != 24*time.Hour {
		t.Fatalf("expected default ZipTTL after invalid env, got %s", ZipTTL)
	}
	if CleanupTick != 5*time.Minute {
		t.Fatalf("expected default CleanupTick after invalid env, got %s", CleanupTick)
	}
	if ProcessingDelay != 0 {
		t.Fatalf("expected default ProcessingDelay after invalid env, got %s", ProcessingDelay)
	}
}
