package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appconfig "github.com/jair/bulkdownload/internal/config"
	jobstore "github.com/jair/bulkdownload/internal/jobs"
)

func TestCreateJobDispatchesTypedJobs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		rawType      string
		seedExisting []string
		generatedIDs []string
		makeFiles    func(*testing.T, string) []string
		wantJob      jobstore.Job
		wantCalls    int
	}{
		{
			name:         "creates and dispatches zip job",
			rawType:      "zip",
			generatedIDs: []string{"codon-bam"},
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents")
				return []string{"alpha.txt"}
			},
			wantJob: jobstore.Job{
				ID:     "codon-bam",
				Type:   jobstore.JobTypeZip,
				Status: jobstore.StatusPending,
			},
			wantCalls: 1,
		},
		{
			name:         "retries duplicate id",
			rawType:      "zip",
			seedExisting: []string{"allele-atac"},
			generatedIDs: []string{"allele-atac", "codon-bam"},
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents")
				return []string{"alpha.txt"}
			},
			wantJob: jobstore.Job{
				ID:     "codon-bam",
				Type:   jobstore.JobTypeZip,
				Status: jobstore.StatusPending,
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

			jobs := jobstore.NewJobs()
			for _, id := range tt.seedExisting {
				if err := jobs.Add(jobstore.Job{ID: id, Type: jobstore.JobTypeZip, ExpiresAt: time.Unix(100, 0)}); err != nil {
					t.Fatalf("Add(%q) error = %v", id, err)
				}
			}

			calls := 0
			manager := newManager(jobs, config, func() string {
				id := tt.generatedIDs[calls]
				calls++
				return id
			})
			t.Cleanup(manager.Shutdown)

			files := tt.makeFiles(t, root)
			job, err := manager.CreateJob(tt.rawType, files)
			if err != nil {
				t.Fatalf("CreateJob() error = %v", err)
			}

			if job.ID == "" {
				t.Fatal("CreateJob() returned empty job")
			}

			want := tt.wantJob
			want.Files = files
			want.CreatedAt = job.CreatedAt
			want.ExpiresAt = job.ExpiresAt
			want.InputSize = job.InputSize

			if diff := cmp.Diff(want, job, cmpopts.EquateApproxTime(time.Second)); diff != "" {
				t.Errorf("CreateJob() mismatch (-want +got):\n%s", diff)
			}
			if job.CreatedAt.IsZero() {
				t.Fatal("CreateJob() created at is zero")
			}
			if job.InputSize <= 0 {
				t.Fatalf("CreateJob() input size = %d, want positive", job.InputSize)
			}

			if diff := cmp.Diff(tt.wantCalls, calls); diff != "" {
				t.Errorf("generateID call count mismatch (-want +got):\n%s", diff)
			}

			// Wait for job to complete
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				stored, ok := jobs.Get(job.ID)
				if ok && stored.Status == jobstore.StatusDone {
					return
				}
				if ok && stored.Status == jobstore.StatusFailed {
					t.Fatalf("job failed: %s", stored.Error)
				}
				time.Sleep(25 * time.Millisecond)
			}
			t.Fatalf("timed out waiting for job %s to complete", job.ID)
		})
	}
}

