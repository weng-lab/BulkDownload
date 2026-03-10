package core

import (
	"testing"
	"time"
)

func TestLoadConfigUsesDefaultsWithoutEnv(t *testing.T) {
	t.Setenv("ZIP_TTL", "")
	t.Setenv("CLEANUP_TICK", "")
	t.Setenv("PROCESSING_DELAY", "")

	LoadConfig()

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
	t.Setenv("ZIP_TTL", "30s")
	t.Setenv("CLEANUP_TICK", "5s")
	t.Setenv("PROCESSING_DELAY", "2s")

	LoadConfig()

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
	t.Setenv("ZIP_TTL", "nope")
	t.Setenv("CLEANUP_TICK", "still-nope")
	t.Setenv("PROCESSING_DELAY", "bad")

	LoadConfig()

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
