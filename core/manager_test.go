package core

import (
	"testing"
	"time"
)

func TestManagerCreateZipJobRetriesDuplicateID(t *testing.T) {
	config := testConfig(t)
	jobs := NewJobs()
	if err := jobs.Add(&Job{ID: "allele-atac", Type: JobTypeZip, ExpiresAt: time.Unix(100, 0)}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	ids := []string{"allele-atac", "codon-bam"}
	manager := newManager(jobs, config, func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	})

	job, err := manager.CreateZipJob([]string{"file.txt"})
	if err != nil {
		t.Fatalf("CreateZipJob() error = %v", err)
	}
	if job.ID != "codon-bam" {
		t.Fatalf("job.ID = %q, want %q", job.ID, "codon-bam")
	}
}

func TestManagerCreateZipJobExhaustsRetries(t *testing.T) {
	config := testConfig(t)
	jobs := NewJobs()
	if err := jobs.Add(&Job{ID: "allele-atac", Type: JobTypeZip, ExpiresAt: time.Unix(100, 0)}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	manager := newManager(jobs, config, func() string { return "allele-atac" })

	job, err := manager.CreateZipJob([]string{"file.txt"})
	if err == nil {
		t.Fatal("CreateZipJob() error = nil, want non-nil")
	}
	if job != nil {
		t.Fatalf("CreateZipJob() job = %#v, want nil", job)
	}
}
