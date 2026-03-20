package core

import (
	"archive/zip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func ProcessJob(store *Store, job *Job) {
	store.SetStatus(job.ID, StatusProcessing)

	log.Printf("job %s processing started", job.ID)
	time.Sleep(ProcessingDelay)

	filename := job.ID + ".zip"
	outPath := filepath.Join(JobsDir, filename)

	if err := createZip(outPath, job.Files, func(progress int) {
		store.SetProgress(job.ID, progress)
	}); err != nil {
		log.Printf("zip failed for job %s: %v", job.ID, err)
		if removeErr := os.Remove(outPath); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("cleanup failed zip for job %s: %v", job.ID, removeErr)
		}
		store.SetFailed(job.ID, err)
		return
	}

	store.SetDone(job.ID, filename)

	log.Printf("job %s complete: %s", job.ID, outPath)
}

func createZip(dest string, files []string, onProgress func(int)) error {
	total, err := totalFileSize(files)
	if err != nil {
		return fmt.Errorf("calculate zip progress: %w", err)
	}
	reporter := newProgressReporter(total, onProgress)

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, path := range files {
		if err := addFileToZip(zw, path, reporter); err != nil {
			return fmt.Errorf("add %s: %w", path, err)
		}
	}

	return nil
}

func addFileToZip(zw *zip.Writer, path string, reporter *progressReporter) error {
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

	return copyWithProgress(w, src, reporter)
}
