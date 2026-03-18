package core

import (
	"testing"
	"time"
)

func TestLoadConfigUsesDefaultsWithoutEnv(t *testing.T) {
	t.Cleanup(func() {
		LoadConfig()
	})
	t.Setenv("JOBS_DIR", "")
	t.Setenv("SOURCE_ROOT_DIR", "")
	t.Setenv("PUBLIC_BASE_URL", "")
	t.Setenv("DOWNLOAD_ROOT_DIR", "")
	t.Setenv("PORT", "")
	t.Setenv("ZIP_TTL", "")
	t.Setenv("CLEANUP_TICK", "")
	t.Setenv("PROCESSING_DELAY", "")

	LoadConfig()

	if JobsDir != "./jobs" {
		t.Fatalf("expected default JobsDir, got %s", JobsDir)
	}
	if SourceRootDir != "" {
		t.Fatalf("expected default SourceRootDir, got %s", SourceRootDir)
	}
	if PublicBaseURL != "https://download.mohd.org" {
		t.Fatalf("expected default PublicBaseURL, got %s", PublicBaseURL)
	}
	if DownloadRootDir != "mohd_data" {
		t.Fatalf("expected default DownloadRootDir, got %s", DownloadRootDir)
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
	t.Setenv("JOBS_DIR", "/tmp/bulkdownload-jobs")
	t.Setenv("SOURCE_ROOT_DIR", "/mnt/source-data")
	t.Setenv("PUBLIC_BASE_URL", "https://example.com/data")
	t.Setenv("DOWNLOAD_ROOT_DIR", "custom-data")
	t.Setenv("PORT", "9090")
	t.Setenv("ZIP_TTL", "30s")
	t.Setenv("CLEANUP_TICK", "5s")
	t.Setenv("PROCESSING_DELAY", "2s")

	LoadConfig()

	if JobsDir != "/tmp/bulkdownload-jobs" {
		t.Fatalf("expected JobsDir override, got %s", JobsDir)
	}
	if SourceRootDir != "/mnt/source-data" {
		t.Fatalf("expected SourceRootDir override, got %s", SourceRootDir)
	}
	if PublicBaseURL != "https://example.com/data" {
		t.Fatalf("expected PublicBaseURL override, got %s", PublicBaseURL)
	}
	if DownloadRootDir != "custom-data" {
		t.Fatalf("expected DownloadRootDir override, got %s", DownloadRootDir)
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
	t.Setenv("JOBS_DIR", "")
	t.Setenv("SOURCE_ROOT_DIR", "")
	t.Setenv("PUBLIC_BASE_URL", "")
	t.Setenv("DOWNLOAD_ROOT_DIR", "")
	t.Setenv("PORT", "")
	t.Setenv("ZIP_TTL", "nope")
	t.Setenv("CLEANUP_TICK", "still-nope")
	t.Setenv("PROCESSING_DELAY", "bad")

	LoadConfig()

	if JobsDir != "./jobs" {
		t.Fatalf("expected default JobsDir after invalid env, got %s", JobsDir)
	}
	if PublicBaseURL != "https://download.mohd.org" {
		t.Fatalf("expected default PublicBaseURL after invalid env, got %s", PublicBaseURL)
	}
	if DownloadRootDir != "mohd_data" {
		t.Fatalf("expected default DownloadRootDir after invalid env, got %s", DownloadRootDir)
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
