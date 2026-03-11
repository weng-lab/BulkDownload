package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartCleanupRemovesExpiredJobsAndFiles(t *testing.T) {
	outputDir := useTestRuntime(t, 3*time.Second, 200*time.Millisecond, 0)

	store := NewStore()
	filename := "expired.zip"
	zipPath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(zipPath, []byte("zip bytes"), 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	store.Set(&Job{
		ID:        "expired-job",
		Status:    StatusDone,
		Filename:  filename,
		ExpiresAt: time.Now().Add(-time.Second),
	})
	store.Set(&Job{
		ID:        "active-job",
		Status:    StatusDone,
		ExpiresAt: time.Now().Add(time.Second),
	})

	StartCleanup(store)

	waitFor(t, 2*time.Second, 50*time.Millisecond, func() bool {
		_, expiredExists := store.Get("expired-job")
		_, activeExists := store.Get("active-job")
		_, err := os.Stat(zipPath)
		return !expiredExists && activeExists && os.IsNotExist(err)
	}, "cleanup to remove expired job and zip file")
}

func TestStartCleanupDeletesExpiredJobWhenFileMissing(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 200*time.Millisecond, 0)

	store := NewStore()
	store.Set(&Job{
		ID:        "expired-missing-file",
		Status:    StatusDone,
		Filename:  "missing.zip",
		ExpiresAt: time.Now().Add(-time.Second),
	})

	StartCleanup(store)

	waitFor(t, 2*time.Second, 50*time.Millisecond, func() bool {
		_, ok := store.Get("expired-missing-file")
		return !ok
	}, "cleanup to delete expired job with missing file")
}
