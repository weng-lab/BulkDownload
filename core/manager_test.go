package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appconfig "github.com/jair/bulkdownload/internal/config"
)

func TestDispatchZipJob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		seedExisting []string
		generatedIDs []string
		makeFiles    func(*testing.T, string) []string
		wantJob      Job
		wantCalls    int
	}{
		{
			name:         "creates and dispatches zip job",
			generatedIDs: []string{"codon-bam"},
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents")
				return []string{"alpha.txt"}
			},
			wantJob: Job{
				ID:     "codon-bam",
				Type:   JobTypeZip,
				Status: StatusPending,
			},
			wantCalls: 1,
		},
		{
			name:         "retries duplicate id",
			seedExisting: []string{"allele-atac"},
			generatedIDs: []string{"allele-atac", "codon-bam"},
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents")
				return []string{"alpha.txt"}
			},
			wantJob: Job{
				ID:     "codon-bam",
				Type:   JobTypeZip,
				Status: StatusPending,
			},
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			config := appconfig.Config{
				JobsDir:       filepath.Join(t.TempDir(), "jobs"),
				SourceRootDir: root,
				JobTTL:        3 * time.Second,
			}
			if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
			}

			jobs := NewJobs()
			for _, id := range tt.seedExisting {
				if err := jobs.Add(Job{ID: id, Type: JobTypeZip, ExpiresAt: time.Unix(100, 0)}); err != nil {
					t.Fatalf("Add(%q) error = %v", id, err)
				}
			}

			calls := 0
			manager := newManager(jobs, config, func() string {
				id := tt.generatedIDs[calls]
				calls++
				return id
			})

			files := tt.makeFiles(t, root)
			job, err := manager.DispatchZipJob(files)
			if err != nil {
				t.Fatalf("DispatchZipJob() error = %v", err)
			}

			if job.ID == "" {
				t.Fatal("DispatchZipJob() returned empty job")
			}

			want := tt.wantJob
			want.Files = files
			want.ExpiresAt = job.ExpiresAt

			if diff := cmp.Diff(want, job, cmpopts.EquateApproxTime(time.Second)); diff != "" {
				t.Errorf("DispatchZipJob() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.wantCalls, calls); diff != "" {
				t.Errorf("generateID call count mismatch (-want +got):\n%s", diff)
			}

			// Wait for job to complete
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				stored, ok := jobs.Get(job.ID)
				if ok && stored.Status == StatusDone {
					return
				}
				if ok && stored.Status == StatusFailed {
					t.Fatalf("job failed: %s", stored.Error)
				}
				time.Sleep(25 * time.Millisecond)
			}
			t.Fatalf("timed out waiting for job %s to complete", job.ID)
		})
	}
}

func TestDispatchTarballJob(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	config := appconfig.Config{
		JobsDir:       filepath.Join(t.TempDir(), "jobs"),
		SourceRootDir: root,
		JobTTL:        3 * time.Second,
	}
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
	}

	writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents")
	files := []string{"alpha.txt"}

	jobs := NewJobs()
	manager := newManager(jobs, config, func() string { return "codon-bam" })

	job, err := manager.DispatchTarballJob(files)
	if err != nil {
		t.Fatalf("DispatchTarballJob() error = %v", err)
	}

	if job.ID != "codon-bam" {
		t.Errorf("job.ID = %q, want %q", job.ID, "codon-bam")
	}
	if job.Type != JobTypeTarball {
		t.Errorf("job.Type = %q, want %q", job.Type, JobTypeTarball)
	}
	if job.Status != StatusPending {
		t.Errorf("job.Status = %q, want %q", job.Status, StatusPending)
	}

	// Wait for job to complete
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, ok := jobs.Get(job.ID)
		if ok && stored.Status == StatusDone {
			return
		}
		if ok && stored.Status == StatusFailed {
			t.Fatalf("job failed: %s", stored.Error)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to complete", job.ID)
}

func TestDispatchScriptJob(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	config := appconfig.Config{
		JobsDir:         filepath.Join(t.TempDir(), "jobs"),
		SourceRootDir:   root,
		PublicBaseURL:   "https://download.mohd.org",
		DownloadRootDir: "mohd_data",
		JobTTL:          3 * time.Second,
	}
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
	}

	writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents")
	files := []string{"alpha.txt"}

	jobs := NewJobs()
	manager := newManager(jobs, config, func() string { return "codon-bam" })

	job, err := manager.DispatchScriptJob(files)
	if err != nil {
		t.Fatalf("DispatchScriptJob() error = %v", err)
	}

	if job.ID != "codon-bam" {
		t.Errorf("job.ID = %q, want %q", job.ID, "codon-bam")
	}
	if job.Type != JobTypeScript {
		t.Errorf("job.Type = %q, want %q", job.Type, JobTypeScript)
	}
	if job.Status != StatusPending {
		t.Errorf("job.Status = %q, want %q", job.Status, StatusPending)
	}

	// Wait for job to complete
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, ok := jobs.Get(job.ID)
		if ok && stored.Status == StatusDone {
			return
		}
		if ok && stored.Status == StatusFailed {
			t.Fatalf("job failed: %s", stored.Error)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to complete", job.ID)
}

func TestDispatchJobReturnsStoredSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dispatch func(*Manager, []string) (Job, error)
	}{
		{
			name:     "zip job",
			dispatch: (*Manager).DispatchZipJob,
		},
		{
			name:     "tarball job",
			dispatch: (*Manager).DispatchTarballJob,
		},
		{
			name:     "script job",
			dispatch: (*Manager).DispatchScriptJob,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := testConfig(t)
			config.SourceRootDir = t.TempDir()
			if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
			}

			writeTestFile(t, filepath.Join(config.SourceRootDir, "file.txt"), "contents")
			files := []string{"file.txt"}

			jobs := NewJobs()
			manager := newManager(jobs, config, func() string { return "codon-bam" })

			job, err := tt.dispatch(manager, files)
			if err != nil {
				t.Fatalf("dispatch() error = %v", err)
			}
			if job.ID == "" {
				t.Fatal("dispatch returned empty job")
			}

			job.Files[0] = "changed.txt"

			stored, ok := jobs.Get(job.ID)
			if !ok {
				t.Fatalf("Get(%q) ok = false, want true", job.ID)
			}
			if diff := cmp.Diff([]string{"file.txt"}, stored.Files); diff != "" {
				t.Errorf("stored job files mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDispatchJobCopiesRequestedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jobType  JobType
		dispatch func(*Manager, []string) (Job, error)
	}{
		{
			name:     "zip job",
			jobType:  JobTypeZip,
			dispatch: (*Manager).DispatchZipJob,
		},
		{
			name:     "tarball job",
			jobType:  JobTypeTarball,
			dispatch: (*Manager).DispatchTarballJob,
		},
		{
			name:     "script job",
			jobType:  JobTypeScript,
			dispatch: (*Manager).DispatchScriptJob,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := testConfig(t)
			config.SourceRootDir = t.TempDir()
			if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
			}
			writeTestFile(t, filepath.Join(config.SourceRootDir, "reads", "sample.fastq"), "reads")
			writeTestFile(t, filepath.Join(config.SourceRootDir, "variants", "sample.vcf"), "variants")
			files := []string{"reads/sample.fastq", "variants/sample.vcf"}

			manager := newManager(NewJobs(), config, func() string { return "codon-bam" })

			job, err := tt.dispatch(manager, files)
			if err != nil {
				t.Fatalf("dispatch(%q) error = %v", tt.jobType, err)
			}
			if job.ID == "" {
				t.Fatal("dispatch returned empty job")
			}

			// Verify files are copied
			if diff := cmp.Diff([]string{"reads/sample.fastq", "variants/sample.vcf"}, job.Files); diff != "" {
				t.Errorf("job.Files mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.jobType, job.Type); diff != "" {
				t.Errorf("job.Type mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(StatusPending, job.Status); diff != "" {
				t.Errorf("job.Status mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDispatchJobMarksFailureWhenExecutionFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jobType  JobType
		dispatch func(*Manager, []string) (Job, error)
	}{
		{
			name:     "zip job",
			jobType:  JobTypeZip,
			dispatch: (*Manager).DispatchZipJob,
		},
		{
			name:     "tarball job",
			jobType:  JobTypeTarball,
			dispatch: (*Manager).DispatchTarballJob,
		},
		{
			name:     "script job",
			jobType:  JobTypeScript,
			dispatch: (*Manager).DispatchScriptJob,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			config := appconfig.Config{
				JobsDir:         filepath.Join(root, "missing", "jobs"),
				SourceRootDir:   filepath.Join(root, "source"),
				PublicBaseURL:   "https://download.mohd.org",
				DownloadRootDir: "mohd_data",
				JobTTL:          3 * time.Second,
			}
			if err := os.MkdirAll(config.SourceRootDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", config.SourceRootDir, err)
			}
			writeTestFile(t, filepath.Join(config.SourceRootDir, "alpha.txt"), "alpha contents")

			jobs := NewJobs()
			manager := newManager(jobs, config, func() string { return "codon-bam" })

			job, err := tt.dispatch(manager, []string{"alpha.txt"})
			if err != nil {
				t.Fatalf("dispatch(%q) error = %v", tt.jobType, err)
			}
			if diff := cmp.Diff(StatusPending, job.Status); diff != "" {
				t.Errorf("job.Status mismatch (-want +got):\n%s", diff)
			}

			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				stored, ok := jobs.Get(job.ID)
				if ok && stored.Status == StatusFailed {
					if stored.Error == "" {
						t.Fatal("expected failed job to record an error")
					}
					if stored.Filename != "" {
						t.Fatalf("expected failed job filename to be empty, got %q", stored.Filename)
					}
					return
				}
				time.Sleep(25 * time.Millisecond)
			}

			t.Fatalf("timed out waiting for job %s to fail", job.ID)
		})
	}
}

func TestDispatchZipJobReturnsErrorWhenJobCreationFails(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	jobs := NewJobs()
	if err := jobs.Add(Job{ID: "allele-atac", Type: JobTypeZip, ExpiresAt: time.Unix(100, 0)}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	manager := newManager(jobs, config, func() string { return "allele-atac" })

	job, err := manager.DispatchZipJob([]string{"reads/sample.fastq"})
	if err == nil {
		t.Fatal("DispatchZipJob() error = nil, want non-nil")
	}
	if diff := cmp.Diff(Job{}, job); diff != "" {
		t.Errorf("DispatchZipJob() job mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("create zip job: generate job id: exhausted retries", err.Error()); diff != "" {
		t.Errorf("DispatchZipJob() error mismatch (-want +got):\n%s", diff)
	}
}
