package core

import (
	"strings"
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

func TestStoreCreateJobUsesUniqueWordPairID(t *testing.T) {
	store := NewStore()
	store.Set(&Job{ID: "genome-atac", Status: StatusPending, ExpiresAt: time.Now().Add(time.Minute)})

	job, err := store.CreateJob([]string{"file.txt"})
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}

	if job == nil {
		t.Fatal("expected CreateJob to return a job")
	}
	if _, ok := store.Get(job.ID); !ok {
		t.Fatalf("expected job %q to be stored", job.ID)
	}
	if job.ID == "genome-atac" {
		t.Fatalf("expected CreateJob to avoid reusing an existing id")
	}

	parts := strings.Split(job.ID, "-")
	if len(parts) != 2 {
		t.Fatalf("expected id %q to contain two words", job.ID)
	}
	if !containsWord(jobIDWords, parts[0]) {
		t.Fatalf("expected first word %q to come from the job id list", parts[0])
	}
	if !containsWord(jobIDWords, parts[1]) {
		t.Fatalf("expected second word %q to come from the job id list", parts[1])
	}
	if parts[0] == parts[1] {
		t.Fatalf("expected id %q to use two distinct words", job.ID)
	}
	if len(job.Files) != 1 || job.Files[0] != "file.txt" {
		t.Fatalf("expected files to be preserved, got %#v", job.Files)
	}
}

func TestStoreCreateJobRetriesUntilUnusedID(t *testing.T) {
	store := NewStore()
	store.Set(&Job{ID: "allele-atac", Status: StatusPending, ExpiresAt: time.Now().Add(time.Minute)})

	originalGenerateJobID := generateJobID
	defer func() {
		generateJobID = originalGenerateJobID
	}()

	ids := []string{"allele-atac", "codon-bam"}
	generateJobID = func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	}

	job, err := store.CreateJob([]string{"file.txt"})
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	if job.ID != "codon-bam" {
		t.Fatalf("expected CreateJob to retry into unused id, got %q", job.ID)
	}
}

func TestStoreCreateJobReturnsErrorAfterTimeout(t *testing.T) {
	store := NewStore()
	store.Set(&Job{ID: "allele-atac", Status: StatusPending, ExpiresAt: time.Now().Add(time.Minute)})

	originalGenerateJobID := generateJobID
	originalTimeout := jobIDGenerationTimeout
	defer func() {
		generateJobID = originalGenerateJobID
		jobIDGenerationTimeout = originalTimeout
	}()

	generateJobID = func() string {
		return "allele-atac"
	}
	jobIDGenerationTimeout = 10 * time.Millisecond

	job, err := store.CreateJob([]string{"file.txt"})
	if err == nil {
		t.Fatal("expected CreateJob to return an error after timing out")
	}
	if job != nil {
		t.Fatalf("expected no job on timeout, got %#v", job)
	}
	if err.Error() != "generate job id: timed out finding unique id" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func containsWord(words []string, want string) bool {
	for _, word := range words {
		if word == want {
			return true
		}
	}

	return false
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
