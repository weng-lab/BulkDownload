package core

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func ProcessJob(store *Store, job *Job) {
	store.mu.Lock()
	job.Status = StatusProcessing
	store.mu.Unlock()

	log.Printf("job %s processing started", job.ID)
	log.Printf("job %s delaying zip creation for %s", job.ID, ProcessingDelay)
	time.Sleep(ProcessingDelay)
	log.Printf("job %s delay finished, creating zip", job.ID)

	filename := job.ID + ".zip"
	outPath := filepath.Join(OutputDir, filename)

	if err := createZip(outPath, job.Files); err != nil {
		log.Printf("zip failed for job %s: %v", job.ID, err)
		store.mu.Lock()
		job.Status = StatusFailed
		job.Error = err.Error()
		store.mu.Unlock()
		return
	}

	store.mu.Lock()
	job.Status = StatusDone
	job.Filename = filename
	store.mu.Unlock()

	log.Printf("job %s complete: %s", job.ID, outPath)
}

func createZip(dest string, files []string) error {
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, path := range files {
		if err := addFileToZip(zw, path); err != nil {
			return fmt.Errorf("add %s: %w", path, err)
		}
	}

	return nil
}

func addFileToZip(zw *zip.Writer, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.Base(path)
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, src)
	return err
}
