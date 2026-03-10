package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateZipWritesFlatArchiveWithFileContents(t *testing.T) {
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

	if err := createZip(zipPath, []string{firstPath, secondPath}); err != nil {
		t.Fatalf("create zip: %v", err)
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
