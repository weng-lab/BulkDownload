package core

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateTarballWritesFlatArchiveWithFileContents(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 5*time.Minute, 0)

	tempDir := t.TempDir()
	firstPath := filepath.Join(tempDir, "alpha.txt")
	secondDir := filepath.Join(tempDir, "nested")
	secondPath := filepath.Join(secondDir, "bravo.txt")
	tarballPath := filepath.Join(tempDir, "result.tar.gz")

	if err := os.WriteFile(firstPath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	if err := os.MkdirAll(secondDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("bravo contents"), 0o644); err != nil {
		t.Fatalf("write second file: %v", err)
	}

	if err := createTarball(tarballPath, []string{firstPath, secondPath}); err != nil {
		t.Fatalf("create tarball: %v", err)
	}

	f, err := os.Open(tarballPath)
	if err != nil {
		t.Fatalf("open tarball: %v", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("open gzip stream: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	got := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar entry: %v", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read tar entry %q: %v", header.Name, err)
		}
		got[header.Name] = string(data)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 files in tarball, got %d", len(got))
	}
	if got["alpha.txt"] != "alpha contents" {
		t.Fatalf("expected alpha.txt contents to match, got %q", got["alpha.txt"])
	}
	if got["bravo.txt"] != "bravo contents" {
		t.Fatalf("expected bravo.txt contents to match, got %q", got["bravo.txt"])
	}
	if _, ok := got[filepath.Join("nested", "bravo.txt")]; ok {
		t.Fatalf("expected tarball entries to use flat basenames only")
	}
}

func TestProcessTarballJobCreatesTarballAndMarksDone(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 5*time.Minute, 750*time.Millisecond)

	store := NewStore()
	filePath := filepath.Join(t.TempDir(), "alpha.txt")
	if err := os.WriteFile(filePath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	job := NewJob([]string{filePath})
	store.Set(job)

	go ProcessTarballJob(store, job)

	waitFor(t, 500*time.Millisecond, 25*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusProcessing
	}, "tarball job to reach processing")

	waitFor(t, 2*time.Second, 50*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusDone && got.Filename != ""
	}, "tarball job to reach done")

	tarballPath := filepath.Join(JobsDir, job.Filename)
	if _, err := os.Stat(tarballPath); err != nil {
		t.Fatalf("expected tarball file to exist: %v", err)
	}
}

func TestProcessTarballJobMarksFailureForMissingFile(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 5*time.Minute, 750*time.Millisecond)

	store := NewStore()
	job := NewJob([]string{"missing-file.txt"})
	store.Set(job)

	go ProcessTarballJob(store, job)

	waitFor(t, 500*time.Millisecond, 25*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusProcessing
	}, "tarball job to reach processing")

	waitFor(t, 2*time.Second, 50*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusFailed && got.Error != ""
	}, "tarball job to reach failed")

	if job.Filename != "" {
		t.Fatalf("expected failed job to have no filename, got %q", job.Filename)
	}

	tarballPath := filepath.Join(JobsDir, job.ID+".tar.gz")
	if _, err := os.Stat(tarballPath); !os.IsNotExist(err) {
		t.Fatalf("expected failed job tarball to be cleaned up, got err=%v", err)
	}
}
