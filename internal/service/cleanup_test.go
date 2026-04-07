package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	jobstore "github.com/jair/bulkdownload/internal/jobs"
)

func TestSweepExpired(t *testing.T) {
	t.Parallel()

	now := time.Unix(100, 0)

	tests := []struct {
		name                string
		jobs                []jobstore.Job
		files               map[string]string
		wantRemainingJobIDs []string
		wantRemovedFiles    []string
		wantKeptFiles       []string
	}{
		{
			name: "removes expired download with generated file",
			jobs: []jobstore.Job{
				{ID: "expired", Type: jobstore.JobTypeZip, Status: jobstore.StatusDone, Filename: "expired.zip", ExpiresAt: now.Add(-time.Second)},
				{ID: "active", Type: jobstore.JobTypeZip, Status: jobstore.StatusDone, Filename: "active.zip", ExpiresAt: now.Add(time.Second)},
			},
			files: map[string]string{
				"expired.zip": "zip bytes",
				"active.zip":  "still here",
			},
			wantRemainingJobIDs: []string{"active"},
			wantRemovedFiles:    []string{"expired.zip"},
			wantKeptFiles:       []string{"active.zip"},
		},
		{
			name: "removes expired job when output file is already gone",
			jobs: []jobstore.Job{
				{ID: "expired", Type: jobstore.JobTypeScript, Status: jobstore.StatusDone, Filename: "missing.sh", ExpiresAt: now.Add(-time.Second)},
			},
			wantRemovedFiles: []string{"missing.sh"},
		},
		{
			name: "removes expired job without output filename",
			jobs: []jobstore.Job{
				{ID: "expired", Type: jobstore.JobTypeTarball, Status: jobstore.StatusFailed, ExpiresAt: now.Add(-time.Second)},
				{ID: "active", Type: jobstore.JobTypeTarball, Status: jobstore.StatusPending, ExpiresAt: now.Add(time.Second)},
			},
			wantRemainingJobIDs: []string{"active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jobsDir := t.TempDir()
			store := jobstore.NewJobs()
			for _, job := range tt.jobs {
				if err := store.Add(job); err != nil {
					t.Fatalf("Add(%q) error = %v", job.ID, err)
				}
			}

			for name, content := range tt.files {
				filePath := filepath.Join(jobsDir, name)
				if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
					t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(filePath), err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
					t.Fatalf("WriteFile(%q) error = %v", filePath, err)
				}
			}

			sweepExpired(store, jobsDir, now)

			var gotRemainingJobIDs []string
			for _, job := range tt.jobs {
				if _, ok := store.Get(job.ID); ok {
					gotRemainingJobIDs = append(gotRemainingJobIDs, job.ID)
				}
			}
			if diff := cmp.Diff(tt.wantRemainingJobIDs, gotRemainingJobIDs); diff != "" {
				t.Errorf("remaining jobs mismatch (-want +got):\n%s", diff)
			}

			for _, name := range tt.wantRemovedFiles {
				filePath := filepath.Join(jobsDir, name)
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Errorf("Stat(%q) error = %v, want not exist", filePath, err)
				}
			}

			for _, name := range tt.wantKeptFiles {
				filePath := filepath.Join(jobsDir, name)
				if _, err := os.Stat(filePath); err != nil {
					t.Errorf("Stat(%q) error = %v", filePath, err)
				}
			}
		})
	}
}

func TestStartCleanup_SweepsOnTick(t *testing.T) {
	t.Parallel()

	jobsDir := t.TempDir()
	store := jobstore.NewJobs()
	now := time.Now()
	job := jobstore.Job{
		ID:        "expired",
		Type:      jobstore.JobTypeZip,
		Status:    jobstore.StatusDone,
		Filename:  "expired.zip",
		ExpiresAt: now.Add(-time.Second),
	}
	if err := store.Add(job); err != nil {
		t.Fatalf("Add(%q) error = %v", job.ID, err)
	}

	archivePath := filepath.Join(jobsDir, job.Filename)
	if err := os.WriteFile(archivePath, []byte("zip bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", archivePath, err)
	}

	stopCleanup := StartCleanup(store, jobsDir, 10*time.Millisecond)
	t.Cleanup(stopCleanup)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, ok := store.Get(job.ID); !ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if _, ok := store.Get(job.ID); ok {
		t.Fatal("Get(expired) ok = true, want false after cleanup tick")
	}
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", archivePath, err)
	}
}
