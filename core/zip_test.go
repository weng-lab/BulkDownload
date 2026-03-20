package core

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateZipWritesFlatArchiveWithFileContents(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 5*time.Minute, 0)

	tempDir := t.TempDir()
	firstPath := filepath.Join(tempDir, "alpha.txt")
	secondDir := filepath.Join(tempDir, "nested")
	secondPath := filepath.Join(secondDir, "bravo.txt")
	zipPath := filepath.Join(tempDir, "result.zip")

	if err := os.WriteFile(firstPath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	if err := os.MkdirAll(secondDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("bravo contents"), 0o644); err != nil {
		t.Fatalf("write second file: %v", err)
	}

	progressUpdates := []int{}
	if err := createZip(zipPath, []string{firstPath, secondPath}, func(progress int) {
		progressUpdates = append(progressUpdates, progress)
	}); err != nil {
		t.Fatalf("create zip: %v", err)
	}
	if len(progressUpdates) == 0 {
		t.Fatalf("expected createZip to report progress")
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	if len(reader.File) != 2 {
		t.Fatalf("expected 2 files in zip, got %d", len(reader.File))
	}

	got := map[string]string{}
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zipped file %q: %v", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zipped file %q: %v", file.Name, err)
		}
		got[file.Name] = string(data)
	}

	if got["alpha.txt"] != "alpha contents" {
		t.Fatalf("expected alpha.txt contents to match, got %q", got["alpha.txt"])
	}
	if got["bravo.txt"] != "bravo contents" {
		t.Fatalf("expected bravo.txt contents to match, got %q", got["bravo.txt"])
	}
	if _, ok := got[filepath.Join("nested", "bravo.txt")]; ok {
		t.Fatalf("expected zip entries to use flat basenames only")
	}
}

func TestProcessJobCreatesZipAndMarksDone(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 5*time.Minute, 750*time.Millisecond)

	store := NewStore()
	filePath := filepath.Join(t.TempDir(), "alpha.txt")
	if err := os.WriteFile(filePath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	job, err := store.CreateJob([]string{filePath})
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}

	go ProcessJob(store, job)

	waitFor(t, 500*time.Millisecond, 25*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusProcessing
	}, "job to reach processing")

	waitFor(t, 2*time.Second, 50*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusDone && got.Filename != "" && got.Progress == 100
	}, "job to reach done")

	zipPath := filepath.Join(JobsDir, job.Filename)
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("expected zip file to exist: %v", err)
	}
}

func TestProcessJobMarksFailureForMissingFile(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 5*time.Minute, 750*time.Millisecond)

	store := NewStore()
	job, err := store.CreateJob([]string{"missing-file.txt"})
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}

	go ProcessJob(store, job)

	waitFor(t, 500*time.Millisecond, 25*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusProcessing
	}, "job to reach processing")

	waitFor(t, 2*time.Second, 50*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusFailed && got.Error != ""
	}, "job to reach failed")

	got, ok := store.Get(job.ID)
	if !ok {
		t.Fatalf("expected failed job %q to exist", job.ID)
	}
	if got.Progress != 0 {
		t.Fatalf("expected failed job progress to remain %d, got %d", 0, got.Progress)
	}

	if job.Filename != "" {
		t.Fatalf("expected failed job to have no filename, got %q", job.Filename)
	}

	zipPath := filepath.Join(JobsDir, job.ID+".zip")
	if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
		t.Fatalf("expected failed job zip to be cleaned up, got err=%v", err)
	}
}
