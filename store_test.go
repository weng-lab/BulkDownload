package main

import (
	"testing"
	"time"
)

func TestStoreSetGetDelete(t *testing.T) {
	store := NewStore()
	job := &Job{ID: "job-1", Status: StatusPending, ExpiresAt: time.Now().Add(time.Minute)}

	store.Set(job)

	got, ok := store.Get(job.ID)
	if !ok {
		t.Fatalf("expected job %q to exist", job.ID)
	}
	if got != job {
		t.Fatalf("expected stored job pointer to match original")
	}

	store.Delete(job.ID)

	if _, ok := store.Get(job.ID); ok {
		t.Fatalf("expected job %q to be deleted", job.ID)
	}
}

func TestStoreExpiredReturnsOnlyExpiredJobs(t *testing.T) {
	store := NewStore()
	expired := &Job{ID: "expired", ExpiresAt: time.Now().Add(-time.Second)}
	active := &Job{ID: "active", ExpiresAt: time.Now().Add(time.Second)}

	store.Set(expired)
	store.Set(active)

	jobs := store.Expired()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 expired job, got %d", len(jobs))
	}
	if jobs[0].ID != expired.ID {
		t.Fatalf("expected expired job %q, got %q", expired.ID, jobs[0].ID)
	}
}
