package core

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestJobsAddGetDelete(t *testing.T) {
	jobs := NewJobs()
	job := &Job{
		ID:        "job-1",
		Type:      JobTypeZip,
		Status:    StatusPending,
		ExpiresAt: time.Unix(100, 0),
		Files:     []string{"alpha.txt"},
	}

	if err := jobs.Add(job); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if diff := cmp.Diff(job, got); diff != "" {
		t.Errorf("Get() mismatch (-want +got):\n%s", diff)
	}

	got.Files[0] = "changed.txt"
	again, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if diff := cmp.Diff([]string{"alpha.txt"}, again.Files); diff != "" {
		t.Errorf("Get() leaked internal state (-want +got):\n%s", diff)
	}

	jobs.Delete(job.ID)
	if _, ok := jobs.Get(job.ID); ok {
		t.Fatalf("Get(%q) ok = true, want false after delete", job.ID)
	}
}

func TestJobsAddRejectsDuplicateID(t *testing.T) {
	jobs := NewJobs()
	job := &Job{ID: "job-1", Type: JobTypeZip}

	if err := jobs.Add(job); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}
	if err := jobs.Add(job); !errors.Is(err, ErrJobExists) {
		t.Fatalf("second Add() error = %v, want %v", err, ErrJobExists)
	}
}

func TestJobsUpdate(t *testing.T) {
	jobs := NewJobs()
	job := &Job{
		ID:        "job-1",
		Type:      JobTypeZip,
		Status:    StatusPending,
		ExpiresAt: time.Unix(100, 0),
	}
	if err := jobs.Add(job); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	errBoom := errors.New("boom")
	if err := jobs.Update(job.ID, func(stored *Job) error {
		stored.Status = StatusProcessing
		stored.Progress = 42
		stored.Error = errBoom.Error()
		return nil
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if diff := cmp.Diff(&Job{
		ID:        "job-1",
		Type:      JobTypeZip,
		Status:    StatusProcessing,
		Progress:  42,
		ExpiresAt: time.Unix(100, 0),
		Error:     "boom",
	}, got); diff != "" {
		t.Errorf("updated job mismatch (-want +got):\n%s", diff)
	}
}

func TestJobsUpdateReturnsNotFound(t *testing.T) {
	jobs := NewJobs()

	err := jobs.Update("missing", func(job *Job) error {
		job.Status = StatusDone
		return nil
	})
	if !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, ErrJobNotFound)
	}
}

func TestJobsExpiredReturnsSnapshots(t *testing.T) {
	jobs := NewJobs()
	now := time.Unix(100, 0)
	if err := jobs.Add(&Job{ID: "expired", Type: JobTypeZip, ExpiresAt: now.Add(-time.Second), Files: []string{"alpha.txt"}}); err != nil {
		t.Fatalf("Add(expired) error = %v", err)
	}
	if err := jobs.Add(&Job{ID: "active", Type: JobTypeZip, ExpiresAt: now.Add(time.Second)}); err != nil {
		t.Fatalf("Add(active) error = %v", err)
	}

	expired := jobs.Expired(now)
	if len(expired) != 1 {
		t.Fatalf("Expired() len = %d, want 1", len(expired))
	}
	if diff := cmp.Diff("expired", expired[0].ID); diff != "" {
		t.Errorf("Expired() unexpected job (-want +got):\n%s", diff)
	}

	expired[0].Files[0] = "changed.txt"
	got, ok := jobs.Get("expired")
	if !ok {
		t.Fatal("Get(expired) ok = false, want true")
	}
	if diff := cmp.Diff([]string{"alpha.txt"}, got.Files); diff != "" {
		t.Errorf("Expired() leaked internal state (-want +got):\n%s", diff)
	}
}
