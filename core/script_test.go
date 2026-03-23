package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestManagerExecuteScriptJob(t *testing.T) {
	t.Parallel()

	fixture := newTestFixture(t)
	if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
	}

	job, err := fixture.manager.createJob(JobTypeScript, []string{"rna/accession.bigwig", "dna/sample.cram"})
	if err != nil {
		t.Fatalf("createJob(%q) error = %v", JobTypeScript, err)
	}

	if err := fixture.manager.executeScriptJob(job.ID); err != nil {
		t.Fatalf("executeScriptJob() error = %v", err)
	}

	got, ok := fixture.jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}

	if got.Filename == "" {
		t.Fatal("processed script job filename = \"\", want non-empty")
	}

	want := Job{
		ID:        job.ID,
		Type:      JobTypeScript,
		Status:    StatusDone,
		ExpiresAt: job.ExpiresAt,
		Files:     []string{"rna/accession.bigwig", "dna/sample.cram"},
		Filename:  got.Filename,
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
}
