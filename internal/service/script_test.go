package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	jobstore "github.com/jair/bulkdownload/internal/jobs"
)

func TestManagerExecuteScriptJob(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
	}

	job, err := fixture.manager.createJob(jobstore.JobTypeScript, []string{"rna/accession.bigwig", "dna/sample.cram"}, 16)
	if err != nil {
		t.Fatalf("createJob(%q) error = %v", jobstore.JobTypeScript, err)
	}

	if err := fixture.manager.executeScriptJob(context.Background(), job.ID); err != nil {
		t.Fatalf("executeScriptJob() error = %v", err)
	}

	got, ok := fixture.jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}

	if got.Filename == "" {
		t.Fatal("processed script job filename = \"\", want non-empty")
	}

	want := jobstore.Job{
		ID:           job.ID,
		Type:         jobstore.JobTypeScript,
		Status:       jobstore.StatusDone,
		CreationTime: job.CreationTime,
		ExpiresAt:    job.ExpiresAt,
		Files:        []string{"rna/accession.bigwig", "dna/sample.cram"},
		InputSize:    16,
		OutputSize:   got.OutputSize,
		Filename:     got.Filename,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("processed script job mismatch (-want +got):\n%s", diff)
	}

	scriptPath := filepath.Join(fixture.config.JobsDir, got.Filename)
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", scriptPath, err)
	}
	if len(content) == 0 {
		t.Fatalf("ReadFile(%q) returned empty script", scriptPath)
	}
	if diff := cmp.Diff(int64(len(content)), got.OutputSize); diff != "" {
		t.Errorf("output size mismatch (-want +got):\n%s", diff)
	}
}

func TestManagerExecuteScriptJobCancellationCleansPartialFile(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
	}

	job, err := fixture.manager.createJob(jobstore.JobTypeScript, []string{"rna/accession.bigwig", "dna/sample.cram"}, 16)
	if err != nil {
		t.Fatalf("createJob(%q) error = %v", jobstore.JobTypeScript, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = fixture.manager.executeScriptJob(ctx, job.ID)
	if err == nil {
		t.Fatal("executeScriptJob() error = nil, want non-nil")
	}

	got, ok := fixture.jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if diff := cmp.Diff(jobstore.StatusFailed, got.Status); diff != "" {
		t.Errorf("job status mismatch (-want +got):\n%s", diff)
	}
	if got.Filename != "" {
		t.Fatalf("job filename = %q, want empty", got.Filename)
	}

	scriptPath := filepath.Join(fixture.config.JobsDir, job.ID+".sh")
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", scriptPath, err)
	}
}
