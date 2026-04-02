package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jair/bulkdownload/internal/jobs"
)

func TestCreateJobRoutesSupportedRequestsThroughCreateAndDispatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		jobType        string
		files          []string
		setup          func(*testing.T, string)
		wantType       jobs.JobType
		wantProgress   int
		wantNameSuffix string
	}{
		{
			name:    "zip",
			jobType: "zip",
			files:   []string{"nested/alpha.txt"},
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "nested", "alpha.txt"), "alpha contents")
			},
			wantType:       jobs.JobTypeZip,
			wantProgress:   100,
			wantNameSuffix: ".zip",
		},
		{
			name:    "tarball",
			jobType: "tarball",
			files:   []string{"nested/alpha.txt", "nested/bravo.txt"},
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "nested", "alpha.txt"), "alpha contents")
				writeTestFile(t, filepath.Join(root, "nested", "bravo.txt"), "bravo contents")
			},
			wantType:       jobs.JobTypeTarball,
			wantProgress:   100,
			wantNameSuffix: ".tar.gz",
		},
		{
			name:    "script",
			jobType: "script",
			files:   []string{"rna/accession.bigwig", "dna/sample.cram"},
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "rna", "accession.bigwig"), "rna data")
				writeTestFile(t, filepath.Join(root, "dna", "sample.cram"), "dna data")
			},
			wantType:       jobs.JobTypeScript,
			wantProgress:   0,
			wantNameSuffix: ".sh",
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
			tt.setup(t, fixture.config.SourceRootDir)

			job, err := fixture.manager.CreateJob(tt.jobType, tt.files)
			if err != nil {
				t.Fatalf("CreateJob() error = %v", err)
			}

			want := jobs.Job{
				ID:        job.ID,
				Type:      tt.wantType,
				Status:    jobs.StatusPending,
				CreatedAt: job.CreatedAt,
				ExpiresAt: job.ExpiresAt,
				Files:     tt.files,
				InputSize: job.InputSize,
			}
			if diff := cmp.Diff(want, job, cmpopts.EquateApproxTime(time.Second)); diff != "" {
				t.Errorf("CreateJob() mismatch (-want +got):\n%s", diff)
			}
			if job.CreatedAt.IsZero() {
				t.Fatal("CreateJob() created at is zero")
			}
			if job.InputSize <= 0 {
				t.Fatalf("CreateJob() input size = %d, want positive", job.InputSize)
			}

			stored := waitForStoredJob(t, fixture.jobs, job.ID)
			if diff := cmp.Diff(tt.wantType, stored.Type); diff != "" {
				t.Errorf("stored job type mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantProgress, stored.Progress); diff != "" {
				t.Errorf("stored job progress mismatch (-want +got):\n%s", diff)
			}
			if !strings.HasSuffix(stored.Filename, tt.wantNameSuffix) {
				t.Fatalf("stored job filename = %q, want suffix %q", stored.Filename, tt.wantNameSuffix)
			}
			artifactPath := filepath.Join(fixture.config.JobsDir, stored.Filename)
			info, err := os.Stat(artifactPath)
			if err != nil {
				t.Fatalf("Stat(%q) error = %v", filepath.Join(fixture.config.JobsDir, stored.Filename), err)
			}
			if diff := cmp.Diff(stored.InputSize, job.InputSize); diff != "" {
				t.Errorf("stored job input size mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(info.Size(), stored.OutputSize); diff != "" {
				t.Errorf("stored job output size mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateJobReturnsValidationErrors(t *testing.T) {
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
		{
			name:    "tarball missing file",
			jobType: "tarball",
			files:   []string{"nested/missing.txt"},
			wantErr: "file not found: nested/missing.txt",
		},
		{
			name:    "script missing file",
			jobType: "script",
			files:   []string{"nested/missing.txt"},
			wantErr: "file not found: nested/missing.txt",
		},
		{
			name:    "tarball absolute path",
			jobType: "tarball",
			files:   []string{"/tmp/source/nested/alpha.txt"},
			wantErr: "absolute paths are not allowed: /tmp/source/nested/alpha.txt",
		},
		{
			name:    "script absolute path",
			jobType: "script",
			files:   []string{"/tmp/source/nested/alpha.txt"},
			wantErr: "absolute paths are not allowed: /tmp/source/nested/alpha.txt",
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
			if !errors.Is(err, ErrCreateJobRequest) {
				t.Fatalf("CreateJob() error = %v, want request sentinel", err)
			}
			if diff := cmp.Diff(tt.wantErr, strings.TrimPrefix(err.Error(), ErrCreateJobRequest.Error()+": ")); diff != "" {
				t.Errorf("CreateJob() error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateJobTrimsFilePathsBeforeDispatch(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	fixture.config.SourceRootDir = t.TempDir()
	fixture.manager.sourceRootDir = fixture.config.SourceRootDir
	if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
	}

	writeTestFile(t, filepath.Join(fixture.config.SourceRootDir, "nested", "alpha.txt"), "alpha contents")

	job, err := fixture.manager.CreateJob("zip", []string{"  nested/alpha.txt  "})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	if diff := cmp.Diff([]string{"nested/alpha.txt"}, job.Files); diff != "" {
		t.Errorf("CreateJob() files mismatch (-want +got):\n%s", diff)
	}

	stored := waitForStoredJob(t, fixture.jobs, job.ID)
	if diff := cmp.Diff([]string{"nested/alpha.txt"}, stored.Files); diff != "" {
		t.Errorf("stored job files mismatch (-want +got):\n%s", diff)
	}
}

func waitForStoredJob(t *testing.T, jobStore *jobs.Jobs, id string) jobs.Job {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, ok := jobStore.Get(id)
		if ok && stored.Status == jobs.StatusDone {
			return stored
		}
		if ok && stored.Status == jobs.StatusFailed {
			t.Fatalf("job failed: %s", stored.Error)
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for job %s to complete", id)
	return jobs.Job{}
}
