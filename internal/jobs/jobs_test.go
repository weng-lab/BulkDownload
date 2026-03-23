package jobs

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestJobs_AddAndGet(t *testing.T) {
	t.Parallel()

	jobs := NewJobs()
	job := Job{
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
}

func TestJobs_GetReturnsSnapshot(t *testing.T) {
	t.Parallel()

	jobs := NewJobs()
	job := Job{
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
	got.Files[0] = "changed.txt"

	again, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if diff := cmp.Diff([]string{"alpha.txt"}, again.Files); diff != "" {
		t.Errorf("Get() leaked internal state (-want +got):\n%s", diff)
	}
}

func TestJobs_DeleteRemovesJob(t *testing.T) {
	t.Parallel()

	jobs := NewJobs()
	job := Job{ID: "job-1", Type: JobTypeZip}
	if err := jobs.Add(job); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	jobs.Delete(job.ID)

	if _, ok := jobs.Get(job.ID); ok {
		t.Fatalf("Get(%q) ok = true, want false after delete", job.ID)
	}
}

func TestJobs_AddRejectsInvalidJobs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		job     Job
		wantErr error
	}{
		{
			name:    "empty id",
			job:     Job{Type: JobTypeZip},
			wantErr: ErrInvalidJobID,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jobs := NewJobs()
			err := jobs.Add(tt.job)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Add() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobs_AddRejectsDuplicateID(t *testing.T) {
	t.Parallel()

	jobs := NewJobs()
	job := Job{ID: "job-1", Type: JobTypeZip}

	if err := jobs.Add(job); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}
	if err := jobs.Add(job); !errors.Is(err, ErrJobExists) {
		t.Fatalf("second Add() error = %v, want %v", err, ErrJobExists)
	}
}

func TestJobs_MarkProcessing(t *testing.T) {
	t.Parallel()

	jobs := NewJobs()
	job := addLifecycleTestJob(t, jobs)

	if err := jobs.MarkProcessing(job.ID); err != nil {
		t.Fatalf("MarkProcessing() error = %v", err)
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}

	want := job
	want.Status = StatusProcessing
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MarkProcessing() mismatch (-want +got):\n%s", diff)
	}
}

func TestJobs_SetProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		progress     int
		wantProgress int
	}{
		{
			name:         "clamps negative progress",
			progress:     -5,
			wantProgress: 0,
		},
		{
			name:         "keeps in-range progress",
			progress:     42,
			wantProgress: 42,
		},
		{
			name:         "clamps progress above one hundred",
			progress:     150,
			wantProgress: 100,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jobs := NewJobs()
			job := addLifecycleTestJob(t, jobs)

			if err := jobs.SetProgress(job.ID, tt.progress); err != nil {
				t.Fatalf("SetProgress() error = %v", err)
			}

			got, ok := jobs.Get(job.ID)
			if !ok {
				t.Fatalf("Get(%q) ok = false, want true", job.ID)
			}

			want := job
			want.Progress = tt.wantProgress
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("SetProgress() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestJobs_MarkFailed(t *testing.T) {
	t.Parallel()

	jobs := NewJobs()
	job := addLifecycleTestJob(t, jobs)

	errBoom := errors.New("boom")
	if err := jobs.MarkFailed(job.ID, errBoom); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}

	want := job
	want.Status = StatusFailed
	want.Error = errBoom.Error()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MarkFailed() mismatch (-want +got):\n%s", diff)
	}
}

func TestJobs_MarkDone(t *testing.T) {
	t.Parallel()

	jobs := NewJobs()
	job := addLifecycleTestJob(t, jobs)
	if err := jobs.SetProgress(job.ID, 42); err != nil {
		t.Fatalf("SetProgress() error = %v", err)
	}
	if err := jobs.MarkFailed(job.ID, errors.New("boom")); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}

	if err := jobs.MarkDone(job.ID, "job-1.zip"); err != nil {
		t.Fatalf("MarkDone() error = %v", err)
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}

	want := job
	want.Status = StatusDone
	want.Progress = 42
	want.Filename = "job-1.zip"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MarkDone() mismatch (-want +got):\n%s", diff)
	}
}

func TestJobs_LifecycleMethodsReturnNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*Jobs) error
	}{
		{
			name: "MarkProcessing",
			run: func(jobs *Jobs) error {
				return jobs.MarkProcessing("missing")
			},
		},
		{
			name: "SetProgress",
			run: func(jobs *Jobs) error {
				return jobs.SetProgress("missing", 42)
			},
		},
		{
			name: "MarkFailed",
			run: func(jobs *Jobs) error {
				return jobs.MarkFailed("missing", errors.New("boom"))
			},
		},
		{
			name: "MarkDone",
			run: func(jobs *Jobs) error {
				return jobs.MarkDone("missing", "job-1.zip")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jobs := NewJobs()
			if err := tt.run(jobs); !errors.Is(err, ErrJobNotFound) {
				t.Fatalf("%s() error = %v, want %v", tt.name, err, ErrJobNotFound)
			}
		})
	}
}

func TestJobs_ExpiredReturnsSnapshots(t *testing.T) {
	t.Parallel()

	jobs := NewJobs()
	now := time.Unix(100, 0)
	if err := jobs.Add(Job{ID: "expired", Type: JobTypeZip, ExpiresAt: now.Add(-time.Second), Files: []string{"alpha.txt"}}); err != nil {
		t.Fatalf("Add(expired) error = %v", err)
	}
	if err := jobs.Add(Job{ID: "active", Type: JobTypeZip, ExpiresAt: now.Add(time.Second)}); err != nil {
		t.Fatalf("Add(active) error = %v", err)
	}

	expired := jobs.Expired(now)
	if diff := cmp.Diff(1, len(expired)); diff != "" {
		t.Fatalf("Expired() len mismatch (-want +got):\n%s", diff)
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

func addLifecycleTestJob(t *testing.T, jobs *Jobs) Job {
	t.Helper()

	job := Job{
		ID:        "job-1",
		Type:      JobTypeZip,
		Status:    StatusPending,
		ExpiresAt: time.Unix(100, 0),
	}
	if err := jobs.Add(job); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	return job
}
