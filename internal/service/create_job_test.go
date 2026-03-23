package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jair/bulkdownload/internal/jobs"
)

func TestCreateJobDispatchesZipRequest(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	fixture.config.SourceRootDir = t.TempDir()
	fixture.manager.sourceRootDir = fixture.config.SourceRootDir
	if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
	}
	writeTestFile(t, filepath.Join(fixture.config.SourceRootDir, "nested", "alpha.txt"), "alpha contents")

	job, err := fixture.manager.CreateJob("zip", []string{"nested/alpha.txt"})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	want := jobs.Job{
		ID:        job.ID,
		Type:      jobs.JobTypeZip,
		Status:    jobs.StatusPending,
		ExpiresAt: job.ExpiresAt,
		Files:     []string{"nested/alpha.txt"},
	}
	if diff := cmp.Diff(want, job, cmpopts.EquateApproxTime(time.Second)); diff != "" {
		t.Errorf("CreateJob() mismatch (-want +got):\n%s", diff)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, ok := fixture.jobs.Get(job.ID)
		if ok && stored.Status == jobs.StatusDone {
			return
		}
		if ok && stored.Status == jobs.StatusFailed {
			t.Fatalf("job failed: %s", stored.Error)
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for job %s to complete", job.ID)
}

func TestCreateJobReturnsTypedRequestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jobType string
		files   []string
		setup   func(*testing.T, *Manager)
		wantErr string
	}{
		{
			name:    "invalid job type",
			jobType: "invalid",
			files:   []string{"nested/alpha.txt"},
			wantErr: "invalid job type: invalid",
		},
		{
			name:    "empty file path",
			jobType: "zip",
			files:   []string{"   "},
			wantErr: "file path cannot be empty",
		},
		{
			name:    "absolute path",
			jobType: "zip",
			files:   []string{"/tmp/source/nested/alpha.txt"},
			wantErr: "absolute paths are not allowed: /tmp/source/nested/alpha.txt",
		},
		{
			name:    "missing file",
			jobType: "zip",
			files:   []string{"nested/missing.txt"},
			wantErr: "file not found: nested/missing.txt",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newTestFixture(t)
			fixture.config.SourceRootDir = t.TempDir()
			fixture.manager.sourceRootDir = fixture.config.SourceRootDir
			if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
			}
			if tt.setup != nil {
				tt.setup(t, fixture.manager)
			}

			job, err := fixture.manager.CreateJob(tt.jobType, tt.files)
			if diff := cmp.Diff(jobs.Job{}, job); diff != "" {
				t.Errorf("CreateJob() job mismatch (-want +got):\n%s", diff)
			}
			if err == nil {
				t.Fatal("CreateJob() error = nil, want non-nil")
			}
			if !IsCreateJobRequestError(err) {
				t.Fatalf("CreateJob() error type = %T, want request error", err)
			}
			if diff := cmp.Diff(tt.wantErr, err.Error()); diff != "" {
				t.Errorf("CreateJob() error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
