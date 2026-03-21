package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSweepExpiredRemovesExpiredJobsAndFiles(t *testing.T) {
	_, jobs, config := testManager(t)
	now := time.Unix(100, 0)

	filename := "expired.zip"
	archivePath := filepath.Join(config.JobsDir, filename)
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(archivePath, []byte("zip bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := jobs.Add(&Job{ID: "expired", Type: JobTypeZip, Status: StatusDone, Filename: filename, ExpiresAt: now.Add(-time.Second)}); err != nil {
		t.Fatalf("Add(expired) error = %v", err)
	}
	if err := jobs.Add(&Job{ID: "active", Type: JobTypeZip, Status: StatusDone, ExpiresAt: now.Add(time.Second)}); err != nil {
		t.Fatalf("Add(active) error = %v", err)
	}

	SweepExpired(jobs, config.JobsDir, now)

	if _, ok := jobs.Get("expired"); ok {
		t.Fatal("Get(expired) ok = true, want false")
	}
	if _, ok := jobs.Get("active"); !ok {
		t.Fatal("Get(active) ok = false, want true")
	}
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("Stat() error = %v, want not exist", err)
	}
}

func TestSweepExpiredDeletesJobsWithMissingFiles(t *testing.T) {
	_, jobs, config := testManager(t)
	now := time.Unix(100, 0)

	if err := jobs.Add(&Job{ID: "expired", Type: JobTypeScript, Status: StatusDone, Filename: "missing.sh", ExpiresAt: now.Add(-time.Second)}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	SweepExpired(jobs, config.JobsDir, now)

	if _, ok := jobs.Get("expired"); ok {
		t.Fatal("Get(expired) ok = true, want false")
	}
}
