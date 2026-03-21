package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateZipWritesFlatArchiveWithFileContents(t *testing.T) {
	tempDir := t.TempDir()
	firstPath := filepath.Join(tempDir, "alpha.txt")
	secondDir := filepath.Join(tempDir, "nested")
	secondPath := filepath.Join(secondDir, "bravo.txt")
	zipPath := filepath.Join(tempDir, "result.zip")

	if err := os.WriteFile(firstPath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}
	if err := os.MkdirAll(secondDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("bravo contents"), 0o644); err != nil {
		t.Fatalf("WriteFile(second) error = %v", err)
	}

	progressUpdates := []int{}
	if err := createZip(zipPath, []string{firstPath, secondPath}, func(progress int) {
		progressUpdates = append(progressUpdates, progress)
	}); err != nil {
		t.Fatalf("createZip() error = %v", err)
	}
	if len(progressUpdates) == 0 {
		t.Fatal("createZip() reported no progress")
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer reader.Close()

	got := map[string]string{}
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("Open(%q) error = %v", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("ReadAll(%q) error = %v", file.Name, err)
		}
		got[file.Name] = string(data)
	}

	if got["alpha.txt"] != "alpha contents" {
		t.Fatalf("alpha.txt = %q, want %q", got["alpha.txt"], "alpha contents")
	}
	if got["bravo.txt"] != "bravo contents" {
		t.Fatalf("bravo.txt = %q, want %q", got["bravo.txt"], "bravo contents")
	}
	if _, ok := got[filepath.Join("nested", "bravo.txt")]; ok {
		t.Fatal("zip should use flat basenames only")
	}
}

func TestManagerProcessZipJobCreatesArchiveAndMarksDone(t *testing.T) {
	manager, jobs, config := testManager(t)
	filePath := filepath.Join(t.TempDir(), "alpha.txt")
	if err := os.WriteFile(filePath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	job, err := manager.CreateZipJob([]string{filePath})
	if err != nil {
		t.Fatalf("CreateZipJob() error = %v", err)
	}
	if job.Type != JobTypeZip {
		t.Fatalf("job.Type = %q, want %q", job.Type, JobTypeZip)
	}
	if got, want := time.Until(job.ExpiresAt), config.JobTTL; got < want-time.Second || got > want+time.Second {
		t.Fatalf("job TTL window = %s, want about %s", got, want)
	}

	if err := manager.ProcessZipJob(job.ID); err != nil {
		t.Fatalf("ProcessZipJob() error = %v", err)
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if got.Status != StatusDone || got.Progress != 100 || got.Filename == "" {
		t.Fatalf("processed job = %#v, want done job with filename", got)
	}

	archivePath := filepath.Join(config.JobsDir, got.Filename)
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestManagerProcessZipJobMarksFailureForMissingFile(t *testing.T) {
	manager, jobs, config := testManager(t)
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	job, err := manager.CreateZipJob([]string{"missing-file.txt"})
	if err != nil {
		t.Fatalf("CreateZipJob() error = %v", err)
	}

	err = manager.ProcessZipJob(job.ID)
	if err == nil {
		t.Fatal("ProcessZipJob() error = nil, want non-nil")
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if got.Status != StatusFailed || got.Error == "" {
		t.Fatalf("failed job = %#v, want failed status with error", got)
	}
	if got.Progress != 0 {
		t.Fatalf("failed job progress = %d, want 0", got.Progress)
	}

	archivePath := filepath.Join(config.JobsDir, job.ID+".zip")
	if _, statErr := os.Stat(archivePath); !os.IsNotExist(statErr) {
		t.Fatalf("Stat() error = %v, want not exist", statErr)
	}
}

func TestCreateTarballWritesFlatArchiveWithFileContents(t *testing.T) {
	tempDir := t.TempDir()
	firstPath := filepath.Join(tempDir, "alpha.txt")
	secondDir := filepath.Join(tempDir, "nested")
	secondPath := filepath.Join(secondDir, "bravo.txt")
	tarballPath := filepath.Join(tempDir, "result.tar.gz")

	if err := os.WriteFile(firstPath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}
	if err := os.MkdirAll(secondDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("bravo contents"), 0o644); err != nil {
		t.Fatalf("WriteFile(second) error = %v", err)
	}

	progressUpdates := []int{}
	if err := createTarball(tarballPath, []string{firstPath, secondPath}, func(progress int) {
		progressUpdates = append(progressUpdates, progress)
	}); err != nil {
		t.Fatalf("createTarball() error = %v", err)
	}
	if len(progressUpdates) == 0 {
		t.Fatal("createTarball() reported no progress")
	}

	f, err := os.Open(tarballPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
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
			t.Fatalf("Next() error = %v", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("ReadAll(%q) error = %v", header.Name, err)
		}
		got[header.Name] = string(data)
	}

	if got["alpha.txt"] != "alpha contents" {
		t.Fatalf("alpha.txt = %q, want %q", got["alpha.txt"], "alpha contents")
	}
	if got["bravo.txt"] != "bravo contents" {
		t.Fatalf("bravo.txt = %q, want %q", got["bravo.txt"], "bravo contents")
	}
	if _, ok := got[filepath.Join("nested", "bravo.txt")]; ok {
		t.Fatal("tarball should use flat basenames only")
	}
}

func TestManagerProcessTarballJobCreatesArchiveAndMarksDone(t *testing.T) {
	manager, jobs, config := testManager(t)
	filePath := filepath.Join(t.TempDir(), "alpha.txt")
	if err := os.WriteFile(filePath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	job, err := manager.CreateTarballJob([]string{filePath})
	if err != nil {
		t.Fatalf("CreateTarballJob() error = %v", err)
	}

	if err := manager.ProcessTarballJob(job.ID); err != nil {
		t.Fatalf("ProcessTarballJob() error = %v", err)
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if got.Status != StatusDone || got.Progress != 100 || got.Filename == "" {
		t.Fatalf("processed job = %#v, want done job with filename", got)
	}

	archivePath := filepath.Join(config.JobsDir, got.Filename)
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestManagerProcessTarballJobMarksFailureForMissingFile(t *testing.T) {
	manager, jobs, config := testManager(t)
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	job, err := manager.CreateTarballJob([]string{"missing-file.txt"})
	if err != nil {
		t.Fatalf("CreateTarballJob() error = %v", err)
	}

	err = manager.ProcessTarballJob(job.ID)
	if err == nil {
		t.Fatal("ProcessTarballJob() error = nil, want non-nil")
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if got.Status != StatusFailed || got.Error == "" {
		t.Fatalf("failed job = %#v, want failed status with error", got)
	}
	if got.Progress != 0 {
		t.Fatalf("failed job progress = %d, want 0", got.Progress)
	}

	archivePath := filepath.Join(config.JobsDir, job.ID+".tar.gz")
	if _, statErr := os.Stat(archivePath); !os.IsNotExist(statErr) {
		t.Fatalf("Stat() error = %v, want not exist", statErr)
	}
}