func TestCreateJobTarball(t *testing.T) {
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

	jobs := jobstore.NewJobs()
	manager := newManager(jobs, config, func() string { return "codon-bam" })
	t.Cleanup(manager.Shutdown)

	job, err := manager.CreateJob("tarball", files)
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	if job.ID != "codon-bam" {
		t.Errorf("job.ID = %q, want %q", job.ID, "codon-bam")
	}
	if job.Type != jobstore.JobTypeTarball {
		t.Errorf("job.Type = %q, want %q", job.Type, jobstore.JobTypeTarball)
	}
	if job.Status != jobstore.StatusPending {
		t.Errorf("job.Status = %q, want %q", job.Status, jobstore.StatusPending)
	}
	if job.CreatedAt.IsZero() {
		t.Fatal("job.CreatedAt is zero")
	}
	if job.InputSize <= 0 {
		t.Fatalf("job.InputSize = %d, want positive", job.InputSize)
	}

	// Wait for job to complete
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, ok := jobs.Get(job.ID)
		if ok && stored.Status == jobstore.StatusDone {
			return
		}
		if ok && stored.Status == jobstore.StatusFailed {
			t.Fatalf("job failed: %s", stored.Error)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to complete", job.ID)
}

func TestCreateJobScript(t *testing.T) {
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

	jobs := jobstore.NewJobs()
	manager := newManager(jobs, config, func() string { return "codon-bam" })
	t.Cleanup(manager.Shutdown)

	job, err := manager.CreateJob("script", files)
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	if job.ID != "codon-bam" {
		t.Errorf("job.ID = %q, want %q", job.ID, "codon-bam")
	}
	if job.Type != jobstore.JobTypeScript {
		t.Errorf("job.Type = %q, want %q", job.Type, jobstore.JobTypeScript)
	}
	if job.Status != jobstore.StatusPending {
		t.Errorf("job.Status = %q, want %q", job.Status, jobstore.StatusPending)
	}
	if job.CreatedAt.IsZero() {
		t.Fatal("job.CreatedAt is zero")
	}
	if job.InputSize <= 0 {
		t.Fatalf("job.InputSize = %d, want positive", job.InputSize)
	}

	// Wait for job to complete
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, ok := jobs.Get(job.ID)
		if ok && stored.Status == jobstore.StatusDone {
			return
		}
		if ok && stored.Status == jobstore.StatusFailed {
			t.Fatalf("job failed: %s", stored.Error)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to complete", job.ID)
}

func TestCreateJobReturnsStoredSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawType string
	}{
		{
			name:    "zip job",
			rawType: "zip",
		},
		{
			name:    "tarball job",
			rawType: "tarball",
		},
		{
			name:    "script job",
			rawType: "script",
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

			jobs := jobstore.NewJobs()
			manager := newManager(jobs, config, func() string { return "codon-bam" })
			t.Cleanup(manager.Shutdown)

			job, err := manager.CreateJob(tt.rawType, files)
			if err != nil {
				t.Fatalf("CreateJob() error = %v", err)
			}
			if job.ID == "" {
				t.Fatal("CreateJob returned empty job")
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
		name    string
		rawType string
		jobType jobstore.JobType
	}{
		{
			name:    "zip job",
			rawType: "zip",
			jobType: jobstore.JobTypeZip,
		},
		{
			name:    "tarball job",
			rawType: "tarball",
			jobType: jobstore.JobTypeTarball,
		},
		{
			name:    "script job",
			rawType: "script",
			jobType: jobstore.JobTypeScript,
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

			manager := newManager(jobstore.NewJobs(), config, func() string { return "codon-bam" })
			t.Cleanup(manager.Shutdown)

			job, err := manager.CreateJob(tt.rawType, files)
			if err != nil {
				t.Fatalf("CreateJob(%q) error = %v", tt.rawType, err)
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
			if diff := cmp.Diff(jobstore.StatusPending, job.Status); diff != "" {
				t.Errorf("job.Status mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDispatchJobMarksFailureWhenExecutionFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawType string
		jobType jobstore.JobType
	}{
		{
			name:    "zip job",
			rawType: "zip",
			jobType: jobstore.JobTypeZip,
		},
		{
			name:    "tarball job",
			rawType: "tarball",
			jobType: jobstore.JobTypeTarball,
		},
		{
			name:    "script job",
			rawType: "script",
			jobType: jobstore.JobTypeScript,
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

			jobs := jobstore.NewJobs()
			manager := newManager(jobs, config, func() string { return "codon-bam" })
			t.Cleanup(manager.Shutdown)

			job, err := manager.CreateJob(tt.rawType, []string{"alpha.txt"})
			if err != nil {
				t.Fatalf("CreateJob(%q) error = %v", tt.rawType, err)
			}
			if diff := cmp.Diff(jobstore.StatusPending, job.Status); diff != "" {
				t.Errorf("job.Status mismatch (-want +got):\n%s", diff)
			}

			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				stored, ok := jobs.Get(job.ID)
				if ok && stored.Status == jobstore.StatusFailed {
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

func TestDispatchJobsWaitForSharedExecutionSlot(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	config.SourceRootDir = t.TempDir()
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
	}
	writeTestFile(t, filepath.Join(config.SourceRootDir, "reads", "sample.fastq"), "reads")

	store := jobstore.NewJobs()
	manager := newManager(store, config, nil)
	t.Cleanup(manager.Shutdown)

	for range maxConcurrentJobs {
		manager.sem <- struct{}{}
	}

	rawTypes := []string{
		"zip",
		"script",
		"tarball",
		"zip",
		"script",
	}

	jobsByOrder := make([]jobstore.Job, 0, len(rawTypes))
	for _, rawType := range rawTypes {
		job, err := manager.CreateJob(rawType, []string{"reads/sample.fastq"})
		if err != nil {
			t.Fatalf("CreateJob() error = %v", err)
		}
		jobsByOrder = append(jobsByOrder, job)
	}

	for _, job := range jobsByOrder {
		stored, ok := store.Get(job.ID)
		if !ok {
			t.Fatalf("Get(%q) ok = false, want true", job.ID)
		}
		if diff := cmp.Diff(jobstore.StatusPending, stored.Status); diff != "" {
			t.Errorf("job %s status mismatch (-want +got):\n%s", job.ID, diff)
		}
	}

	<-manager.sem

	if _, err := waitForAnyActiveJob(store, jobsByOrder, 2*time.Second); err != nil {
		t.Fatal(err)
	}

	for range maxConcurrentJobs - 1 {
		<-manager.sem
	}

	for _, job := range jobsByOrder {
		if _, err := waitForJobStatus(store, job.ID, 2*time.Second, jobstore.StatusDone); err != nil {
			t.Fatal(err)
		}
	}
}

func TestManagerShutdownStopsQueuedJobs(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	config.SourceRootDir = t.TempDir()
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
	}
	writeTestFile(t, filepath.Join(config.SourceRootDir, "reads", "sample.fastq"), "reads")

	store := jobstore.NewJobs()
	manager := newManager(store, config, nil)

	for range maxConcurrentJobs {
		manager.sem <- struct{}{}
	}

	rawTypes := []string{
		"zip",
		"tarball",
		"script",
	}

	jobsByOrder := make([]jobstore.Job, 0, len(rawTypes))
	for _, rawType := range rawTypes {
		job, err := manager.CreateJob(rawType, []string{"reads/sample.fastq"})
		if err != nil {
			t.Fatalf("CreateJob() error = %v", err)
		}
		jobsByOrder = append(jobsByOrder, job)
	}

	done := make(chan struct{})
	go func() {
		manager.Shutdown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for manager shutdown")
	}

	for _, job := range jobsByOrder {
		stored, ok := store.Get(job.ID)
		if !ok {
			t.Fatalf("Get(%q) ok = false, want true", job.ID)
		}
		if diff := cmp.Diff(jobstore.StatusPending, stored.Status); diff != "" {
			t.Errorf("job %s status mismatch (-want +got):\n%s", job.ID, diff)
		}
	}
}

func waitForAnyActiveJob(store *jobstore.Jobs, jobs []jobstore.Job, timeout time.Duration) (jobstore.Job, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, job := range jobs {
			stored, ok := store.Get(job.ID)
			if ok && stored.Status != jobstore.StatusPending {
				return stored, nil
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	return jobstore.Job{}, fmt.Errorf("timed out waiting for any job to leave pending")
}

func waitForJobStatus(store *jobstore.Jobs, jobID string, timeout time.Duration, want jobstore.JobStatus) (jobstore.Job, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		stored, ok := store.Get(jobID)
		if ok && stored.Status == want {
			return stored, nil
		}
		time.Sleep(25 * time.Millisecond)
	}

	return jobstore.Job{}, fmt.Errorf("timed out waiting for job %s to reach status %q", jobID, want)
}

func TestCreateJobReturnsErrorWhenJobCreationFails(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	config.SourceRootDir = t.TempDir()
	writeTestFile(t, filepath.Join(config.SourceRootDir, "reads", "sample.fastq"), "reads")
	jobs := jobstore.NewJobs()
	if err := jobs.Add(jobstore.Job{ID: "allele-atac", Type: jobstore.JobTypeZip, ExpiresAt: time.Unix(100, 0)}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	manager := newManager(jobs, config, func() string { return "allele-atac" })
	t.Cleanup(manager.Shutdown)

	job, err := manager.CreateJob("zip", []string{"reads/sample.fastq"})
	if err == nil {
		t.Fatal("CreateJob() error = nil, want non-nil")
	}
	if diff := cmp.Diff(jobstore.Job{}, job); diff != "" {
		t.Errorf("CreateJob() job mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("create zip job: generate job id: exhausted retries", err.Error()); diff != "" {
		t.Errorf("CreateJob() error mismatch (-want +got):\n%s", diff)
	}
}

func TestManagerDeleteJobRemovesCompletedArtifactAndStoreEntry(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
	}

	artifactPath := filepath.Join(fixture.config.JobsDir, "job-done.zip")
	writeTestFile(t, artifactPath, "archive contents")
	job := jobstore.Job{
		ID:        "job-done",
		Type:      jobstore.JobTypeZip,
		Status:    jobstore.StatusDone,
		Filename:  "job-done.zip",
		CreatedAt: time.Now().Add(-time.Minute),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := fixture.jobs.Add(job); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := fixture.manager.DeleteJob(job.ID); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}
	if _, ok := fixture.jobs.Get(job.ID); ok {
		t.Fatalf("Get(%q) ok = true, want false", job.ID)
	}
	assertFileAbsent(t, artifactPath)
}

func TestManagerDeleteJobRemovesFailedJobWithoutArtifact(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	job := jobstore.Job{
		ID:        "job-failed",
		Type:      jobstore.JobTypeScript,
		Status:    jobstore.StatusFailed,
		Error:     "boom",
		CreatedAt: time.Now().Add(-time.Minute),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := fixture.jobs.Add(job); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := fixture.manager.DeleteJob(job.ID); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}
	if _, ok := fixture.jobs.Get(job.ID); ok {
		t.Fatalf("Get(%q) ok = true, want false", job.ID)
	}
}

func TestManagerDeleteJobCancelsPendingJobAndRemovesStoreEntry(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	config.SourceRootDir = t.TempDir()
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
	}
	writeTestFile(t, filepath.Join(config.SourceRootDir, "reads", "sample.fastq"), "reads")

	jobStore := jobstore.NewJobs()
	manager := newManager(jobStore, config, func() string { return "pending-delete" })
	t.Cleanup(manager.Shutdown)

	for range cap(manager.sem) {
		manager.sem <- struct{}{}
	}

	job, err := manager.CreateJob("zip", []string{"reads/sample.fastq"})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		manager.jobRunsMu.Lock()
		_, running := manager.jobRuns[job.ID]
		manager.jobRunsMu.Unlock()
		stored, ok := jobStore.Get(job.ID)
		if running && ok && stored.Status == jobstore.StatusPending {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := manager.DeleteJob(job.ID); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}
	if _, ok := jobStore.Get(job.ID); ok {
		t.Fatalf("Get(%q) ok = true, want false", job.ID)
	}
}

func TestManagerDeleteJobCancelsActiveJobsWaitsForCompletionAndCleansArtifacts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  jobstore.Job
	}{
		{
			name: "processing zip job",
			job: jobstore.Job{
				ID:        "job-zip",
				Type:      jobstore.JobTypeZip,
				Status:    jobstore.StatusProcessing,
				CreatedAt: time.Now().Add(-time.Minute),
				ExpiresAt: time.Now().Add(time.Minute),
			},
		},
		{
			name: "processing tarball job",
			job: jobstore.Job{
				ID:        "job-tarball",
				Type:      jobstore.JobTypeTarball,
				Status:    jobstore.StatusProcessing,
				CreatedAt: time.Now().Add(-time.Minute),
				ExpiresAt: time.Now().Add(time.Minute),
			},
		},
		{
			name: "processing script job",
			job: jobstore.Job{
				ID:        "job-script",
				Type:      jobstore.JobTypeScript,
				Status:    jobstore.StatusProcessing,
				CreatedAt: time.Now().Add(-time.Minute),
				ExpiresAt: time.Now().Add(time.Minute),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newTestFixture(t)
			if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
			}
			if err := fixture.jobs.Add(tt.job); err != nil {
				t.Fatalf("Add() error = %v", err)
			}

			artifactPath := filepath.Join(fixture.config.JobsDir, artifactFilename(tt.job))
			writeTestFile(t, artifactPath, "partial artifact")

			cancelled := make(chan struct{}, 1)
			done := make(chan struct{})
			fixture.manager.jobRunsMu.Lock()
			fixture.manager.jobRuns[tt.job.ID] = &jobRun{
				cancel: func() {
					select {
					case cancelled <- struct{}{}:
					default:
					}
				},
				done: done,
			}
			fixture.manager.jobRunsMu.Unlock()

			errCh := make(chan error, 1)
			go func() {
				errCh <- fixture.manager.DeleteJob(tt.job.ID)
			}()

			select {
			case <-cancelled:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for job cancellation")
			}

			select {
			case err := <-errCh:
				t.Fatalf("DeleteJob() returned before run completed: %v", err)
			case <-time.After(50 * time.Millisecond):
			}

			if _, ok := fixture.jobs.Get(tt.job.ID); !ok {
				t.Fatalf("Get(%q) ok = false before run completion, want true", tt.job.ID)
			}
			if _, err := os.Stat(artifactPath); err != nil {
				t.Fatalf("Stat(%q) error before run completion = %v", artifactPath, err)
			}

			close(done)

			if err := <-errCh; err != nil {
				t.Fatalf("DeleteJob() error = %v", err)
			}
			if _, ok := fixture.jobs.Get(tt.job.ID); ok {
				t.Fatalf("Get(%q) ok = true, want false", tt.job.ID)
			}
			assertFileAbsent(t, artifactPath)
		})
	}
}

func TestManagerDeleteJobReturnsNotFoundForMissingAndExpiredJobs(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	expired := jobstore.Job{
		ID:        "expired",
		Type:      jobstore.JobTypeTarball,
		Status:    jobstore.StatusDone,
		CreatedAt: time.Now().Add(-2 * time.Minute),
		ExpiresAt: time.Now().Add(-time.Second),
	}
	if err := fixture.jobs.Add(expired); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	for _, id := range []string{"missing", expired.ID} {
		if err := fixture.manager.DeleteJob(id); !errors.Is(err, jobstore.ErrJobNotFound) {
			t.Fatalf("DeleteJob(%q) error = %v, want ErrJobNotFound", id, err)
		}
	}
}

func TestManagerDeleteJobReturnsErrorWhenArtifactCleanupFails(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
	}

	artifactDir := filepath.Join(fixture.config.JobsDir, "job-dir")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", artifactDir, err)
	}
	writeTestFile(t, filepath.Join(artifactDir, "nested.txt"), "still here")
	job := jobstore.Job{
		ID:        "job-done",
		Type:      jobstore.JobTypeZip,
		Status:    jobstore.StatusDone,
		Filename:  "job-dir",
		CreatedAt: time.Now().Add(-time.Minute),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := fixture.jobs.Add(job); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	err := fixture.manager.DeleteJob(job.ID)
	if err == nil {
		t.Fatal("DeleteJob() error = nil, want non-nil")
	}
	if _, ok := fixture.jobs.Get(job.ID); !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
}
