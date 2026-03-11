package core

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

func TestStoreSettersUpdateStoredJob(t *testing.T) {
	store := NewStore()
	job := &Job{ID: "job-1", Status: StatusPending, ExpiresAt: time.Now().Add(time.Minute)}
	store.Set(job)

	if !store.SetStatus(job.ID, StatusProcessing) {
		t.Fatalf("expected SetStatus to update existing job")
	}
	if job.Status != StatusProcessing {
		t.Fatalf("expected status %q, got %q", StatusProcessing, job.Status)
	}

	err := store.SetFailed(job.ID, assertErr("zip failed"))
	if !err {
		t.Fatalf("expected SetFailed to update existing job")
	}
	if job.Status != StatusFailed {
		t.Fatalf("expected status %q, got %q", StatusFailed, job.Status)
	}
	if job.Error != "zip failed" {
		t.Fatalf("expected error to be recorded, got %q", job.Error)
	}

	if !store.SetDone(job.ID, "job-1.zip") {
		t.Fatalf("expected SetDone to update existing job")
	}
	if job.Status != StatusDone {
		t.Fatalf("expected status %q, got %q", StatusDone, job.Status)
	}
	if job.Filename != "job-1.zip" {
		t.Fatalf("expected filename %q, got %q", "job-1.zip", job.Filename)
	}
	if job.Error != "" {
		t.Fatalf("expected error to be cleared, got %q", job.Error)
	}
}

func TestStoreSettersReturnFalseForMissingJob(t *testing.T) {
	store := NewStore()

	if store.SetStatus("missing", StatusProcessing) {
		t.Fatalf("expected SetStatus to fail for missing job")
	}
	if store.SetFailed("missing", assertErr("zip failed")) {
		t.Fatalf("expected SetFailed to fail for missing job")
	}
	if store.SetDone("missing", "missing.zip") {
		t.Fatalf("expected SetDone to fail for missing job")
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
